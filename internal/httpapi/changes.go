package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/store"
)

const maxChangeBody = 64 << 10

type changeRequest struct {
	Changes       map[string]any `json:"changes"`
	Justification string         `json:"justification"`
	Supersede     bool           `json:"supersede"` // cancel an open pre-merge revision and replace it (SUPERSEDED)
}

// requireWrite enforces resource.owner ∈ user.squads for mutations; SRE read is not enough.
func requireWrite(c *gin.Context, r store.TrackedResource) bool {
	if principal(c).CanWrite(r.Owner) {
		return true
	}
	problem(c, http.StatusForbidden, "forbidden", "Not allowed to change this resource",
		"Only members of squad "+r.Owner+" can request changes")
	return false
}

func bindChange(c *gin.Context) (changeRequest, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChangeBody)
	var req changeRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		problem(c, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return req, false
	}
	return req, true
}

// previewInput loads form + applied spec for a change against r.
func (h *handlers) previewInput(c *gin.Context, r store.TrackedResource, req changeRequest) (changes.Input, bool) {
	f, _, ok := h.formFor(c, r.Kind)
	if !ok {
		return changes.Input{}, false
	}
	spec, ok := h.appliedSpec(c, r)
	if !ok {
		return changes.Input{}, false
	}
	return changes.Input{
		Form: f, CurrentSpec: spec, Changes: req.Changes, Environment: r.Environment, ResourceState: r.ResourceState,
		ResourceName: r.Name, Owner: r.Owner, Provider: r.Provider, Kind: r.Kind,
	}, true
}

func (h *handlers) previewChanges(c *gin.Context) {
	r := resourceOf(c)
	if !requireWrite(c, r) {
		return
	}
	req, ok := bindChange(c)
	if !ok {
		return
	}
	in, ok := h.previewInput(c, r, req)
	if !ok {
		return
	}
	res, err := changes.Preview(c.Request.Context(), in, h.policies...)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ChangeStore persists change requests (satisfied by *store.Store).
type ChangeStore interface {
	InsertRevision(ctx context.Context, r store.Revision) (store.Revision, error)
	UpdateRevision(ctx context.Context, r store.Revision) error
	InsertOperation(ctx context.Context, o store.Operation) (store.Operation, error)
	UpdateOperationsForRevision(ctx context.Context, revisionID uuid.UUID, status string, finishedAt *time.Time) error
	InsertEvent(ctx context.Context, e store.Event) (bool, error)
}

// GitOps opens and closes change Pull Requests (satisfied by *gitops.Orchestrator).
type GitOps interface {
	Submit(ctx context.Context, req gitops.Request) (gitops.Result, error)
	Manifest(ctx context.Context, repository, path string) ([]byte, error)
	Cancel(ctx context.Context, repository string, number int, by, reason string) error
}

type changeResponse struct {
	OperationID   string `json:"operationId"`
	RevisionID    string `json:"revisionId"`
	Rev           int    `json:"rev"`
	CorrelationID string `json:"correlationId"`
	PRNumber      int    `json:"prNumber"`
	PRURL         string `json:"prUrl"`
	Branch        string `json:"branch"`
}

// createChange implements the Update Workflow (spec §42): preview → Operation → Revision → branch/commit/PR.
func (h *handlers) createChange(c *gin.Context) {
	r := resourceOf(c)
	if !requireWrite(c, r) {
		return
	}
	if h.gitops == nil || h.changes == nil {
		problem(c, http.StatusServiceUnavailable, "gitops-unavailable", "GitOps not configured", "GitHub App is not configured on the API")
		return
	}
	req, ok := bindChange(c)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Justification) == "" {
		problem(c, http.StatusBadRequest, "justification-required", "Justification required", "")
		return
	}
	in, ok := h.previewInput(c, r, req)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	pv, err := changes.Preview(ctx, in, h.policies...)
	if err != nil {
		internalError(c, err)
		return
	}
	if !pv.Valid {
		c.JSON(http.StatusUnprocessableEntity, pv)
		return
	}
	if !h.checkConsistency(c, r, in) {
		return
	}

	revs, err := h.store.ListRevisions(ctx, r.ID)
	if err != nil {
		internalError(c, err)
		return
	}
	if open := currentRevision(revs); open != nil && !domain.RevisionState(open.State).Terminal() {
		if !req.Supersede || !cancellable[open.State] {
			h.revisionOpen(c, revs)
			return
		}
		if err := h.endRevision(ctx, r, *open, domain.RevisionSuperseded, principal(c).User, ""); err != nil {
			_ = c.Error(err)
			problem(c, http.StatusBadGateway, "gitops-failed", "Could not close the superseded Pull Request", err.Error())
			return
		}
	}
	next := 1
	if len(revs) > 0 {
		next = revs[0].Rev + 1
	}
	cid, _ := correlation.FromContext(ctx)
	user := principal(c).User
	now := time.Now().UTC()
	changesJSON, _ := json.Marshal(pv.Changes)

	rev, err := h.changes.InsertRevision(ctx, store.Revision{ResourceID: r.ID, Rev: next, Kind: "UPDATE", State: "CREATED",
		Changes: changesJSON, StartedAt: &now})
	if errors.Is(err, store.ErrConflict) {
		h.revisionOpen(c, revs)
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	op, err := h.changes.InsertOperation(ctx, store.Operation{ResourceID: r.ID, RevisionID: &rev.ID, CorrelationID: cid,
		Type: "UPDATE", Status: "QUEUED", RequestedBy: user, RequestedAt: now})
	if err != nil {
		internalError(c, err)
		return
	}
	h.emit(ctx, r, &rev, "request:"+cid, "crosswatc", "REQUEST.CREATED", "info",
		fmt.Sprintf("Mudança solicitada por %s (rev %d)", user, next))

	res, err := h.gitops.Submit(ctx, gitops.Request{
		Repository: r.Repository, Path: r.GitPath, ClaimName: r.ClaimName, Kind: r.Kind, Environment: r.Environment,
		ResourceName: r.Name, Rev: next, CorrelationID: cid, RequestedBy: user, Justification: req.Justification,
		Changes: pv.Changes, Approval: pv.Impact.ApprovalRequired,
	})
	if err != nil {
		reason, _ := json.Marshal(map[string]string{"technicalCode": "GitOpsError", "userMessage": "Não foi possível abrir o Pull Request.",
			"technicalMessage": err.Error(), "owner": "platform"})
		rev.State, rev.FailureReason, rev.CompletedAt = "FAILED", reason, &now
		_ = h.changes.UpdateRevision(ctx, rev)
		_ = h.changes.UpdateOperationsForRevision(ctx, rev.ID, "FAILED", &now)
		h.emit(ctx, r, &rev, "gitops-failed:"+rev.ID.String(), "github", "GITOPS.PR.FAILED", "error", err.Error())
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "gitops-failed", "Could not open Pull Request", err.Error())
		return
	}

	pr := res.PRNumber
	rev.State, rev.Branch, rev.CommitSHA, rev.PRNumber, rev.PRURL = "VALIDATING", res.Branch, res.CommitSHA, &pr, res.PRURL
	if err := h.changes.UpdateRevision(ctx, rev); err != nil {
		internalError(c, err)
		return
	}
	_ = h.changes.UpdateOperationsForRevision(ctx, rev.ID, "RUNNING", nil)
	h.emit(ctx, r, &rev, fmt.Sprintf("rev:%s:VALIDATING", rev.ID), "github", "GITOPS.PR.CREATED", "info",
		fmt.Sprintf("Pull Request #%d criado", pr))

	c.JSON(http.StatusCreated, changeResponse{OperationID: op.ID.String(), RevisionID: rev.ID.String(), Rev: next,
		CorrelationID: cid, PRNumber: pr, PRURL: res.PRURL, Branch: res.Branch})
}

func (h *handlers) revisionOpen(c *gin.Context, revs []store.Revision) {
	detail := "Resource already has an open revision"
	if cur := currentRevision(revs); cur != nil && !domain.RevisionState(cur.State).Terminal() {
		detail = fmt.Sprintf("Revision %d is %s (id %s)", cur.Rev, cur.State, cur.ID)
	}
	problem(c, http.StatusConflict, "revision-open", "Revision already open", detail)
}

func (h *handlers) emit(ctx context.Context, r store.TrackedResource, rev *store.Revision, key, source, typ, level, msg string) {
	cid, _ := correlation.FromContext(ctx)
	e := store.Event{DedupeKey: key, ResourceID: &r.ID, CorrelationID: cid, Source: source, Type: typ, Level: level, Message: msg}
	if rev != nil {
		e.RevisionID = &rev.ID
	}
	_, _ = h.changes.InsertEvent(ctx, e)
}
