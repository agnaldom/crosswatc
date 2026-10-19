package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/forms"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/redact"
	"github.com/agnaldom/crosswatc/internal/store"
)

// Divergence is a field whose Git desired value differs from the applied Claim (spec §43).
type Divergence struct {
	Field   string `json:"field"`
	Git     any    `json:"git"`
	Cluster any    `json:"cluster"`
}

// checkConsistency blocks updates while Git and the Claim disagree (spec §43): opening a PR
// on top of an unsynced desired state would hide what is actually being changed.
func (h *handlers) checkConsistency(c *gin.Context, r store.TrackedResource, in changes.Input) bool {
	ctx := c.Request.Context()
	if synced, known := h.syncedWithGit(ctx, r); known && !synced {
		problem(c, http.StatusConflict, "update-blocked", "Update blocked",
			"Argo CD has not applied the current Git state yet; retry after it syncs")
		return false
	}
	manifest, err := h.gitops.Manifest(ctx, r.Repository, r.GitPath)
	if err != nil {
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "git-unavailable", "Could not read desired state from Git", err.Error())
		return false
	}
	gitParams, err := gitops.ClaimParameters(manifest, r.ClaimName)
	if err != nil {
		problem(c, http.StatusConflict, "update-blocked", "Update blocked", "Claim manifest not readable: "+err.Error())
		return false
	}
	clusterParams, _ := in.CurrentSpec["parameters"].(map[string]any)
	div := diverging(in.Form, gitParams, clusterParams)
	if len(div) == 0 {
		return true
	}
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(http.StatusConflict, gin.H{
		"type": problemBase + "update-blocked", "title": "Update blocked", "status": http.StatusConflict,
		"detail": "Git desired state differs from the applied Claim; wait for reconciliation", "divergences": div,
	})
	return false
}

func (h *handlers) syncedWithGit(ctx context.Context, r store.TrackedResource) (synced, known bool) {
	p, err := h.store.GetProjection(ctx, r.ID)
	if err != nil || len(p.AppliedState) == 0 {
		return false, false
	}
	var applied struct {
		SyncedWithGit *bool `json:"syncedWithGit"`
	}
	if json.Unmarshal(p.AppliedState, &applied) != nil || applied.SyncedWithGit == nil {
		return false, false
	}
	return *applied.SyncedWithGit, true
}

// diverging compares only fields declared in Git: fields present only on the Claim are API-server
// defaults or late-initialized values, not drift (same rule as drift.Compare). Redacted/sensitive
// values cannot be compared and are skipped.
func diverging(f forms.Form, git, cluster map[string]any) []Divergence {
	var out []Divergence
	for _, fl := range f.Fields {
		g, gok := forms.Lookup(git, fl.Key)
		if !gok {
			continue
		}
		k, _ := forms.Lookup(cluster, fl.Key)
		if k == redact.Placeholder || redact.Sensitive(lastKey(fl.Key)) {
			continue
		}
		if fmt.Sprint(g) != fmt.Sprint(k) {
			out = append(out, Divergence{Field: fl.Key, Git: g, Cluster: k})
		}
	}
	return out
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

// cancellable states: before merge (spec §32 exit transitions).
var cancellable = map[string]bool{"CREATED": true, "VALIDATING": true, "WAITING_APPROVAL": true}

// cancelRevision closes the PR and marks the revision CANCELLED.
func (h *handlers) cancelRevision(c *gin.Context) {
	r := resourceOf(c)
	if !requireWrite(c, r) {
		return
	}
	if h.gitops == nil || h.changes == nil {
		problem(c, http.StatusServiceUnavailable, "gitops-unavailable", "GitOps not configured", "")
		return
	}
	n, err := strconv.Atoi(c.Param("rev"))
	if err != nil || n < 1 {
		notFound(c)
		return
	}
	var req cancelRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChangeBody)
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		problem(c, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return
	}
	ctx := c.Request.Context()
	rev, err := h.store.GetRevision(ctx, r.ID, n)
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(c)
		return
	case err != nil:
		internalError(c, err)
		return
	}
	if !cancellable[rev.State] {
		problem(c, http.StatusConflict, "not-cancellable", "Revision cannot be cancelled",
			fmt.Sprintf("Revision %d is %s; only revisions before merge can be cancelled", rev.Rev, rev.State))
		return
	}
	if err := h.endRevision(ctx, r, rev, domain.RevisionCancelled, principal(c).User, req.Reason); err != nil {
		_ = c.Error(err)
		problem(c, http.StatusBadGateway, "gitops-failed", "Could not close Pull Request", err.Error())
		return
	}
	reason, _ := json.Marshal(map[string]string{"reason": req.Reason})
	h.audit(c, AuditRevisionCancelled, r, &rev, nil, nil, reason)
	c.Status(http.StatusNoContent)
}

// endRevision closes the PR (if any) and moves rev to CANCELLED or SUPERSEDED.
func (h *handlers) endRevision(ctx context.Context, r store.TrackedResource, rev store.Revision, state domain.RevisionState, by, reason string) error {
	if rev.PRNumber != nil {
		msg := reason
		if state == domain.RevisionSuperseded {
			msg = strings.TrimSpace("Substituída por uma nova revision. " + reason)
		}
		if err := h.gitops.Cancel(ctx, r.Repository, *rev.PRNumber, by, msg); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	rev.State, rev.CompletedAt = string(state), &now
	if err := h.changes.UpdateRevision(ctx, rev); err != nil {
		return err
	}
	if err := h.changes.UpdateOperationsForRevision(ctx, rev.ID, "CANCELLED", &now); err != nil {
		return err
	}
	h.emit(ctx, r, &rev, fmt.Sprintf("rev:%s:%s", rev.ID, state), "crosswatc", "GITOPS.PR.CLOSED", "warning",
		fmt.Sprintf("Revision %d %s por %s", rev.Rev, strings.ToLower(string(state)), by))
	return nil
}

func lastKey(k string) string {
	if i := strings.LastIndex(k, "."); i >= 0 {
		return k[i+1:]
	}
	return k
}
