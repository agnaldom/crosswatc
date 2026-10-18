package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/agnaldom/crosswatc/internal/events"
	"github.com/agnaldom/crosswatc/internal/informer"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/metrics"
	"github.com/agnaldom/crosswatc/internal/store"
)

const (
	inboxBatch       = 50
	inboxMaxAttempts = 10
)

// InboxStore is satisfied by *store.Store.
type InboxStore interface {
	ProcessInbox(ctx context.Context, limit, maxAttempts int, fn func(store.InboxItem) error) (int, error)
	ListResources(ctx context.Context, f store.ResourceFilter, p store.Page) ([]store.TrackedResource, string, error)
}

// consumeInbox drains webhook deliveries (spec §35.1). Webhooks are triggers, not truth: each delivery is
// normalized to canonical events, routed to the affected resources, and those Claims are re-read from the
// sources by the reconciler. Unroutable deliveries are acknowledged; the periodic resync covers them.
func (r *Runner) consumeInbox(ctx context.Context) {
	interval := r.InboxInterval
	if interval <= 0 {
		interval = time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		for {
			n, err := r.Inbox.ProcessInbox(ctx, inboxBatch, inboxMaxAttempts, func(it store.InboxItem) error {
				return r.handleDelivery(ctx, it)
			})
			if err != nil && ctx.Err() == nil {
				r.log().Warn("inbox processing failed", "err", err)
			}
			if err != nil || n < inboxBatch {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// normalize maps an inbox row to canonical events.
func normalize(it store.InboxItem) (events.Normalized, error) {
	switch it.Source {
	case events.SourceGitHub:
		return events.FromGitHub(it.EventType, it.DeliveryID, it.Payload, it.ReceivedAt)
	case events.SourceArgoCD:
		return events.FromArgo(it.Payload, it.ReceivedAt)
	}
	return events.Normalized{}, fmt.Errorf("unknown inbox source %q", it.Source)
}

func (r *Runner) handleDelivery(ctx context.Context, it store.InboxItem) error {
	metrics.EventIngestionLag.WithLabelValues(it.Source).Observe(time.Since(it.ReceivedAt).Seconds())
	n, err := normalize(it)
	if err != nil {
		return fmt.Errorf("%w: %w", store.ErrPermanent, err)
	}
	for _, e := range n.Events {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%w: %w", store.ErrPermanent, err)
		}
		r.log().Info("canonical event", "id", e.EventID, "type", e.Type, "source", e.Source, "metadata", e.Metadata)
	}
	targets, err := r.route(ctx, n.Hints)
	if err != nil {
		return err // retried
	}
	for _, t := range targets {
		r.Enqueue(t)
	}
	r.log().Debug("webhook routed", "source", it.Source, "delivery", it.DeliveryID, "event", it.EventType, "targets", len(targets))
	return nil
}

// route resolves webhook hints to reconcile targets: correlation IDs first, then the GitOps repository.
// Argo CD notifications trigger a resync, which re-reads every Application through the reconciler.
func (r *Runner) route(ctx context.Context, h events.Hints) ([]informer.Target, error) {
	if h.ArgoApp != "" {
		return []informer.Target{{Full: true}}, nil
	}
	if len(h.CorrelationIDs) > 0 {
		ts, err := r.targets(ctx, store.ResourceFilter{CorrelationIDs: h.CorrelationIDs})
		if err != nil || len(ts) > 0 {
			return ts, err
		}
	}
	if h.Repository != "" {
		return r.targets(ctx, store.ResourceFilter{Repository: h.Repository})
	}
	return nil, nil
}

func (r *Runner) targets(ctx context.Context, f store.ResourceFilter) ([]informer.Target, error) {
	var out []informer.Target
	cursor := ""
	for {
		page, next, err := r.Inbox.ListResources(ctx, f, store.Page{Limit: 200, Cursor: cursor})
		if err != nil {
			return nil, err
		}
		for _, tr := range page {
			out = append(out, informer.Target{Claim: kube.Ref{
				APIVersion: tr.ClaimAPIVersion, Kind: tr.ClaimKind, Namespace: tr.ClaimNamespace, Name: tr.ClaimName}})
		}
		if next == "" {
			return out, nil
		}
		cursor = next
	}
}
