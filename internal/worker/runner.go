// Package worker runs the leader-only loop: informers feed a debounced work queue that refreshes single
// Claims, and a periodic full resync covers anything missed (spec §25, §35.1, §35.2).
package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/agnaldom/crosswatc/internal/informer"
	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/reconciler"
)

// Reconciler is satisfied by *reconciler.Reconciler.
type Reconciler interface {
	RunOnce(ctx context.Context) (reconciler.Result, error)
	ReconcileClaim(ctx context.Context, ref kube.Ref) error
	Catalog() ([]kube.XRD, kube.MRIndex)
}

// Informers is satisfied by *informer.Manager.
type Informers interface {
	informer.Cache
	Ensure(ctx context.Context, watches []informer.Watch)
	HasSynced() bool
	Stop()
}

// Runner is the worker main loop. It must only run on the elected leader.
type Runner struct {
	Rec       Reconciler
	Informers Informers // nil: periodic resync only
	Watch     informer.Options
	Resync    time.Duration
	Debounce  time.Duration // coalesces bursts of informer events (default 200ms)
	Log       *slog.Logger

	Inbox         InboxStore // optional: webhook deliveries (spec §35.1)
	InboxInterval time.Duration

	resolver atomic.Pointer[informer.Resolver]
	resynced atomic.Bool
	mu       sync.Mutex
	queue    workqueue.TypedRateLimitingInterface[informer.Target]
}

// Ready reports that a full resync completed and every informer cache is synced (spec §39 /readyz).
func (r *Runner) Ready() bool {
	return r.resynced.Load() && (r.Informers == nil || r.Informers.HasSynced())
}

// OnEvent is the informer handler: it resolves the affected Claim and enqueues it.
func (r *Runner) OnEvent(ev informer.RawEvent) {
	res := r.resolver.Load()
	if res == nil {
		return
	}
	t, ok := res.Resolve(ev)
	if !ok {
		return
	}
	r.log().Debug("informer event", "type", ev.Type, "watch", ev.Watch.Key(), "object", ev.Object.GetName(), "target", t)
	r.Enqueue(t)
}

// Enqueue schedules a target after the debounce delay. Duplicates waiting in the queue are coalesced.
// It is a no-op while the runner is not leading.
func (r *Runner) Enqueue(t informer.Target) {
	r.mu.Lock()
	q := r.queue
	r.mu.Unlock()
	if q == nil {
		return
	}
	d := r.Debounce
	if d <= 0 {
		d = 200 * time.Millisecond
	}
	q.AddAfter(t, d)
}

// Run processes the queue until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) error {
	q := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.NewTypedItemExponentialFailureRateLimiter[informer.Target](time.Second, 2*time.Minute),
		workqueue.TypedRateLimitingQueueConfig[informer.Target]{Name: "crosswatc-worker"})
	r.mu.Lock()
	r.queue = q
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.queue = nil
		r.mu.Unlock()
		q.ShutDown()
		if r.Informers != nil {
			r.Informers.Stop()
		}
		r.resolver.Store(nil)
		r.resynced.Store(false)
	}()

	resync := r.Resync
	if resync <= 0 {
		resync = 5 * time.Minute
	}
	go func() {
		t := time.NewTicker(resync)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				q.ShutDown()
				return
			case <-t.C:
				q.Add(informer.Target{Full: true})
			}
		}
	}()

	if r.Inbox != nil {
		go r.consumeInbox(ctx)
	}
	q.Add(informer.Target{Full: true})
	for r.processNext(ctx, q) {
	}
	return ctx.Err()
}

func (r *Runner) processNext(ctx context.Context, q workqueue.TypedRateLimitingInterface[informer.Target]) bool {
	t, shutdown := q.Get()
	if shutdown {
		return false
	}
	defer q.Done(t)
	if t.Full {
		r.fullResync(ctx)
		q.Forget(t) // the ticker retries full resyncs
		return true
	}
	if err := r.Rec.ReconcileClaim(ctx, t.Claim); err != nil {
		if ctx.Err() == nil {
			r.log().Warn("claim reconcile failed; retrying", "claim", t.Claim.ID(), "err", err)
			q.AddRateLimited(t)
		}
		return true
	}
	q.Forget(t)
	return true
}

func (r *Runner) fullResync(ctx context.Context) {
	start := time.Now()
	res, err := r.Rec.RunOnce(ctx)
	if err != nil {
		r.log().Error("resync finished with errors", "err", err, "result", res, "took", time.Since(start))
	} else {
		r.resynced.Store(true)
		r.log().Info("resync finished", "result", res, "took", time.Since(start))
	}
	xrds, mrs := r.Rec.Catalog()
	if r.Informers != nil {
		r.resolver.Store(&informer.Resolver{Cache: r.Informers, XRDs: xrds})
		r.Informers.Ensure(ctx, informer.Watches(xrds, mrs, r.Watch))
	}
}

func (r *Runner) log() *slog.Logger {
	if r.Log != nil {
		return r.Log
	}
	return slog.Default()
}
