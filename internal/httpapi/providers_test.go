package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/kube"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fakeProviderManager struct {
	fakePlatform
	installed  string
	installErr error
	existing   []kube.Package
	calls      int
	failAt     int
}

func (f *fakeProviderManager) Packages(context.Context, string) ([]kube.Package, error) {
	return f.existing, f.err
}
func (f *fakeProviderManager) ProviderConfigs(context.Context) ([]kube.ProviderConfigSummary, error) {
	return []kube.ProviderConfigSummary{{Name: "default", Provider: "azure"}}, f.err
}
func (f *fakeProviderManager) InstallProvider(_ context.Context, name, ref, actor string) (kube.Package, error) {
	f.calls++
	if f.failAt > 0 && f.calls == f.failAt {
		return kube.Package{}, errors.New("temporary cluster failure")
	}
	f.installed += name + " " + ref + " " + actor + "\n"
	if f.installErr == nil {
		f.existing = append(f.existing, kube.Package{Name: name, Package: ref})
	}
	return kube.Package{Name: name, Package: ref}, f.installErr
}

func TestProviderCatalogAccessAndInstallation(t *testing.T) {
	manager := &fakeProviderManager{}
	h := New(Deps{Store: &fakeStore{}, Platform: manager, Auth: auth.Dev{Mapping: auth.GroupMapping{SREGroup: "sre", AdminGroup: "admins"}}, MutationsPerMinute: -1})
	for _, path := range []string{"provider-catalog", "provider-configs"} {
		req := httptest.NewRequest("GET", "/api/v1/platform/"+path, nil)
		req.Header.Set("X-Dev-User", "observer")
		req.Header.Set("X-Dev-SRE", "true")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[`) {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body)
		}
	}
	for _, tc := range []struct {
		name, id, version string
		admin             bool
		want              int
	}{
		{"observer cannot install", "azure", "v2.8.0", false, 403},
		{"unknown repository rejected", "arbitrary", "v1.0.0", true, 404},
		{"mutable version rejected", "aws", "latest", true, 400},
		{"bundle release must match package inventory", "azure", "v2.7.0", true, 400},
		{"path injection rejected", "aws", "v1.0.0/other", true, 400},
		{"explicit version accepted", "azure", "v2.8.0", true, 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager.installed = ""
			req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/"+tc.id+"/install", strings.NewReader(`{"version":"`+tc.version+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Dev-User", "operator")
			req.Header.Set("X-Dev-SRE", "true")
			if tc.admin {
				req.Header.Set("X-Dev-Admin", "true")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("%d: %s", w.Code, w.Body)
			}
			if tc.want != 202 && manager.installed != "" {
				t.Fatal("rejected request reached Kubernetes")
			}
			if tc.want == 202 && !strings.Contains(manager.installed, "crosswatc-provider-family-azure xpkg.crossplane.io/crossplane-contrib/provider-family-azure:v2.8.0 operator") {
				t.Fatal(manager.installed)
			}
		})
	}
	// A retry is idempotent across the entire cloud bundle, preserving existing versions.
	manager.installed = ""
	retry := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/azure/install", strings.NewReader(`{"version":"v2.8.0"}`))
	retry.Header.Set("X-Dev-User", "operator")
	retry.Header.Set("X-Dev-SRE", "true")
	retry.Header.Set("X-Dev-Admin", "true")
	retryResult := httptest.NewRecorder()
	h.ServeHTTP(retryResult, retry)
	if retryResult.Code != 202 || manager.installed != "" || !strings.Contains(retryResult.Body.String(), `"existing":92`) {
		t.Fatalf("retry: %d %s", retryResult.Code, retryResult.Body)
	}
	manager.existing = nil
	manager.installErr = apierrors.NewAlreadyExists(schema.GroupResource{Group: "pkg.crossplane.io", Resource: "providers"}, "crosswatc-azure")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/azure/install", strings.NewReader(`{"version":"v2.8.0"}`))
	req.Header.Set("X-Dev-User", "operator")
	req.Header.Set("X-Dev-SRE", "true")
	req.Header.Set("X-Dev-Admin", "true")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 409 {
		t.Fatalf("duplicate: %d %s", w.Code, w.Body)
	}
}

func TestProviderSelectiveInstallation(t *testing.T) {
	manager := &fakeProviderManager{}
	h := New(Deps{Store: &fakeStore{}, Platform: manager, Auth: auth.Dev{Mapping: auth.GroupMapping{SREGroup: "sre", AdminGroup: "admins"}}, MutationsPerMinute: -1})

	// 1. Invalid package rejected
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/azure/install", strings.NewReader(`{"version":"v2.8.0","packages":["xpkg.crossplane.io/crossplane-contrib/unknown-service"]}`))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("X-Dev-User", "operator")
	reqBad.Header.Set("X-Dev-SRE", "true")
	reqBad.Header.Set("X-Dev-Admin", "true")
	wBad := httptest.NewRecorder()
	h.ServeHTTP(wBad, reqBad)
	if wBad.Code != 400 || !strings.Contains(wBad.Body.String(), "invalid-package") {
		t.Fatalf("expected 400 invalid-package, got %d: %s", wBad.Code, wBad.Body)
	}

	// 2. Selective install of compute and storage automatically includes family package
	manager.installed = ""
	reqGood := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/azure/install", strings.NewReader(`{"version":"v2.8.0","packages":["xpkg.crossplane.io/crossplane-contrib/provider-azure-compute","xpkg.crossplane.io/crossplane-contrib/provider-azure-storage"]}`))
	reqGood.Header.Set("Content-Type", "application/json")
	reqGood.Header.Set("X-Dev-User", "operator")
	reqGood.Header.Set("X-Dev-SRE", "true")
	reqGood.Header.Set("X-Dev-Admin", "true")
	wGood := httptest.NewRecorder()
	h.ServeHTTP(wGood, reqGood)
	if wGood.Code != 202 {
		t.Fatalf("expected 202, got %d: %s", wGood.Code, wGood.Body)
	}
	if !strings.Contains(wGood.Body.String(), `"created":3`) || !strings.Contains(wGood.Body.String(), `"total":3`) {
		t.Fatalf("expected 3 packages total (2 selected + 1 family), got: %s", wGood.Body)
	}
	if !strings.Contains(manager.installed, "crosswatc-provider-family-azure") ||
		!strings.Contains(manager.installed, "crosswatc-provider-azure-compute") ||
		!strings.Contains(manager.installed, "crosswatc-provider-azure-storage") {
		t.Fatalf("unexpected installed list: %s", manager.installed)
	}
	if strings.Contains(manager.installed, "crosswatc-provider-azure-sql") {
		t.Fatalf("unselected package should not be installed: %s", manager.installed)
	}
}

func TestCompleteCloudCatalog(t *testing.T) {
	expected := map[string]int{"azure": 92, "aws": 181, "gcp": 82}
	for _, entry := range providerCatalog {
		seen := map[string]bool{}
		if count, ok := expected[entry.ID]; ok && len(entry.Packages) != count {
			t.Fatalf("%s: expected %d packages, got %d", entry.ID, count, len(entry.Packages))
		}
		for _, repo := range entry.Packages {
			name := catalogPackageName(repo)
			if seen[name] || len("crosswatc-"+name) > 63 || !strings.HasPrefix(repo, "xpkg.crossplane.io/crossplane-contrib/") {
				t.Fatal(repo)
			}
			seen[name] = true
		}
	}
}

func TestProviderBundleResumesAfterPartialFailure(t *testing.T) {
	manager := &fakeProviderManager{failAt: 3}
	h := New(Deps{Store: &fakeStore{}, Platform: manager, Auth: auth.Dev{Mapping: auth.GroupMapping{SREGroup: "sre", AdminGroup: "admin"}}, MutationsPerMinute: -1})
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/azure/install", strings.NewReader(`{"version":"v2.8.0"}`))
		req.Header.Set("X-Dev-User", "operator")
		req.Header.Set("X-Dev-SRE", "true")
		req.Header.Set("X-Dev-Admin", "true")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	first := send()
	if first.Code != 502 || len(manager.existing) != 2 {
		t.Fatalf("partial: %d %s", first.Code, first.Body)
	}
	manager.failAt = 0
	manager.calls = 0
	second := send()
	if second.Code != 202 || manager.calls != 90 || len(manager.existing) != 92 || !strings.Contains(second.Body.String(), `"existing":2`) {
		t.Fatalf("resume: %d %s calls=%d", second.Code, second.Body, manager.calls)
	}
}

type fakeProviderObserver struct{ fakeProviderManager }

func (f *fakeProviderObserver) ProviderDiagnostics(context.Context, string) (kube.ProviderDiagnostics, error) {
	return kube.ProviderDiagnostics{Provider: kube.Package{Name: "provider-a"}}, nil
}
func (f *fakeProviderObserver) ProviderLogs(context.Context, string) ([]kube.ProviderLog, error) {
	return []kube.ProviderLog{{Text: "controller output"}}, nil
}

func TestProviderDiagnosticsAuthorizationAndConflicts(t *testing.T) {
	observer := &fakeProviderObserver{}
	h := New(Deps{Store: &fakeStore{}, Platform: observer, Auth: auth.Dev{Mapping: auth.GroupMapping{SREGroup: "sre", AdminGroup: "admins"}}, MutationsPerMinute: -1})
	for _, tc := range []struct {
		path  string
		admin bool
		want  int
	}{
		{"/api/v1/platform/providers/provider-a/diagnostics", false, 200},
		{"/api/v1/platform/providers/provider-a/logs", false, 403},
		{"/api/v1/platform/providers/provider-a/logs", true, 200},
		{"/api/v1/platform/providers/Bad_Name/diagnostics", true, 400},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("X-Dev-User", "operator")
		req.Header.Set("X-Dev-SRE", "true")
		if tc.admin {
			req.Header.Set("X-Dev-Admin", "true")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body)
		}
	}
	observer.existing = []kube.Package{{Name: "legacy-azure", Package: "xpkg.upbound.io/upbound/provider-family-azure:v1.3.0"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/provider-catalog/azure/install", strings.NewReader(`{"version":"v2.8.0"}`))
	req.Header.Set("X-Dev-User", "operator")
	req.Header.Set("X-Dev-SRE", "true")
	req.Header.Set("X-Dev-Admin", "true")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 409 || observer.calls != 0 || !strings.Contains(w.Body.String(), "v1.3.0") {
		t.Fatalf("conflict must be detected before any create: %d %s", w.Code, w.Body)
	}
}
