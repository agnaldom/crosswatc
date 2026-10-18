package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agnaldom/crosswatc/internal/drift"
	"github.com/agnaldom/crosswatc/internal/events"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
	"github.com/agnaldom/crosswatc/internal/metrics"
	"github.com/agnaldom/crosswatc/internal/store"
)

// GitReader is satisfied by *github.Adapter.
type GitReader interface {
	GetFile(ctx context.Context, repository, path, ref string) ([]byte, string, error)
}

type gitEntry struct {
	content []byte
	err     error
	at      time.Time
}

// manifest returns the cached manifest, refetching after GitCacheTTL.
func (r *Reconciler) manifest(ctx context.Context, repo, path string) ([]byte, error) {
	ttl := r.GitCacheTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	key := repo + "@" + path
	r.gitMu.Lock()
	if r.gitCache == nil {
		r.gitCache = map[string]gitEntry{}
	}
	e, ok := r.gitCache[key]
	r.gitMu.Unlock()
	if ok && r.now().Sub(e.at) < ttl {
		return e.content, e.err
	}
	base := r.BaseBranch
	if base == "" {
		base = "main"
	}
	b, _, err := r.Git.GetFile(ctx, repo, path, base)
	r.gitMu.Lock()
	r.gitCache[key] = gitEntry{content: b, err: err, at: r.now()}
	r.gitMu.Unlock()
	return b, err
}

// detectDrift evaluates §47 signals and emits DETECTED/RESOLVED transitions.
func (r *Reconciler) detectDrift(ctx context.Context, tr store.TrackedResource, d kube.DiscoveredResource, sig lifecycle.Signals) drift.State {
	sig.EverOperational = sig.EverOperational || tr.EverOperational
	var prev drift.State
	if p, err := r.Store.GetProjection(ctx, tr.ID); err == nil && len(p.DriftState) > 0 {
		_ = json.Unmarshal(p.DriftState, &prev)
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		r.log().Warn("read previous drift failed", "resource", tr.ID, "err", err)
	}

	in := drift.Input{Signals: sig, Previous: prev, Now: r.now()}
	in.ClaimParameters, _ = d.Claim.Spec["parameters"].(map[string]any)
	if r.Git != nil && tr.Repository != "" && tr.GitPath != "" && sig.EverOperational && sig.OpenRevision == nil {
		if b, err := r.manifest(ctx, tr.Repository, tr.GitPath); err != nil {
			r.log().Warn("read manifest for drift failed", "resource", tr.ID, "err", err)
		} else if params, err := gitops.ClaimParameters(b, tr.ClaimName); err != nil {
			r.log().Warn("parse manifest for drift failed", "resource", tr.ID, "err", err)
		} else {
			in.GitParameters = params
		}
	}

	cur := drift.Detect(in)
	switch detected, resolved := drift.Transition(prev, cur); {
	case detected:
		metrics.DriftTotal.Inc()
		msg := "Drift de configuração detectado: " + strings.Join(cur.Signals, ", ")
		if len(cur.Differences) > 0 {
			df := cur.Differences[0]
			msg += fmt.Sprintf(" (%s: desejado %v, observado %v)", df.Field, df.Desired, df.Observed)
		}
		r.event(ctx, tr, nil, fmt.Sprintf("drift:%s:%d", tr.ID, cur.DetectedAt.UnixNano()), events.SourceCrosswatc, events.LifecycleDriftDetect, msg)
	case resolved:
		at := int64(0)
		if prev.DetectedAt != nil {
			at = prev.DetectedAt.UnixNano()
		}
		r.event(ctx, tr, nil, fmt.Sprintf("drift-resolved:%s:%d", tr.ID, at), events.SourceCrosswatc, events.LifecycleDriftResolve,
			"Drift de configuração resolvido")
	}
	if cur.Detected {
		r.driftActive.Add(1)
	}
	return cur
}
