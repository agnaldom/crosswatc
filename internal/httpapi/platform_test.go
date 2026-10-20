package httpapi

import (
	"context"
	"errors"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/domain"
	"github.com/agnaldom/crosswatc/internal/kube"
)

type fakePlatform struct{ err, v2err error }

func (f fakePlatform) MRDs(context.Context) ([]kube.MRD, error) {
	if f.v2err != nil {
		return nil, f.v2err
	}
	return []kube.MRD{{Name: "flexibleservers.azure", Kind: "FlexibleServer", State: "Active", Active: true}}, nil
}
func (f fakePlatform) MRAPs(context.Context) ([]kube.MRAP, error) {
	if f.v2err != nil {
		return nil, f.v2err
	}
	return nil, nil
}

func (f fakePlatform) Packages(_ context.Context, kind string) ([]kube.Package, error) {
	return []kube.Package{{Kind: kind, Name: "provider-azure", Conditions: domain.Conditions{{Type: "Healthy", Status: "True"}}}}, f.err
}
func (f fakePlatform) Compositions(context.Context) ([]kube.Composition, error) {
	return []kube.Composition{{Name: "pg-azure", Mode: "Pipeline", Functions: []string{"function-go-templating"}}}, f.err
}
func (f fakePlatform) DiscoverXRDs(context.Context) ([]kube.XRD, error) {
	return []kube.XRD{{Name: "xpg", Kind: "XPostgreSQLInstance", Established: true, Versions: []kube.XRDVersion{{Name: "v1", Served: true}}}}, f.err
}

func TestPlatformCatalogEndpoints(t *testing.T) {
	fs, _, _ := setup()
	mapping := auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}
	h := New(Deps{Store: fs, Platform: fakePlatform{}, Auth: auth.Dev{Mapping: mapping}})
	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}

	if w := do(h, "/api/v1/me", payments); w.Body.String() != `{"squads":["payments"],"sre":false,"user":"agnaldo"}` {
		t.Fatalf("%s", w.Body)
	}
	if w := do(h, "/api/v1/me", sre); !strings.Contains(w.Body.String(), `"sre":true`) {
		t.Fatalf("%s", w.Body)
	}
	for _, p := range []string{"providers", "functions", "configurations", "xrds", "compositions"} {
		if w := do(h, "/api/v1/platform/"+p, payments); w.Code != 403 {
			t.Errorf("%s: squad members are not SRE: %d", p, w.Code)
		}
		if w := do(h, "/api/v1/platform/"+p, sre); w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[`) {
			t.Errorf("%s: %d %s", p, w.Code, w.Body)
		}
	}
	if w := do(h, "/api/v1/platform/functions", sre); !strings.Contains(w.Body.String(), `"kind":"Function"`) {
		t.Fatalf("%s", w.Body)
	}
	if w := do(New(Deps{Store: fs, Platform: fakePlatform{err: errors.New("x")}, Auth: auth.Dev{Mapping: mapping}}), "/api/v1/platform/providers", sre); w.Code != 502 {
		t.Fatal(w.Code)
	}
	if w := do(New(Deps{Store: fs, Auth: auth.Dev{Mapping: mapping}}), "/api/v1/platform/xrds", sre); w.Code != 503 {
		t.Fatal(w.Code)
	}
}

func TestPlatformV2CatalogDegrades(t *testing.T) {
	fs, _, _ := setup()
	mapping := auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "sre"}
	sre := map[string]string{"X-Dev-User": "ops", "X-Dev-SRE": "true"}
	h := New(Deps{Store: fs, Platform: fakePlatform{}, Auth: auth.Dev{Mapping: mapping}})
	if w := do(h, "/api/v1/platform/mrds", sre); w.Code != 200 || !strings.Contains(w.Body.String(), `"available":true`) ||
		!strings.Contains(w.Body.String(), `"active":true`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if w := do(h, "/api/v1/platform/mraps", sre); w.Body.String() != `{"available":true,"items":[]}` {
		t.Fatalf("nil list must be []: %s", w.Body)
	}
	if w := do(h, "/api/v1/platform/mrds", payments); w.Code != 403 {
		t.Fatalf("SRE only: %d", w.Code)
	}
	missing := apierrors.NewNotFound(schema.GroupResource{Group: "apiextensions.crossplane.io", Resource: "managedresourcedefinitions"}, "")
	hv1 := New(Deps{Store: fs, Platform: fakePlatform{v2err: missing}, Auth: auth.Dev{Mapping: mapping}})
	for _, p := range []string{"mrds", "mraps"} {
		if w := do(hv1, "/api/v1/platform/"+p, sre); w.Code != 200 || w.Body.String() != `{"available":false,"items":[]}` {
			t.Fatalf("%s on Crossplane v1: %d %s", p, w.Code, w.Body)
		}
	}
	hErr := New(Deps{Store: fs, Platform: fakePlatform{v2err: errors.New("boom")}, Auth: auth.Dev{Mapping: mapping}})
	if w := do(hErr, "/api/v1/platform/mrds", sre); w.Code != 502 {
		t.Fatalf("real errors still fail: %d", w.Code)
	}
}
