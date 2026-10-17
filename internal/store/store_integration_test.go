//go:build integration

package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var testStore *Store

// TestMain creates an isolated database per run, migrates it and removes it afterwards.
func TestMain(m *testing.M) {
	base := os.Getenv("CROSSWATC_TEST_DATABASE_URL")
	if base == "" {
		fmt.Println("CROSSWATC_TEST_DATABASE_URL not set; skipping integration tests")
		os.Exit(0)
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	must(err)
	name := "cw_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	must(err)

	u, err := url.Parse(base)
	must(err)
	u.Path = "/" + name
	must(Migrate(ctx, u.String()))
	testStore, err = Open(ctx, u.String())
	must(err)

	code := m.Run()

	testStore.Close()
	_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
	_ = admin.Close(ctx)
	os.Exit(code)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func newResource(name, owner, env string) TrackedResource {
	return TrackedResource{
		Name: name, Kind: "PostgreSQLInstance", Provider: "azure", Environment: env, Owner: owner,
		ClaimNamespace: "team-" + owner, ClaimName: name, ClaimKind: "PostgreSQLInstance",
		ClaimAPIVersion: "database.crosswatc.io/v1alpha1", Repository: "org/argocd", GitPath: "claims/" + name + ".yaml",
		CorrelationID: "crosswatc-req-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:26],
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	u := testStore.pool.Config().ConnConfig.ConnString()
	if err := Migrate(context.Background(), u); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertResourceByClaimPreservesIdentity(t *testing.T) {
	ctx := context.Background()
	r1, err := testStore.UpsertResourceByClaim(ctx, newResource("pg-ident", "payments", "dev"))
	if err != nil {
		t.Fatal(err)
	}
	upd := newResource("pg-ident", "payments", "dev")
	upd.Region = "brazilsouth"
	r2, err := testStore.UpsertResourceByClaim(ctx, upd)
	if err != nil {
		t.Fatal(err)
	}
	if r1.ID != r2.ID || r1.CorrelationID != r2.CorrelationID || !r1.CreatedAt.Equal(r2.CreatedAt) {
		t.Fatal("identity fields must be preserved on upsert")
	}
	if r2.Region != "brazilsouth" || r2.Version != r1.Version+1 {
		t.Fatalf("unexpected update: %+v", r2)
	}
}

func TestUpdateResourceStateOptimisticLock(t *testing.T) {
	ctx := context.Background()
	r, err := testStore.UpsertResourceByClaim(ctx, newResource("pg-lock", "payments", "dev"))
	if err != nil {
		t.Fatal(err)
	}
	v, err := testStore.UpdateResourceState(ctx, r.ID, r.Version, "OPERATIONAL", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.UpdateResourceState(ctx, r.ID, r.Version, "DEGRADED", false); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version must conflict, got %v", err)
	}
	if _, err := testStore.UpdateResourceState(ctx, r.ID, v, "DEGRADED", false); err != nil {
		t.Fatal(err)
	}
	got, err := testStore.GetResource(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ResourceState != "DEGRADED" || !got.EverOperational {
		t.Fatalf("ever_operational must be sticky: %+v", got)
	}
}

func TestGetResourceNotFound(t *testing.T) {
	if _, err := testStore.GetResource(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestListResourcesFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	for i := range 5 {
		if _, err := testStore.UpsertResourceByClaim(ctx, newResource(fmt.Sprintf("list-a-%d", i), "alpha", "dev")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := testStore.UpsertResourceByClaim(ctx, newResource("list-b-0", "beta", "prd")); err != nil {
		t.Fatal(err)
	}

	var names []string
	cur := ""
	for {
		page, next, err := testStore.ListResources(ctx, ResourceFilter{Owners: []string{"alpha"}, NamePrefix: "list-"}, Page{Limit: 2, Cursor: cur})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range page {
			names = append(names, r.Name)
		}
		if next == "" {
			break
		}
		cur = next
	}
	if strings.Join(names, ",") != "list-a-0,list-a-1,list-a-2,list-a-3,list-a-4" {
		t.Fatalf("got %v", names)
	}

	got, _, err := testStore.ListResources(ctx, ResourceFilter{Environment: "prd", NamePrefix: "list-"}, Page{})
	if err != nil || len(got) != 1 || got[0].Owner != "beta" {
		t.Fatalf("env filter: %v %v", got, err)
	}
	got, _, err = testStore.ListResources(ctx, ResourceFilter{Owners: []string{}}, Page{})
	if err != nil || len(got) != 0 {
		t.Fatalf("empty owner set must return nothing: %v %v", got, err)
	}
	if _, _, err := testStore.ListResources(ctx, ResourceFilter{}, Page{Cursor: "!!"}); !errors.Is(err, ErrBadCursor) {
		t.Fatalf("got %v", err)
	}
}

func TestProjectionAndTopology(t *testing.T) {
	ctx := context.Background()
	r, err := testStore.UpsertResourceByClaim(ctx, newResource("pg-proj", "payments", "dev"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	p := Projection{ResourceID: r.ID, Stages: []byte(`[{"stage":"REQUEST","status":"SKIPPED"}]`), LastReconcileAt: &now}
	if err := testStore.UpsertProjection(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := testStore.UpsertProjection(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := testStore.GetProjection(ctx, r.ID)
	if err != nil || got.Version != 1 || !strings.Contains(string(got.Stages), "SKIPPED") || got.DesiredState != nil {
		t.Fatalf("projection: %+v %v", got, err)
	}

	if err := testStore.UpsertTopology(ctx, Topology{ResourceID: r.ID, Nodes: []byte(`[]`), Edges: []byte(`[]`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.GetTopology(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.GetTopology(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestEventsDedupeAndPagination(t *testing.T) {
	ctx := context.Background()
	r, err := testStore.UpsertResourceByClaim(ctx, newResource("pg-events", "payments", "dev"))
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Truncate(time.Microsecond)
	for i := range 5 {
		ok, err := testStore.InsertEvent(ctx, Event{DedupeKey: fmt.Sprintf("k8s:%d", i), ResourceID: &r.ID, Source: "crossplane",
			Type: "INFRA.RECONCILE.READY", Level: "info", CreatedAt: base.Add(time.Duration(i) * time.Second)})
		if err != nil || !ok {
			t.Fatal(err, ok)
		}
	}
	ok, err := testStore.InsertEvent(ctx, Event{DedupeKey: "k8s:0", ResourceID: &r.ID, Source: "x", Type: "x", Level: "info"})
	if err != nil || ok {
		t.Fatalf("duplicate must be ignored: %v %v", ok, err)
	}

	page1, next, err := testStore.ListEvents(ctx, r.ID, Page{Limit: 3})
	if err != nil || len(page1) != 3 || next == "" || page1[0].DedupeKey != "k8s:4" {
		t.Fatalf("page1: %v %q %v", page1, next, err)
	}
	page2, next, err := testStore.ListEvents(ctx, r.ID, Page{Limit: 3, Cursor: next})
	if err != nil || len(page2) != 2 || next != "" || page2[1].DedupeKey != "k8s:0" {
		t.Fatalf("page2: %v %q %v", page2, next, err)
	}
}

func TestOneOpenRevisionPerResource(t *testing.T) {
	ctx := context.Background()
	r, err := testStore.UpsertResourceByClaim(ctx, newResource("pg-rev", "payments", "dev"))
	if err != nil {
		t.Fatal(err)
	}
	rev1, err := testStore.InsertRevision(ctx, Revision{ResourceID: r.ID, Rev: 1, Kind: "CREATE", State: "VALIDATING"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.InsertRevision(ctx, Revision{ResourceID: r.ID, Rev: 2, Kind: "UPDATE", State: "CREATED"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second open revision must conflict, got %v", err)
	}
	now := time.Now()
	rev1.State, rev1.CompletedAt = "SUCCEEDED", &now
	if err := testStore.UpdateRevision(ctx, rev1); err != nil {
		t.Fatal(err)
	}
	if _, err := testStore.InsertRevision(ctx, Revision{ResourceID: r.ID, Rev: 2, Kind: "UPDATE", State: "CREATED"}); err != nil {
		t.Fatal(err)
	}
	revs, err := testStore.ListRevisions(ctx, r.ID)
	if err != nil || len(revs) != 2 || revs[0].Rev != 2 {
		t.Fatalf("list: %v %v", revs, err)
	}

	op, err := testStore.InsertOperation(ctx, Operation{ResourceID: r.ID, RevisionID: &rev1.ID, CorrelationID: r.CorrelationID,
		Type: "CREATE", Status: "SUCCEEDED", RequestedBy: "agnaldo"})
	if err != nil {
		t.Fatal(err)
	}
	ops, err := testStore.ListOperations(ctx, r.ID)
	if err != nil || len(ops) != 1 || ops[0].ID != op.ID {
		t.Fatalf("ops: %v %v", ops, err)
	}
}

func TestAppendOnlyGrants(t *testing.T) {
	ctx := context.Background()
	var canUpdate bool
	err := testStore.pool.QueryRow(ctx, `SELECT has_table_privilege('crosswatc_app', 'tracker_event', 'UPDATE')`).Scan(&canUpdate)
	if err != nil || canUpdate {
		t.Fatalf("crosswatc_app must not UPDATE tracker_event: %v %v", canUpdate, err)
	}
}

func TestCheckpoint(t *testing.T) {
	ctx := context.Background()
	if err := testStore.SaveCheckpoint(ctx, "github", "etag-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := testStore.SaveCheckpoint(ctx, "github", "", errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	var cur, lastErr string
	var success *time.Time
	if err := testStore.pool.QueryRow(ctx, `SELECT cursor, last_error, last_success_at FROM ingest_checkpoint WHERE source='github'`).
		Scan(&cur, &lastErr, &success); err != nil {
		t.Fatal(err)
	}
	if cur != "etag-1" || lastErr != "boom" || success == nil {
		t.Fatalf("got %q %q %v", cur, lastErr, success)
	}
}
