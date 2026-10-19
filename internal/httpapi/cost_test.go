package httpapi

import (
	"context"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/cost"
	"github.com/agnaldom/crosswatc/internal/forms"
	"github.com/agnaldom/crosswatc/internal/kube"
)

type pricedCatalog struct{}

func (pricedCatalog) DiscoverXRDs(context.Context) ([]kube.XRD, error) {
	x := pgXRDWithForm()
	x.Annotations = map[string]string{
		forms.MetadataAnnotation: `{"vmSize": {"editable": true}}`,
		cost.Annotation:          "- {name: vm, service: Virtual Machines, sku: spec.parameters.vmSize, regionDefault: brazilsouth}",
	}
	return []kube.XRD{x}, nil
}

type prices map[string]float64

func (p prices) Price(_ context.Context, q cost.PriceQuery) (cost.Price, error) {
	return cost.Price{Unit: p[q.SKU], Currency: "USD"}, nil
}

func TestPreviewIncludesCost(t *testing.T) {
	fs, _, _ := setup()
	p := fs.projection[payID]
	p.AppliedState = []byte(`{"spec":{"parameters":{"vmSize":"Standard_D2as_v5"}}}`)
	fs.projection[payID] = p
	h := New(Deps{Store: fs, Catalog: pricedCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}},
		Costs: &cost.Estimator{Prices: prices{"Standard_D2as_v5": 0.1, "Standard_D4as_v5": 0.2}}})
	var res changes.Result
	decode(t, post(h, "/api/v1/resources/"+payID.String()+"/changes/preview", `{"changes":{"vmSize":"Standard_D4as_v5"}}`, payments), &res)
	if res.Cost == nil || res.Cost.Current.Monthly != 73 || res.Cost.New.Monthly != 146 || res.Cost.MonthlyDelta != 73 {
		t.Fatalf("%+v", res.Cost)
	}
	if costSummary(res.Cost) != "73.00 → 146.00 USD/mês (+73.00)" {
		t.Fatalf("%q", costSummary(res.Cost))
	}

	// without estimator or annotation: no cost block
	h = New(Deps{Store: fs, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}},
		Costs: &cost.Estimator{Prices: prices{}}})
	res = changes.Result{}
	decode(t, post(h, "/api/v1/resources/"+payID.String()+"/changes/preview", `{"changes":{"vmSize":"Standard_D4as_v5"}}`, payments), &res)
	if res.Cost != nil {
		t.Fatal("unpriced types have no cost")
	}
}
