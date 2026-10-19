// Package httpapi is the Crosswatc REST API (Gin) — spec §29, §39.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/store"
)

// ReadStore is the read model used by the API (satisfied by *store.Store).
type ReadStore interface {
	Ping(ctx context.Context) error
	GetResource(ctx context.Context, id uuid.UUID) (store.TrackedResource, error)
	ListResources(ctx context.Context, f store.ResourceFilter, p store.Page) ([]store.TrackedResource, string, error)
	GetProjection(ctx context.Context, id uuid.UUID) (store.Projection, error)
	GetTopology(ctx context.Context, id uuid.UUID) (store.Topology, error)
	ListEvents(ctx context.Context, id uuid.UUID, p store.Page) ([]store.Event, string, error)
	ListRevisions(ctx context.Context, id uuid.UUID) ([]store.Revision, error)
	// History (spec §39, §46 Changes/Events).
	ListRevisionsPage(ctx context.Context, id uuid.UUID, p store.Page) ([]store.Revision, string, error)
	GetRevision(ctx context.Context, id uuid.UUID, rev int) (store.Revision, error)
	ListOperationsPage(ctx context.Context, f store.OperationFilter, p store.Page) ([]store.Operation, string, error)
	GetOperation(ctx context.Context, id uuid.UUID) (store.Operation, error)
	ListEventsByRevision(ctx context.Context, revisionID uuid.UUID, limit int) ([]store.Event, error)
}

// Catalog lists platform APIs (satisfied by *kube.Client).
type Catalog interface {
	DiscoverXRDs(ctx context.Context) ([]kube.XRD, error)
}

// Deps wires the API.
type Deps struct {
	Store   ReadStore
	Catalog Catalog // optional
	Auth    auth.Authenticator
	Log     *slog.Logger

	Inbox    Inbox // optional: enables /webhooks/* (spec §35.1)
	Webhooks WebhookConfig

	Policies []changes.Policy // extra policies for change preview (spec §40)
	Changes  ChangeStore      // optional with GitOps: enables POST /resources/{id}/changes (spec §41)
	GitOps   GitOps
	Audit    AuditStore // optional: GET /audit

	// DecommissionEnvironments lists environments where the portal may open decommission PRs ("*" = all; empty disables).
	DecommissionEnvironments []string

	// MutationsPerMinute limits POSTs per user: 0 → default 10/min (burst 5), negative → disabled.
	MutationsPerMinute int

	Stream          Streamer      // optional: GET /api/v1/stream (spec §45)
	StreamHeartbeat time.Duration // default 20s
}

// New builds the Gin engine.
func New(d Deps) *gin.Engine {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), otelgin.Middleware("crosswatc-api"), securityHeaders(), withCorrelation(), observe(), accessLog(d.Log))
	r.NoRoute(notFound)

	h := &handlers{store: d.Store, catalog: &catalogCache{src: d.Catalog, ttl: time.Minute}, streamer: d.Stream, heartbeat: d.StreamHeartbeat, policies: d.Policies,
		changes: d.Changes, gitops: d.GitOps, auditStore: d.Audit,
		decommissionEnvs: d.DecommissionEnvironments}
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/readyz", h.ready)
	if d.Inbox != nil {
		(&webhooks{inbox: d.Inbox, cfg: d.Webhooks}).register(r)
	}

	v1 := r.Group("/api/v1", authenticate(d.Auth), limitMutations(mutationLimiter(d.MutationsPerMinute)))
	v1.GET("/stream", h.stream)
	v1.GET("/catalog/resource-types", h.listResourceTypes)
	v1.GET("/catalog/resource-types/:kind", h.getResourceType)
	v1.GET("/catalog/resource-types/:kind/schema", h.getResourceTypeSchema)
	v1.GET("/catalog/resource-types/:kind/form", h.getResourceTypeForm)

	v1.GET("/operations", h.listOperations)
	v1.GET("/operations/:operationId", h.getOperation)

	v1.GET("/resources", h.listResources)
	v1.GET("/audit", h.listAudit)
	res := v1.Group("/resources/:id", h.loadResource)
	res.GET("", h.getResource)
	res.GET("/state", h.getState)
	res.GET("/health", h.getHealth)
	res.GET("/events", h.listEvents)
	res.GET("/revisions", h.listRevisions)
	res.GET("/revisions/:rev", h.getRevision)
	res.GET("/operations", h.listResourceOperations)
	res.GET("/topology", h.getTopology)
	res.GET("/form", h.getResourceForm)
	res.POST("/changes/preview", h.previewChanges)
	res.POST("/changes", h.createChange)
	res.POST("/revisions/:rev/cancel", h.cancelRevision)
	res.GET("/audit", h.listResourceAudit)
	res.POST("/decommission/preview", h.previewDecommission)
	res.POST("/decommission", h.decommission)
	return r
}

type handlers struct {
	store      ReadStore
	catalog    *catalogCache
	streamer   Streamer
	heartbeat  time.Duration
	policies   []changes.Policy
	changes    ChangeStore
	gitops     GitOps
	auditStore AuditStore

	decommissionEnvs []string
}

const resourceKey = "crosswatc.resource"

func (h *handlers) ready(c *gin.Context) {
	if err := h.store.Ping(c.Request.Context()); err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	c.Status(http.StatusOK)
}

// loadResource fetches {id} and enforces AuthZ. Unauthorized reads return 404 to avoid leaking existence.
func (h *handlers) loadResource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	r, err := h.store.GetResource(c.Request.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(c)
		return
	case err != nil:
		internalError(c, err)
		return
	}
	if !principal(c).CanRead(r.Owner) {
		notFound(c)
		return
	}
	c.Set(resourceKey, r)
	c.Next()
}

func resourceOf(c *gin.Context) store.TrackedResource {
	v, _ := c.Get(resourceKey)
	r, _ := v.(store.TrackedResource)
	return r
}

func pageParams(c *gin.Context) (store.Page, bool) {
	p := store.Page{Cursor: c.Query("cursor")}
	if s := c.Query("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 200 {
			problem(c, http.StatusBadRequest, "invalid-limit", "Invalid limit", "limit must be between 1 and 200")
			return p, false
		}
		p.Limit = n
	}
	return p, true
}

func (h *handlers) listResources(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	f := store.ResourceFilter{
		Owners: principal(c).OwnerScope(), Owner: c.Query("owner"), Environment: c.Query("environment"),
		Provider: c.Query("provider"), Kind: c.Query("kind"), State: c.Query("status"), NamePrefix: c.Query("q"),
	}
	items, next, err := h.store.ListResources(c.Request.Context(), f, p)
	if errors.Is(err, store.ErrBadCursor) {
		problem(c, http.StatusBadRequest, "invalid-cursor", "Invalid cursor", "")
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	out := make([]resourceDTO, len(items))
	for i, r := range items {
		out[i] = toResourceDTO(r)
	}
	c.JSON(http.StatusOK, newPage(out, next))
}

func (h *handlers) getResource(c *gin.Context) {
	c.JSON(http.StatusOK, toResourceDTO(resourceOf(c)))
}

type revisionSummary struct {
	Rev   int    `json:"rev"`
	State string `json:"state"`
	PRURL string `json:"prUrl,omitempty"`
}

type stateDTO struct {
	ResourceState   string             `json:"resourceState"`
	CurrentRevision *revisionSummary   `json:"currentRevision"`
	Stages          []domain.StageView `json:"stages"`
	Drift           json.RawMessage    `json:"drift"`
	LastReconcileAt *time.Time         `json:"lastReconcileAt"`
}

func (h *handlers) getState(c *gin.Context) {
	r := resourceOf(c)
	ctx := c.Request.Context()
	out := stateDTO{ResourceState: r.ResourceState, Stages: []domain.StageView{}, Drift: json.RawMessage(`{"detected":false}`)}

	p, err := h.store.GetProjection(ctx, r.ID)
	switch {
	case err == nil:
		if len(p.Stages) > 0 {
			if err := json.Unmarshal(p.Stages, &out.Stages); err != nil {
				internalError(c, err)
				return
			}
		}
		if len(p.DriftState) > 0 {
			out.Drift = p.DriftState
		}
		out.LastReconcileAt = p.LastReconcileAt
	case !errors.Is(err, store.ErrNotFound):
		internalError(c, err)
		return
	}

	revs, err := h.store.ListRevisions(ctx, r.ID)
	if err != nil {
		internalError(c, err)
		return
	}
	if cur := currentRevision(revs); cur != nil {
		out.CurrentRevision = &revisionSummary{Rev: cur.Rev, State: cur.State, PRURL: cur.PRURL}
	}
	c.JSON(http.StatusOK, out)
}

// currentRevision is the open revision, else the most recent one. revs are newest first.
func currentRevision(revs []store.Revision) *store.Revision {
	for i := range revs {
		if !domain.RevisionState(revs[i].State).Terminal() {
			return &revs[i]
		}
	}
	if len(revs) > 0 {
		return &revs[0]
	}
	return nil
}

func (h *handlers) getHealth(c *gin.Context) {
	p, err := h.store.GetProjection(c.Request.Context(), resourceOf(c).ID)
	switch {
	case errors.Is(err, store.ErrNotFound) || (err == nil && len(p.Health) == 0):
		c.JSON(http.StatusOK, gin.H{})
	case err != nil:
		internalError(c, err)
	default:
		c.Data(http.StatusOK, "application/json", p.Health)
	}
}

func (h *handlers) listEvents(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	items, next, err := h.store.ListEvents(c.Request.Context(), resourceOf(c).ID, p)
	if errors.Is(err, store.ErrBadCursor) {
		problem(c, http.StatusBadRequest, "invalid-cursor", "Invalid cursor", "")
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	out := make([]eventDTO, len(items))
	for i, e := range items {
		out[i] = toEventDTO(e)
	}
	c.JSON(http.StatusOK, newPage(out, next))
}

func (h *handlers) getTopology(c *gin.Context) {
	t, err := h.store.GetTopology(c.Request.Context(), resourceOf(c).ID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusOK, gin.H{"nodes": []any{}, "edges": []any{}})
	case err != nil:
		internalError(c, err)
	default:
		c.JSON(http.StatusOK, gin.H{"nodes": t.Nodes, "edges": t.Edges, "updatedAt": t.UpdatedAt})
	}
}

// --- catalog ---

type resourceTypeDTO struct {
	Kind        string   `json:"kind"`
	Group       string   `json:"group"`
	Version     string   `json:"version"`
	ClaimKind   string   `json:"claimKind,omitempty"`
	Scope       string   `json:"scope"`
	Established bool     `json:"established"`
	Versions    []string `json:"versions"`
}

func toResourceTypeDTO(x kube.XRD) resourceTypeDTO {
	d := resourceTypeDTO{Kind: x.Kind, Group: x.Group, Version: x.Version(), ClaimKind: x.ClaimKind, Scope: x.Scope, Established: x.Established}
	for _, v := range x.Versions {
		if v.Served {
			d.Versions = append(d.Versions, v.Name)
		}
	}
	return d
}

type catalogCache struct {
	src Catalog
	ttl time.Duration

	mu      sync.Mutex
	xrds    []kube.XRD
	fetched time.Time
}

func (cc *catalogCache) get(ctx context.Context) ([]kube.XRD, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.xrds != nil && time.Since(cc.fetched) < cc.ttl {
		return cc.xrds, nil
	}
	xrds, err := cc.src.DiscoverXRDs(ctx)
	if err != nil {
		return nil, err
	}
	if xrds == nil {
		xrds = []kube.XRD{}
	}
	cc.xrds, cc.fetched = xrds, time.Now()
	return xrds, nil
}

func (h *handlers) xrds(c *gin.Context) ([]kube.XRD, bool) {
	if h.catalog.src == nil {
		problem(c, http.StatusServiceUnavailable, "catalog-unavailable", "Catalog unavailable", "Kubernetes access not configured")
		return nil, false
	}
	xrds, err := h.catalog.get(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return nil, false
	}
	return xrds, true
}

// findKind matches the claim kind (user-facing) or the XR kind.
func findKind(xrds []kube.XRD, kind string) (kube.XRD, bool) {
	for _, x := range xrds {
		if x.ClaimKind == kind || x.Kind == kind {
			return x, true
		}
	}
	return kube.XRD{}, false
}

func (h *handlers) listResourceTypes(c *gin.Context) {
	xrds, ok := h.xrds(c)
	if !ok {
		return
	}
	out := make([]resourceTypeDTO, len(xrds))
	for i, x := range xrds {
		out[i] = toResourceTypeDTO(x)
	}
	c.JSON(http.StatusOK, newPage(out, ""))
}

func (h *handlers) getResourceType(c *gin.Context) {
	xrds, ok := h.xrds(c)
	if !ok {
		return
	}
	x, found := findKind(xrds, c.Param("kind"))
	if !found {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, toResourceTypeDTO(x))
}

func (h *handlers) getResourceTypeSchema(c *gin.Context) {
	xrds, ok := h.xrds(c)
	if !ok {
		return
	}
	x, found := findKind(xrds, c.Param("kind"))
	if !found {
		notFound(c)
		return
	}
	schema := x.Schema(c.Query("version"))
	if schema == nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"kind": c.Param("kind"), "version": x.Version(), "openAPIV3Schema": schema})
}
