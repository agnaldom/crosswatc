package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/topology"
)

// Cluster reads live objects and events from the single control cluster (read-only; satisfied by *kube.Client).
type Cluster interface {
	EventsFor(ctx context.Context, ref kube.Ref) ([]kube.Event, error)
}

const (
	maxEventObjects  = 40
	eventConcurrency = 6
)

type objectError struct {
	Object kube.Ref `json:"object"`
	Error  string   `json:"error"`
}

// topologyRefs returns the objects of r known to the read model (missing nodes excluded).
// Only these may be inspected: callers can never name an arbitrary cluster object.
func (h *handlers) topologyRefs(c *gin.Context, r store.TrackedResource) ([]kube.Ref, []topology.Node, bool) {
	t, err := h.store.GetTopology(c.Request.Context(), r.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, true
	}
	if err != nil {
		internalError(c, err)
		return nil, nil, false
	}
	var nodes []topology.Node
	if err := json.Unmarshal(t.Nodes, &nodes); err != nil {
		internalError(c, err)
		return nil, nil, false
	}
	var refs []kube.Ref
	var kept []topology.Node
	for _, n := range nodes {
		if n.Status == topology.StatusMissing || n.Kind == "" || n.Name == "" {
			continue
		}
		refs = append(refs, kube.Ref{APIVersion: n.APIVersion, Kind: n.Kind, Name: n.Name, Namespace: n.Namespace})
		kept = append(kept, n)
	}
	return refs, kept, true
}

// GET /resources/{id}/kube-events — Kubernetes Events of the Claim, XR and MRs (spec §46 Events).
func (h *handlers) listKubeEvents(c *gin.Context) {
	if h.cluster == nil {
		problem(c, http.StatusServiceUnavailable, "cluster-unavailable", "Kubernetes access not configured", "")
		return
	}
	limit := 200
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	refs, _, ok := h.topologyRefs(c, resourceOf(c))
	if !ok {
		return
	}
	if len(refs) > maxEventObjects {
		refs = refs[:maxEventObjects]
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var (
		mu     sync.Mutex
		events = []kube.Event{}
		errs   = []objectError{}
		wg     sync.WaitGroup
		sem    = make(chan struct{}, eventConcurrency)
	)
	for _, ref := range refs {
		wg.Add(1)
		sem <- struct{}{}
		go func(ref kube.Ref) {
			defer func() { <-sem; wg.Done() }()
			es, err := h.cluster.EventsFor(ctx, ref)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, objectError{Object: ref, Error: err.Error()})
				return
			}
			events = append(events, es...)
		}(ref)
	}
	wg.Wait()
	kube.SortEvents(events)
	if len(events) > limit {
		events = events[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"items": events, "errors": errs})
}
