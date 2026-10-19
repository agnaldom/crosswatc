package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/store"
)

// revisionEventsLimit bounds the event history embedded in a revision detail.
const revisionEventsLimit = 200

var (
	operationStatuses = map[string]bool{
		string(domain.OperationQueued): true, string(domain.OperationRunning): true, string(domain.OperationWaitingApproval): true,
		string(domain.OperationSucceeded): true, string(domain.OperationFailed): true, string(domain.OperationCancelled): true,
	}
	operationTypes = map[string]bool{
		string(domain.OperationCreate): true, string(domain.OperationUpdate): true, string(domain.OperationDelete): true,
		string(domain.OperationPlatformUpgrade): true, string(domain.OperationReconcile): true,
	}
)

func badCursor(c *gin.Context, err error) bool {
	if errors.Is(err, store.ErrBadCursor) {
		problem(c, http.StatusBadRequest, "invalid-cursor", "Invalid cursor", "")
		return true
	}
	return false
}

// listRevisions serves GET /resources/{id}/revisions (Changes, §46), newest first, cursor paginated.
func (h *handlers) listRevisions(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	revs, next, err := h.store.ListRevisionsPage(c.Request.Context(), resourceOf(c).ID, p)
	if badCursor(c, err) {
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	out := make([]revisionDTO, len(revs))
	for i, r := range revs {
		out[i] = toRevisionDTO(r)
	}
	c.JSON(http.StatusOK, newPage(out, next))
}

type revisionDetailDTO struct {
	revisionDTO
	Operations []operationDTO `json:"operations"`
	Events     []eventDTO     `json:"events"`
}

// getRevision serves GET /resources/{id}/revisions/{rev}: the revision with its operations and event history.
func (h *handlers) getRevision(c *gin.Context) {
	n, err := strconv.Atoi(c.Param("rev"))
	if err != nil || n < 1 {
		notFound(c)
		return
	}
	ctx := c.Request.Context()
	rev, err := h.store.GetRevision(ctx, resourceOf(c).ID, n)
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(c)
		return
	case err != nil:
		internalError(c, err)
		return
	}
	ops, _, err := h.store.ListOperationsPage(ctx, store.OperationFilter{RevisionID: &rev.ID}, store.Page{Limit: 200})
	if err != nil {
		internalError(c, err)
		return
	}
	evs, err := h.store.ListEventsByRevision(ctx, rev.ID, revisionEventsLimit)
	if err != nil {
		internalError(c, err)
		return
	}
	out := revisionDetailDTO{revisionDTO: toRevisionDTO(rev), Operations: make([]operationDTO, len(ops)), Events: make([]eventDTO, len(evs))}
	for i, o := range ops {
		out.Operations[i] = toOperationDTO(o)
	}
	for i, e := range evs {
		out.Events[i] = toEventDTO(e)
	}
	c.JSON(http.StatusOK, out)
}

// operationFilter parses ?status and ?type (closed vocabularies of §15).
func operationFilter(c *gin.Context) (store.OperationFilter, bool) {
	f := store.OperationFilter{Status: c.Query("status"), Type: c.Query("type")}
	if f.Status != "" && !operationStatuses[f.Status] {
		problem(c, http.StatusBadRequest, "invalid-status", "Invalid status", "unknown operation status "+f.Status)
		return f, false
	}
	if f.Type != "" && !operationTypes[f.Type] {
		problem(c, http.StatusBadRequest, "invalid-type", "Invalid type", "unknown operation type "+f.Type)
		return f, false
	}
	return f, true
}

func (h *handlers) writeOperations(c *gin.Context, f store.OperationFilter) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	ops, next, err := h.store.ListOperationsPage(c.Request.Context(), f, p)
	if badCursor(c, err) {
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	out := make([]operationDTO, len(ops))
	for i, o := range ops {
		out[i] = toOperationDTO(o)
	}
	c.JSON(http.StatusOK, newPage(out, next))
}

// listResourceOperations serves GET /resources/{id}/operations.
func (h *handlers) listResourceOperations(c *gin.Context) {
	f, ok := operationFilter(c)
	if !ok {
		return
	}
	id := resourceOf(c).ID
	f.ResourceID = &id
	h.writeOperations(c, f)
}

// listOperations serves GET /operations: operation history across every resource the caller can read
// (owner ∈ squads, or SRE), optionally narrowed by ?resourceId, ?status and ?type.
func (h *handlers) listOperations(c *gin.Context) {
	f, ok := operationFilter(c)
	if !ok {
		return
	}
	f.Owners = principal(c).OwnerScope()
	if q := c.Query("resourceId"); q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			problem(c, http.StatusBadRequest, "invalid-resource-id", "Invalid resourceId", "")
			return
		}
		f.ResourceID = &id
	}
	h.writeOperations(c, f)
}

// getOperation serves GET /operations/{operationId}. Operations of unreadable resources return 404.
func (h *handlers) getOperation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("operationId"))
	if err != nil {
		notFound(c)
		return
	}
	o, err := h.store.GetOperation(c.Request.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound) || (err == nil && !principal(c).CanRead(o.Owner)):
		notFound(c)
	case err != nil:
		internalError(c, err)
	default:
		c.JSON(http.StatusOK, toOperationDTO(o))
	}
}
