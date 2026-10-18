// Package metrics defines the crosswatc_* Prometheus metrics (spec §52).
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry holds every Crosswatc metric plus Go/process collectors.
var Registry = prometheus.NewRegistry()

var (
	durationBuckets = []float64{1, 5, 15, 30, 60, 120, 300, 600, 1200, 1800, 3600, 7200, 14400}
	latencyBuckets  = []float64{.005, .01, .025, .05, .1, .25, .4, .5, 1, 2.5, 5, 10}

	ResourcesTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "crosswatc_resources_total", Help: "Tracked resources by state.",
	}, []string{"state"})
	OperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crosswatc_operations_total", Help: "Operations created.",
	}, []string{"type"})
	OperationsFailedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crosswatc_operations_failed_total", Help: "Operations that ended FAILED or REJECTED.",
	}, []string{"type", "reason"})
	OperationsStuckTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crosswatc_operations_stuck_total", Help: "Revisions flagged without progress beyond the stuck timeout.",
	})
	RevisionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "crosswatc_revision_duration_seconds", Help: "Revision start to terminal state.", Buckets: durationBuckets,
	}, []string{"kind", "state"})
	StageDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "crosswatc_stage_duration_seconds", Help: "Time a revision spent in each state.", Buckets: durationBuckets,
	}, []string{"state"})
	EventIngestionLag = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "crosswatc_event_ingestion_lag_seconds", Help: "Webhook received → processed.", Buckets: latencyBuckets,
	}, []string{"source"})
	ReconcileDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "crosswatc_reconcile_duration_seconds", Help: "Time to reconcile one resource.", Buckets: latencyBuckets,
	})
	ResyncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "crosswatc_resync_duration_seconds", Help: "Full resync duration.", Buckets: durationBuckets,
	})
	ReconcileErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crosswatc_reconcile_errors_total", Help: "Per-resource reconcile failures.",
	})
	SSEConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "crosswatc_sse_connections", Help: "Open SSE subscriptions.",
	})
	SSEDropped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crosswatc_sse_dropped_total", Help: "Subscribers dropped for being too slow.",
	})
	DriftTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crosswatc_drift_total", Help: "Drift detections.",
	})
	DriftActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "crosswatc_drift_active", Help: "Resources currently in drift (last resync).",
	})
	InformerRestarts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crosswatc_informer_restarts_total", Help: "Informer watch errors (each triggers a relist).",
	}, []string{"watch"})
	WebhookFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crosswatc_webhook_failures_total", Help: "Rejected or failed webhook deliveries.",
	}, []string{"source", "reason"})
	HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "crosswatc_http_request_duration_seconds", Help: "API request latency.", Buckets: latencyBuckets,
	}, []string{"method", "route", "code"})
)

func init() {
	Registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		ResourcesTotal, OperationsTotal, OperationsFailedTotal, OperationsStuckTotal, RevisionDuration, StageDuration,
		EventIngestionLag, ReconcileDuration, ResyncDuration, ReconcileErrors, SSEConnections, SSEDropped, DriftTotal,
		DriftActive, InformerRestarts, WebhookFailures, HTTPDuration)
}

// Handler serves /metrics.
func Handler() http.Handler { return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{}) }
