// Package reconciler is the worker State Reconciler (spec §36): discovery → signals → state machine → read model.
package reconciler

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/adapters/argocd"
	"github.com/agnaldom/crosswatc/internal/adapters/github"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/store"
)

// Discovery is satisfied by *kube.Client.
type Discovery interface {
	DiscoverXRDs(ctx context.Context) ([]kube.XRD, error)
	DiscoverMRDs(ctx context.Context) (kube.MRIndex, error)
	DiscoverResources(ctx context.Context, xrds []kube.XRD, mrs kube.MRIndex) ([]kube.DiscoveredResource, error)
	DiscoverOne(ctx context.Context, x kube.XRD, xrds []kube.XRD, mrs kube.MRIndex, namespace, name string) (kube.DiscoveredResource, error)
}

// PullRequests is satisfied by *github.Adapter.
type PullRequests interface {
	PullRequest(ctx context.Context, l github.Lookup) (lifecycle.PRSignal, error)
}

// Argo is satisfied by *argocd.Adapter.
type Argo interface {
	ResolveApp(labels, annotations map[string]string) (argocd.AppRef, bool)
	Status(ctx context.Context, ref argocd.AppRef) (lifecycle.ArgoSignal, error)
}

// Backstage is satisfied by *backstage.Adapter.
type Backstage interface {
	Task(ctx context.Context, taskID string) (lifecycle.BackstageSignal, error)
}

// Store is the subset of *store.Store used by the reconciler.
type Store interface {
	UpsertResourceByClaim(ctx context.Context, r store.TrackedResource) (store.TrackedResource, error)
	UpdateResourceState(ctx context.Context, id uuid.UUID, expectedVersion int64, state string, everOperational bool) (int64, error)
	ListResources(ctx context.Context, f store.ResourceFilter, p store.Page) ([]store.TrackedResource, string, error)
	GetResourceByClaim(ctx context.Context, namespace, name, kind string) (store.TrackedResource, error)
	ListRevisions(ctx context.Context, resourceID uuid.UUID) ([]store.Revision, error)
	InsertRevision(ctx context.Context, r store.Revision) (store.Revision, error)
	UpdateRevision(ctx context.Context, r store.Revision) error
	InsertOperation(ctx context.Context, o store.Operation) (store.Operation, error)
	UpdateOperationsForRevision(ctx context.Context, revisionID uuid.UUID, status string, finishedAt *time.Time) error
	UpsertProjection(ctx context.Context, p store.Projection) error
	GetProjection(ctx context.Context, id uuid.UUID) (store.Projection, error)
	UpsertTopology(ctx context.Context, t store.Topology) error
	InsertEvent(ctx context.Context, e store.Event) (bool, error)
	SaveCheckpoint(ctx context.Context, source, cursor string, ingestErr error) error
}
