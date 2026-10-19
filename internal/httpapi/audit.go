package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/store"
)

// AuditStore reads the audit trail (satisfied by *store.Store).
type AuditStore interface {
	ListAudit(ctx context.Context, f store.AuditFilter, p store.Page) ([]store.AuditEntry, string, error)
}

// Audit actions (spec §50).
const (
	AuditChangeRequested    = "change.requested"
	AuditChangeFailed       = "change.failed"
	AuditRevisionCancelled  = "revision.cancelled"
	AuditRevisionSuperseded = "revision.superseded"
)

// beforeAfter splits SpecChanges into before/after maps.
func beforeAfter(cs []changes.SpecChange) (json.RawMessage, json.RawMessage) {
	before, after := map[string]any{}, map[string]any{}
	for _, c := range changes.Redacted(cs) {
		before[c.Field], after[c.Field] = c.From, c.To
	}
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	return b, a
}

// audit records who/what/when (spec §50). Failures are logged but never block the action:
// the tracker_event trail still exists and the request has already reached GitHub.
func (h *handlers) audit(c *gin.Context, action string, r store.TrackedResource, rev *store.Revision, opID *uuid.UUID,
	before, after json.RawMessage) {
	ctx := c.Request.Context()
	cid, _ := correlation.FromContext(ctx)
	e := store.AuditEntry{Actor: principal(c).User, Action: action, ResourceID: &r.ID, OperationID: opID,
		CorrelationID: cid, Before: before, After: after}
	if rev != nil {
		e.RevisionID, e.PRURL, e.CommitSHA = &rev.ID, rev.PRURL, rev.CommitSHA
	}
	if err := h.changes.InsertAudit(ctx, e); err != nil {
		_ = c.Error(err)
	}
}

type auditDTO struct {
	ID            int64           `json:"id"`
	Actor         string          `json:"actor"`
	Action        string          `json:"action"`
	ResourceID    string          `json:"resourceId,omitempty"`
	RevisionID    string          `json:"revisionId,omitempty"`
	OperationID   string          `json:"operationId,omitempty"`
	CorrelationID string          `json:"correlationId,omitempty"`
	Before        json.RawMessage `json:"before,omitempty"`
	After         json.RawMessage `json:"after,omitempty"`
	PRURL         string          `json:"prUrl,omitempty"`
	CommitSHA     string          `json:"commitSha,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
}

func toAuditDTO(a store.AuditEntry) auditDTO {
	d := auditDTO{ID: a.ID, Actor: a.Actor, Action: a.Action, CorrelationID: a.CorrelationID, Before: a.Before, After: a.After,
		PRURL: a.PRURL, CommitSHA: a.CommitSHA, CreatedAt: a.CreatedAt}
	for dst, src := range map[*string]*uuid.UUID{&d.ResourceID: a.ResourceID, &d.RevisionID: a.RevisionID, &d.OperationID: a.OperationID} {
		if src != nil {
			*dst = src.String()
		}
	}
	return d
}

func (h *handlers) listAuditWith(c *gin.Context, f store.AuditFilter) {
	if h.auditStore == nil {
		problem(c, http.StatusServiceUnavailable, "audit-unavailable", "Audit not configured", "")
		return
	}
	p, ok := pageParams(c)
	if !ok {
		return
	}
	f.Actor, f.Action = c.Query("actor"), c.Query("action")
	items, next, err := h.auditStore.ListAudit(c.Request.Context(), f, p)
	if errors.Is(err, store.ErrBadCursor) {
		problem(c, http.StatusBadRequest, "invalid-cursor", "Invalid cursor", "")
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	out := make([]auditDTO, len(items))
	for i, a := range items {
		out[i] = toAuditDTO(a)
	}
	c.JSON(http.StatusOK, newPage(out, next))
}

// GET /api/v1/audit — scoped to the caller's squads (SRE: all).
func (h *handlers) listAudit(c *gin.Context) {
	f := store.AuditFilter{Owners: principal(c).OwnerScope()}
	if s := c.Query("resourceId"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			problem(c, http.StatusBadRequest, "invalid-resource-id", "Invalid resourceId", "")
			return
		}
		f.ResourceID = &id
	}
	h.listAuditWith(c, f)
}

// GET /api/v1/resources/{id}/audit — loadResource already enforced read access.
func (h *handlers) listResourceAudit(c *gin.Context) {
	id := resourceOf(c).ID
	h.listAuditWith(c, store.AuditFilter{ResourceID: &id})
}
