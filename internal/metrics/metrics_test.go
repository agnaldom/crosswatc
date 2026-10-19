package metrics

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerExposesCrosswatcMetrics(t *testing.T) {
	OperationsTotal.WithLabelValues("UPDATE").Inc()
	ResourcesTotal.WithLabelValues("OPERATIONAL").Set(3)
	HTTPDuration.WithLabelValues("GET", "/api/v1/resources/:id", "200").Observe(0.05)
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	body, _ := io.ReadAll(w.Body)
	for _, want := range []string{
		`crosswatc_operations_total{type="UPDATE"} 1`,
		`crosswatc_resources_total{state="OPERATIONAL"} 3`,
		`crosswatc_http_request_duration_seconds_bucket{code="200",method="GET",route="/api/v1/resources/:id",le="0.1"} 1`,
		"crosswatc_sse_connections 0",
		"go_goroutines",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q", want)
		}
	}
}
