package httpapi

import (
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
)

func TestSecurityHeaders(t *testing.T) {
	_, _, h := setup()
	w := do(h, "/healthz", nil)
	for k, v := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Cache-Control": "no-store"} {
		if w.Header().Get(k) != v {
			t.Errorf("%s = %q", k, w.Header().Get(k))
		}
	}
}

func TestMutationRateLimit(t *testing.T) {
	fs, _, _ := setup()
	h := New(Deps{Store: fs, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}, MutationsPerMinute: 1})
	path := "/api/v1/resources/" + payID.String() + "/changes/preview"
	limited := 0
	for range 8 {
		if w := post(h, path, `{"changes":{}}`, payments); w.Code == 429 {
			limited++
			if w.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After missing")
			}
		}
	}
	if limited != 3 { // burst 5 then limited
		t.Fatalf("limited %d of 8", limited)
	}
	other := map[string]string{"X-Dev-User": "other", "X-Dev-Squads": "payments"}
	if w := post(h, path, `{"changes":{}}`, other); w.Code == 429 {
		t.Fatal("limit is per user")
	}
	if w := do(h, "/api/v1/resources", payments); w.Code != 200 {
		t.Fatal("GETs are not limited")
	}
	if mutationLimiter(-1) != nil || mutationLimiter(0) == nil {
		t.Fatal("limiter config")
	}
}
