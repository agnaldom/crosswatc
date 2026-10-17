package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/agnaldom/crosswatc/internal/adapters/backstage"
	"github.com/agnaldom/crosswatc/internal/adapters/github"
	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/topology"
)

// Reconciler runs one full resync per RunOnce call.
type Reconciler struct {
	Discovery    Discovery
	PullRequests PullRequests // optional
	Argo         Argo         // optional
	Backstage    Backstage    // optional
	Store        Store
	Log          *slog.Logger
	StuckTimeout time.Duration
	Now          func() time.Time
}

// Result summarizes a run.
type Result struct {
	Discovered int
	Reconciled int
	Vanished   int
	Errors     int
}

// RunOnce discovers every managed Claim and reconciles it. Resources tracked in the read model whose
// Claim was not discovered are reconciled as absent, but only when discovery listed everything cleanly.
func (r *Reconciler) RunOnce(ctx context.Context) (Result, error) {
	var res Result
	xrds, err := r.Discovery.DiscoverXRDs(ctx)
	if err != nil {
		r.checkpoint(ctx, "k8s", err)
		return res, err
	}
	mrs, err := r.Discovery.DiscoverMRDs(ctx)
	if err != nil {
		r.checkpoint(ctx, "k8s", err)
		return res, err
	}
	discovered, listErr := r.Discovery.DiscoverResources(ctx, xrds, mrs)
	r.checkpoint(ctx, "k8s", listErr)
	res.Discovered = len(discovered)

	seen := map[string]bool{}
	for _, d := range discovered {
		seen[claimKey(d.Claim.Namespace, d.Claim.Name, d.Claim.Kind)] = true
		if err := r.reconcile(ctx, d); err != nil {
			res.Errors++
			r.log().Error("reconcile failed", "claim", d.Claim.ID(), "err", err)
			continue
		}
		res.Reconciled++
	}
	if listErr != nil {
		return res, listErr
	}

	vanished, err := r.reconcileVanished(ctx, seen)
	res.Vanished = vanished
	return res, err
}

func (r *Reconciler) reconcileVanished(ctx context.Context, seen map[string]bool) (int, error) {
	n, cursor := 0, ""
	for {
		page, next, err := r.Store.ListResources(ctx, store.ResourceFilter{}, store.Page{Limit: 200, Cursor: cursor})
		if err != nil {
			return n, err
		}
		for _, tr := range page {
			if seen[claimKey(tr.ClaimNamespace, tr.ClaimName, tr.ClaimKind)] || tr.ResourceState == string(domain.ResourceDeleted) {
				continue
			}
			if err := r.reconcileAbsent(ctx, tr); err != nil {
				r.log().Error("reconcile vanished failed", "resource", tr.ID, "err", err)
				continue
			}
			n++
		}
		if next == "" {
			return n, nil
		}
		cursor = next
	}
}

func claimKey(ns, name, kind string) string { return kind + "/" + ns + "/" + name }

func (r *Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

func (r *Reconciler) log() *slog.Logger {
	if r.Log != nil {
		return r.Log
	}
	return slog.Default()
}

func (r *Reconciler) checkpoint(ctx context.Context, source string, err error) {
	if cpErr := r.Store.SaveCheckpoint(ctx, source, "", err); cpErr != nil {
		r.log().Warn("save checkpoint failed", "source", source, "err", cpErr)
	}
}

// reconcile handles one discovered resource (spec §36 steps).
func (r *Reconciler) reconcile(ctx context.Context, d kube.DiscoveredResource) error {
	tr, err := r.Store.UpsertResourceByClaim(ctx, toTrackedResource(d))
	if err != nil {
		return fmt.Errorf("upsert resource: %w", err)
	}
	created := tr.Version == 0
	if created {
		r.event(ctx, tr, nil, "claim:"+d.Claim.UID+tr.ID.String(), "kubernetes", "INFRA.CLAIM.CREATED", "info",
			fmt.Sprintf("Recurso %s descoberto", tr.Name))
	}
	if len(d.MetadataErrors) > 0 {
		r.event(ctx, tr, nil, fmt.Sprintf("meta:%s:%d", tr.ID, d.Claim.Generation), "kubernetes", "INFRA.CLAIM.INVALID_METADATA", "warning",
			fmt.Sprintf("Claim com metadata inválida: %v", d.MetadataErrors))
	}

	revs, err := r.Store.ListRevisions(ctx, tr.ID)
	if err != nil {
		return fmt.Errorf("list revisions: %w", err)
	}
	open := openRevision(revs)
	if len(revs) == 0 {
		rev, err := r.startCreateRevision(ctx, tr)
		if err != nil {
			return err
		}
		open = &rev
		revs = []store.Revision{rev}
	}

	sig := r.signals(ctx, tr, d, open, revs)

	if open != nil {
		if err := r.advanceRevision(ctx, tr, open, sig); err != nil {
			return err
		}
		if domain.RevisionState(open.State).Terminal() {
			sig.OpenRevision = nil
			sig.CreateFailed = sig.CreateFailed || (open.Kind == string(domain.RevisionCreate) && open.State == string(domain.RevisionFailed))
		} else {
			sig.OpenRevision = &lifecycle.OpenRevision{Kind: domain.RevisionKind(open.Kind), State: domain.RevisionState(open.State)}
		}
	}

	if err := r.applyResourceState(ctx, &tr, sig); err != nil {
		return err
	}
	if err := r.Store.UpsertProjection(ctx, projection(tr, d, sig, r.now())); err != nil {
		return fmt.Errorf("upsert projection: %w", err)
	}
	g := topology.Build(d)
	nodes, _ := json.Marshal(g.Nodes)
	edges, _ := json.Marshal(g.Edges)
	if err := r.Store.UpsertTopology(ctx, store.Topology{ResourceID: tr.ID, Nodes: nodes, Edges: edges}); err != nil {
		return fmt.Errorf("upsert topology: %w", err)
	}
	return nil
}

// reconcileAbsent handles a tracked resource whose Claim no longer exists.
func (r *Reconciler) reconcileAbsent(ctx context.Context, tr store.TrackedResource) error {
	revs, err := r.Store.ListRevisions(ctx, tr.ID)
	if err != nil {
		return err
	}
	sig := lifecycle.Signals{EverOperational: tr.EverOperational}
	for _, rv := range revs {
		if rv.Kind == string(domain.RevisionDelete) && rv.State == string(domain.RevisionSucceeded) {
			sig.DeleteSucceeded = true
		}
	}
	if o := openRevision(revs); o != nil {
		sig.OpenRevision = &lifecycle.OpenRevision{Kind: domain.RevisionKind(o.Kind), State: domain.RevisionState(o.State)}
	}
	r.event(ctx, tr, nil, fmt.Sprintf("claim-gone:%s:%d", tr.ID, tr.Version), "kubernetes", "INFRA.CLAIM.DELETED", "warning",
		fmt.Sprintf("Claim %s/%s não encontrado no cluster", tr.ClaimNamespace, tr.ClaimName))
	return r.applyResourceState(ctx, &tr, sig)
}

func openRevision(revs []store.Revision) *store.Revision {
	for i := range revs {
		if !domain.RevisionState(revs[i].State).Terminal() {
			return &revs[i]
		}
	}
	return nil
}

func (r *Reconciler) startCreateRevision(ctx context.Context, tr store.TrackedResource) (store.Revision, error) {
	now := r.now()
	rev, err := r.Store.InsertRevision(ctx, store.Revision{
		ResourceID: tr.ID, Rev: 1, Kind: string(domain.RevisionCreate), State: string(domain.RevisionCreated), StartedAt: &now,
	})
	if err != nil {
		return store.Revision{}, fmt.Errorf("insert create revision: %w", err)
	}
	requestedBy := tr.RequestedBy
	if requestedBy == "" {
		requestedBy = "unknown"
	}
	if _, err := r.Store.InsertOperation(ctx, store.Operation{
		ResourceID: tr.ID, RevisionID: &rev.ID, CorrelationID: tr.CorrelationID, Type: string(domain.OperationCreate),
		Status: string(domain.OperationQueued), RequestedBy: requestedBy, RequestedAt: tr.CreatedAt,
	}); err != nil && !errors.Is(err, store.ErrConflict) {
		return store.Revision{}, fmt.Errorf("insert create operation: %w", err)
	}
	r.event(ctx, tr, &rev, "request:"+tr.CorrelationID, "crosswatc", "REQUEST.CREATED", "info", "Solicitação registrada")
	return rev, nil
}

// signals collects external state. Adapter failures degrade to empty signals and are logged.
func (r *Reconciler) signals(ctx context.Context, tr store.TrackedResource, d kube.DiscoveredResource, open *store.Revision, revs []store.Revision) lifecycle.Signals {
	sig := lifecycle.Signals{
		InvalidMetadata: len(d.MetadataErrors) > 0,
		Claim:           objectSignal(d.Claim),
		SecretAvailable: d.SecretAvailable,
		EverOperational: tr.EverOperational,
	}
	if d.XR != nil {
		sig.XR = objectSignal(*d.XR)
	}
	for _, m := range d.MRs {
		sig.MRs = append(sig.MRs, objectSignal(m))
	}
	for range d.Missing {
		sig.MRs = append(sig.MRs, lifecycle.ObjectSignal{}) // missing composed resource keeps the infra not-ready
	}
	for _, rv := range revs {
		if rv.Kind == string(domain.RevisionCreate) && rv.State == string(domain.RevisionFailed) {
			sig.CreateFailed = true
		}
	}

	if r.Backstage != nil && d.Metadata.BackstageTaskID != "" {
		bs, err := r.Backstage.Task(ctx, d.Metadata.BackstageTaskID)
		switch {
		case errors.Is(err, backstage.ErrTaskNotFound):
			r.log().Warn("backstage task not found", "resource", tr.ID, "task", d.Metadata.BackstageTaskID)
		case err != nil:
			r.log().Warn("backstage lookup failed", "resource", tr.ID, "err", err)
			sig.Backstage = lifecycle.BackstageSignal{TaskID: d.Metadata.BackstageTaskID}
		default:
			sig.Backstage = bs
		}
	}

	if r.PullRequests != nil && tr.Repository != "" {
		l := github.Lookup{Repository: tr.Repository, Path: tr.GitPath, CorrelationID: tr.CorrelationID}
		if open != nil && open.PRNumber != nil {
			l.PRNumber = *open.PRNumber
		}
		pr, err := r.PullRequests.PullRequest(ctx, l)
		if err != nil {
			r.log().Warn("github lookup failed", "resource", tr.ID, "err", err)
		}
		sig.PR = pr
	}

	if r.Argo != nil {
		if ref, ok := r.Argo.ResolveApp(d.Claim.Labels, d.Claim.Annotations); ok {
			as, err := r.Argo.Status(ctx, ref)
			if err != nil {
				r.log().Warn("argo lookup failed", "resource", tr.ID, "app", ref.String(), "err", err)
			}
			sig.Argo = as
		}
	}
	return sig
}

func objectSignal(o kube.Object) lifecycle.ObjectSignal {
	return lifecycle.ObjectSignal{
		Present: true, Deleting: o.Deleting, Generation: o.Generation, ObservedGeneration: o.ObservedGeneration, Conditions: o.Conditions,
	}
}

// transitionEvents maps the new revision state to a canonical event (spec §33).
var transitionEvents = map[domain.RevisionState]struct{ typ, source, level string }{
	domain.RevisionValidating:      {"GITOPS.PR.CREATED", "github", "info"},
	domain.RevisionWaitingApproval: {"GITOPS.CHECK.SUCCEEDED", "github", "info"},
	domain.RevisionMerged:          {"GITOPS.PR.MERGED", "github", "info"},
	domain.RevisionSyncing:         {"GITOPS.SYNC.STARTED", "argocd", "info"},
	domain.RevisionReconciling:     {"GITOPS.SYNC.SUCCEEDED", "argocd", "info"},
	domain.RevisionApplying:        {"INFRA.RECONCILE.READY", "crossplane", "info"},
	domain.RevisionSucceeded:       {"INFRA.MANAGED_RESOURCE.READY", "crossplane", "info"},
	domain.RevisionRejected:        {"GITOPS.PR.CLOSED", "github", "warning"},
	domain.RevisionFailed:          {"INFRA.RECONCILE.FAILED", "crossplane", "error"},
}

func (r *Reconciler) advanceRevision(ctx context.Context, tr store.TrackedResource, rev *store.Revision, sig lifecycle.Signals) error {
	cur := domain.RevisionState(rev.State)
	res := lifecycle.EvaluateRevision(cur, sig, r.now().Sub(rev.StateChangedAt), r.StuckTimeout)

	changed := res.State != cur
	if sig.PR.Found {
		if rev.PRNumber == nil || *rev.PRNumber != sig.PR.Number {
			n := sig.PR.Number
			rev.PRNumber, rev.PRURL, changed = &n, sig.PR.URL, true
		}
		if sig.PR.MergeCommitSHA != "" && rev.CommitSHA != sig.PR.MergeCommitSHA {
			rev.CommitSHA, changed = sig.PR.MergeCommitSHA, true
		}
	}
	if res.Failure != nil {
		b, _ := json.Marshal(res.Failure)
		if string(b) != string(rev.FailureReason) {
			rev.FailureReason, changed = b, true
		}
	}
	if res.Stuck {
		r.event(ctx, tr, rev, fmt.Sprintf("stuck:%s:%s", rev.ID, rev.State), "crosswatc", "OPERATION.STUCK", "warning",
			fmt.Sprintf("Revisão %d sem progresso em %s", rev.Rev, rev.State))
	}
	if !changed {
		return nil
	}

	rev.State = string(res.State)
	var finished *time.Time
	if res.State.Terminal() {
		now := r.now()
		rev.CompletedAt, finished = &now, &now
	}
	if err := r.Store.UpdateRevision(ctx, *rev); err != nil {
		return fmt.Errorf("update revision: %w", err)
	}
	if err := r.Store.UpdateOperationsForRevision(ctx, rev.ID, string(domain.OperationStatusFor(res.State)), finished); err != nil {
		return fmt.Errorf("update operation: %w", err)
	}
	if res.State != cur {
		if ev, ok := transitionEvents[res.State]; ok {
			typ, source := ev.typ, ev.source
			if res.Failure != nil && res.Failure.TechnicalCode == "ArgoSyncFailed" {
				typ, source = "GITOPS.SYNC.FAILED", "argocd"
			}
			msg := fmt.Sprintf("Revisão %d: %s", rev.Rev, res.State)
			if res.Failure != nil {
				msg = res.Failure.UserMessage
			}
			r.event(ctx, tr, rev, fmt.Sprintf("rev:%s:%s", rev.ID, res.State), source, typ, ev.level, msg)
		}
	}
	return nil
}

func (r *Reconciler) applyResourceState(ctx context.Context, tr *store.TrackedResource, sig lifecycle.Signals) error {
	next := lifecycle.DeriveResourceState(sig)
	if string(next) == tr.ResourceState {
		return nil
	}
	v, err := r.Store.UpdateResourceState(ctx, tr.ID, tr.Version, string(next), next == domain.ResourceOperational)
	if err != nil {
		return fmt.Errorf("update resource state: %w", err)
	}
	prev := tr.ResourceState
	tr.ResourceState, tr.Version = string(next), v
	switch next {
	case domain.ResourceOperational:
		r.event(ctx, *tr, nil, fmt.Sprintf("res:%s:%s:%d", tr.ID, next, v), "crosswatc", "LIFECYCLE.OPERATIONAL", "info", "Recurso operacional")
	case domain.ResourceDegraded:
		r.event(ctx, *tr, nil, fmt.Sprintf("res:%s:%s:%d", tr.ID, next, v), "crosswatc", "LIFECYCLE.DEGRADED", "warning",
			fmt.Sprintf("Recurso degradado (antes: %s)", prev))
	}
	return nil
}

func (r *Reconciler) event(ctx context.Context, tr store.TrackedResource, rev *store.Revision, key, source, typ, level, msg string) {
	e := store.Event{
		DedupeKey: key, ResourceID: &tr.ID, CorrelationID: tr.CorrelationID, Source: source, Type: typ, Level: level,
		Message: msg, CreatedAt: r.now(),
	}
	if rev != nil {
		e.RevisionID = &rev.ID
	}
	if _, err := r.Store.InsertEvent(ctx, e); err != nil {
		r.log().Warn("insert event failed", "type", typ, "err", err)
	}
}

// toTrackedResource maps discovery output to the read model (spec §11).
func toTrackedResource(d kube.DiscoveredResource) store.TrackedResource {
	md := d.Metadata
	tr := store.TrackedResource{
		Name: d.Claim.Name, Kind: d.Claim.Kind,
		Provider: orUnknown(md.Provider), Environment: orUnknown(md.Environment), Owner: orUnknown(md.Owner),
		ClaimNamespace: d.Claim.Namespace, ClaimName: d.Claim.Name, ClaimKind: d.Claim.Kind, ClaimAPIVersion: d.Claim.APIVersion,
		Repository: md.Repository, GitPath: md.Path, CorrelationID: md.RequestID,
		Region:        str(d.Claim.Spec, "parameters", "region", "location"),
		ResourceGroup: str(d.Claim.Spec, "parameters", "resourceGroup", "resourceGroupName"),
	}
	if md.Revision > 0 {
		n := md.Revision
		tr.CurrentRevision = &n
	}
	if !correlation.Valid(tr.CorrelationID) {
		tr.CorrelationID = correlation.New() // preserved on later upserts (§16)
	}
	if md.Provider == "" && len(d.MRs) > 0 {
		if p := kube.ProviderOf(d.MRs[0].APIVersion); p != "" {
			tr.Provider = p
		}
	}
	if d.XR != nil {
		tr.CompositionName, tr.CompositionRevision = d.XR.CompositionName, d.XR.CompositionRevision
	}
	if v, ok := d.Claim.Annotations["crosswatc.io/requested-by"]; ok {
		tr.RequestedBy = v
	}
	return tr
}

// str returns the first non-empty string field of m[section] among keys.
func str(spec map[string]any, section string, keys ...string) string {
	m, _ := spec[section].(map[string]any)
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// projection builds resource_projection (spec §18, §31, §46 Overview).
func projection(tr store.TrackedResource, d kube.DiscoveredResource, sig lifecycle.Signals, now time.Time) store.Projection {
	mustJSON := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	desired := map[string]any{"repository": tr.Repository, "path": tr.GitPath, "commitSha": sig.PR.MergeCommitSHA}
	applied := map[string]any{
		"generation": d.Claim.Generation, "spec": d.Claim.Spec,
		"syncedWithGit": sig.Argo.Found && sig.Argo.SyncStatus == "Synced",
	}
	observed := map[string]any{"ready": d.Claim.Ready(), "synced": d.Claim.Synced(), "lastReconcileAt": now}
	health := map[string]any{
		"crossplane":    health(d.XR != nil && d.XR.Ready() && d.XR.Synced(), d.XR != nil),
		"argocd":        argoHealth(sig.Argo),
		"cloud":         health(sig.AllHealthy(), len(d.MRs) > 0),
		"configuration": health(sig.Argo.SyncStatus == "Synced", sig.Argo.Found),
		"secrets":       health(d.SecretAvailable, true),
	}
	return store.Projection{
		ResourceID: tr.ID, DesiredState: mustJSON(desired), AppliedState: mustJSON(applied), ObservedState: mustJSON(observed),
		DriftState: mustJSON(map[string]any{"detected": false}), Stages: mustJSON(lifecycle.ResolveStages(sig)),
		Health: mustJSON(health), LastReconcileAt: &now,
	}
}

func health(ok, known bool) string {
	switch {
	case !known:
		return "unknown"
	case ok:
		return "ok"
	default:
		return "attention"
	}
}

func argoHealth(a lifecycle.ArgoSignal) string {
	if !a.Found {
		return "unknown"
	}
	if a.SyncStatus == "Synced" && (a.Health == "" || a.Health == "Healthy") {
		return "ok"
	}
	return "attention"
}
