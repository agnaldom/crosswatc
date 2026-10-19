package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/correlation"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/metrics"
	"github.com/agnaldom/crosswatc/internal/store"
	"github.com/agnaldom/crosswatc/internal/topology"
)

// AuditDecommissionRequested is recorded when a decommission PR is opened.
const AuditDecommissionRequested = "decommission.requested"

type affected struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Provider     string `json:"provider,omitempty"`
	ExternalName string `json:"externalName,omitempty"`
}

type decommissionPreview struct {
	Allowed          bool       `json:"allowed"`
	Reason           string     `json:"reason,omitempty"`
	Environment      string     `json:"environment"`
	ApprovalRequired bool       `json:"approvalRequired"`
	ConfirmName      string     `json:"confirmName"` // value the user must type back
	Deleted          []affected `json:"deleted"`
	Orphaned         []affected `json:"orphaned"`
	Warnings         []string   `json:"warnings"`
}

// decommissionCheck explains why r cannot be decommissioned ("" when allowed).
func (h *handlers) decommissionCheck(c *gin.Context, r store.TrackedResource) string {
	allowed := h.decommissionEnvs
	switch {
	case len(allowed) == 0:
		return "Decommissioning desabilitado neste ambiente do Crosswatc"
	case !slices.Contains(allowed, "*") && !slices.Contains(allowed, r.Environment):
		return fmt.Sprintf("Decommissioning não é permitido pelo portal em %q; acione a Plataforma", r.Environment)
	case r.ResourceState == string(domain.ResourceDeletePending) || r.ResourceState == string(domain.ResourceDeleting) ||
		r.ResourceState == string(domain.ResourceDeleted):
		return "Recurso já está em remoção (" + r.ResourceState + ")"
	case r.Repository == "" || r.GitPath == "":
		return "Recurso sem repositório/path GitOps conhecido"
	}
	revs, err := h.store.ListRevisions(c.Request.Context(), r.ID)
	if err == nil {
		if open := currentRevision(revs); open != nil && !domain.RevisionState(open.State).Terminal() {
			return fmt.Sprintf("Revision %d ainda aberta (%s); cancele-a ou aguarde", open.Rev, open.State)
		}
	}
	return ""
}

func (h *handlers) buildDecommissionPreview(c *gin.Context, r store.TrackedResource) (decommissionPreview, bool) {
	p := decommissionPreview{Environment: r.Environment, ApprovalRequired: true, ConfirmName: r.Name,
		Deleted: []affected{}, Orphaned: []affected{}, Warnings: []string{}}
	if reason := h.decommissionCheck(c, r); reason != "" {
		p.Reason = reason
	} else {
		p.Allowed = true
	}
	t, err := h.store.GetTopology(c.Request.Context(), r.ID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		p.Warnings = append(p.Warnings, "Topologia ainda não conhecida; não é possível listar os recursos de cloud afetados")
	case err != nil:
		internalError(c, err)
		return p, false
	default:
		var nodes []topology.Node
		if err := json.Unmarshal(t.Nodes, &nodes); err != nil {
			internalError(c, err)
			return p, false
		}
		for _, n := range nodes {
			if n.Type != topology.TypeManaged {
				continue
			}
			a := affected{Kind: n.Kind, Name: n.Name, Provider: n.Provider, ExternalName: n.ExternalName}
			if n.DeletionPolicy == "Orphan" {
				p.Orphaned = append(p.Orphaned, a)
			} else {
				p.Deleted = append(p.Deleted, a)
			}
		}
	}
	if strings.HasPrefix(r.Environment, "pr") {
		p.Warnings = append(p.Warnings, "Ambiente produtivo: a remoção é irreversível e apaga dados")
	}
	if len(p.Deleted) > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d recurso(s) de cloud serão apagados após o merge", len(p.Deleted)))
	}
	return p, true
}

// POST /resources/{id}/decommission/preview — impact preview (spec §60).
func (h *handlers) previewDecommission(c *gin.Context) {
	r := resourceOf(c)
	if !requireWrite(c, r) {
		return
	}
	if p, ok := h.buildDecommissionPreview(c, r); ok {
		c.JSON(http.StatusOK, p)
	}
}

type decommissionRequest struct {
	Justification string `json:"justification"`
	ConfirmName   string `json:"confirmName"`
}

// POST /resources/{id}/decommission — opens the PR removing the Claim (double confirmation: name typed back).
func (h *handlers) decommission(c *gin.Context) {
	r := resourceOf(c)
	if !requireWrite(c, r) {
		return
	}
	if h.gitops == nil || h.changes == nil {
		problem(c, http.StatusServiceUnavailable, "gitops-unavailable", "GitOps not configured", "")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChangeBody)
	var req decommissionRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		problem(c, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return
	}
	if req.ConfirmName != r.Name {
		problem(c, http.StatusBadRequest, "confirmation-mismatch", "Confirmation does not match",
			"Type the resource name exactly to confirm the decommission")
		return
	}
	if len(strings.TrimSpace(req.Justification)) < 15 {
		problem(c, http.StatusBadRequest, "justification-required", "Justification required", "At least 15 characters")
		return
	}
	pv, ok := h.buildDecommissionPreview(c, r)
	if !ok {
		return
	}
	if !pv.Allowed {
		problem(c, http.StatusConflict, "decommission-blocked", "Decommission blocked", pv.Reason)
		return
	}

	ctx := c.Request.Context()
	revs, err := h.store.ListRevisions(ctx, r.ID)
	if err != nil {
		internalError(c, err)
		return
	}
	next := 1
	if len(revs) > 0 {
		next = revs[0].Rev + 1
	}
	cid, _ := correlation.FromContext(ctx)
	user := principal(c).User
	now := time.Now().UTC()
	impact, _ := json.Marshal(map[string]any{"deleted": pv.Deleted, "orphaned": pv.Orphaned})
	rev, err := h.changes.InsertRevision(ctx, store.Revision{ResourceID: r.ID, Rev: next, Kind: string(domain.RevisionDelete),
		State: string(domain.RevisionCreated), Changes: impact, StartedAt: &now})
	if errors.Is(err, store.ErrConflict) {
		h.revisionOpen(c, revs)
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	op, err := h.changes.InsertOperation(ctx, store.Operation{ResourceID: r.ID, RevisionID: &rev.ID, CorrelationID: cid,
		Type: string(domain.OperationDelete), Status: string(domain.OperationQueued), RequestedBy: user, RequestedAt: now})
	if err != nil {
		internalError(c, err)
		return
	}
	metrics.OperationsTotal.WithLabelValues(string(domain.OperationDelete)).Inc()

	names := func(as []affected) []string {
		out := make([]string, len(as))
		for i, a := range as {
			out[i] = a.Kind + "/" + a.Name
		}
		return out
	}
	res, err := h.gitops.SubmitDelete(ctx, gitops.DeleteRequest{
		Repository: r.Repository, Path: r.GitPath, ClaimName: r.ClaimName, Kind: r.Kind, Environment: r.Environment,
		ResourceName: r.Name, Rev: next, CorrelationID: cid, RequestedBy: user, Justification: req.Justification,
		Deleted: names(pv.Deleted), Orphaned: names(pv.Orphaned),
	})
	if err != nil {
		reason, _ := json.Marshal(map[string]string{"technicalCode": "GitOpsError", "userMessage": "Não foi possível abrir o Pull Request.",
			"technicalMessage": err.Error(), "owner": "platform"})
		rev.State, rev.FailureReason, rev.CompletedAt = string(domain.RevisionFailed), reason, &now
		_ = h.changes.UpdateRevision(ctx, rev)
		_ = h.changes.UpdateOperationsForRevision(ctx, rev.ID, string(domain.OperationFailed), &now)
		metrics.OperationsFailedTotal.WithLabelValues("DELETE", "GitOpsError").Inc()
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "gitops-failed", "Could not open Pull Request", err.Error())
		return
	}
	pr := res.PRNumber
	rev.State, rev.Branch, rev.CommitSHA, rev.PRNumber, rev.PRURL = string(domain.RevisionValidating), res.Branch, res.CommitSHA, &pr, res.PRURL
	if err := h.changes.UpdateRevision(ctx, rev); err != nil {
		internalError(c, err)
		return
	}
	_ = h.changes.UpdateOperationsForRevision(ctx, rev.ID, string(domain.OperationRunning), nil)
	h.emit(ctx, r, &rev, fmt.Sprintf("rev:%s:VALIDATING", rev.ID), "github", "GITOPS.PR.CREATED", "warning",
		fmt.Sprintf("Decommission solicitado por %s: Pull Request #%d", user, pr))
	h.audit(c, AuditDecommissionRequested, r, &rev, &op.ID, nil, impact)

	c.JSON(http.StatusCreated, changeResponse{OperationID: op.ID.String(), RevisionID: rev.ID.String(), Rev: next,
		CorrelationID: cid, PRNumber: pr, PRURL: res.PRURL, Branch: res.Branch})
}
