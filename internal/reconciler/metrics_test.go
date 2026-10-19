package reconciler

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/agnaldom/crosswatc/internal/kube"
	"github.com/agnaldom/crosswatc/internal/metrics"
)

func TestReconcilerMetrics(t *testing.T) {
	creates := testutil.ToFloat64(metrics.OperationsTotal.WithLabelValues("CREATE"))
	resyncs := testutil.CollectAndCount(metrics.ResyncDuration)
	st := newMemStore()
	r := &Reconciler{Discovery: &fakeDiscovery{resources: []kube.DiscoveredResource{discovered(ready, "")}}, Store: st}
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ToFloat64(metrics.OperationsTotal.WithLabelValues("CREATE")); got != creates+1 {
		t.Fatalf("operations_total CREATE %v → %v", creates, got)
	}
	if testutil.ToFloat64(metrics.ResourcesTotal.WithLabelValues("OPERATIONAL")) != 1 {
		t.Fatal("resources_total gauge")
	}
	if testutil.CollectAndCount(metrics.ResyncDuration) != resyncs || testutil.CollectAndCount(metrics.RevisionDuration) == 0 {
		t.Fatal("histograms must be registered and observed")
	}
}
