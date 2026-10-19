package cost

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const annotation = `
- name: vm
  service: Virtual Machines
  sku: spec.parameters.vmSize
  region: spec.parameters.location
  regionDefault: brazilsouth
  quantity: spec.parameters.count
  unit: hour
- name: disk
  service: Storage
  sku: spec.parameters.diskSku
  regionDefault: brazilsouth
  quantityPerUnit: spec.parameters.diskGb
  unit: month
`

type fakePrices map[string]float64

func (f fakePrices) Price(_ context.Context, q PriceQuery) (Price, error) {
	if p, ok := f[q.SKU+"@"+q.Region]; ok {
		return Price{Unit: p, Currency: "USD"}, nil
	}
	return Price{}, ErrNoPrice
}

func TestCompare(t *testing.T) {
	cs, err := ParseComponents(annotation)
	if err != nil || len(cs) != 2 {
		t.Fatal(cs, err)
	}
	e := &Estimator{Prices: fakePrices{"Standard_D2as_v5@brazilsouth": 0.1, "Standard_D4as_v5@brazilsouth": 0.2, "P10@brazilsouth": 0.05}}
	cur := map[string]any{"parameters": map[string]any{"vmSize": "Standard_D2as_v5", "location": "brazilsouth", "count": int64(2), "diskSku": "P10", "diskGb": 128.0}}
	next := map[string]any{"parameters": map[string]any{"vmSize": "Standard_D4as_v5", "location": "brazilsouth", "count": int64(2), "diskSku": "P10", "diskGb": 128.0}}
	d := e.Compare(context.Background(), cs, cur, next)
	// vm: 0.1 × 2 × 730 = 146 ; disk: 0.05 × 128 = 6.4
	if d.Current.Monthly != 152.4 || d.New.Monthly != 298.4 || d.MonthlyDelta != 146 || d.Currency != "USD" || d.Partial {
		t.Fatalf("%+v", d)
	}
	if d.New.Lines[0].Quantity != 2 || d.New.Lines[1].Quantity != 128 {
		t.Fatalf("%+v", d.New.Lines)
	}

	unknown := map[string]any{"parameters": map[string]any{"vmSize": "Standard_X", "diskSku": "P10"}}
	d = e.Compare(context.Background(), cs, cur, unknown)
	if !d.Partial || len(d.New.Unknown) != 1 || d.New.Monthly != 0.05 {
		t.Fatalf("unknown SKU must be reported, region default used: %+v", d.New)
	}
	if cs, err := ParseComponents(""); err != nil || cs != nil {
		t.Fatal("empty annotation")
	}
	if _, err := ParseComponents("[bad"); err == nil {
		t.Fatal("bad annotation")
	}
}

func TestAzureRetailPrices(t *testing.T) {
	var hits int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		f := r.URL.Query().Get("$filter")
		if !strings.Contains(f, "armSkuName eq 'Standard_D4as_v5'") || !strings.Contains(f, "priceType eq 'Consumption'") || r.URL.Query().Get("currencyCode") != "'USD'" {
			t.Errorf("query %v", r.URL.Query())
		}
		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": []any{
				map[string]any{"retailPrice": 0.19, "currencyCode": "USD", "meterName": "D4as v5", "productName": "Virtual Machines Dasv5 Series", "unitOfMeasure": "1 Hour"},
			}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"Items": []any{
			map[string]any{"retailPrice": 0.03, "currencyCode": "USD", "meterName": "D4as v5 Spot", "productName": "Virtual Machines Dasv5 Series"},
			map[string]any{"retailPrice": 0.37, "currencyCode": "USD", "meterName": "D4as v5", "productName": "Virtual Machines Dasv5 Series Windows"},
			map[string]any{"retailPrice": 0.21, "currencyCode": "USD", "meterName": "D4as v5", "productName": "Virtual Machines Dasv5 Series"},
		}, "NextPageLink": srv.URL + "/?page=2&$filter=" + strings.ReplaceAll(f, " ", "%20") + "&currencyCode='USD'"})
	}))
	defer srv.Close()
	a := &Azure{BaseURL: srv.URL, HTTP: srv.Client()}
	q := PriceQuery{Service: "Virtual Machines", SKU: "Standard_D4as_v5", Region: "brazilsouth"}
	p, err := a.Price(context.Background(), q)
	if err != nil || p.Unit != 0.19 || p.UnitName != "1 Hour" {
		t.Fatalf("Linux on-demand minimum across pages, Spot/Windows ignored: %+v %v", p, err)
	}
	if _, err := a.Price(context.Background(), q); err != nil || hits != 2 {
		t.Fatalf("cached: hits=%d", hits)
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Items":[]}`))
	}))
	defer empty.Close()
	if _, err := (&Azure{BaseURL: empty.URL}).Price(context.Background(), PriceQuery{Service: "x", SKU: "y'z", Region: "r"}); !errors.Is(err, ErrNoPrice) {
		t.Fatalf("got %v", err)
	}
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }))
	defer down.Close()
	if _, err := (&Azure{BaseURL: down.URL}).Price(context.Background(), q); err == nil || errors.Is(err, ErrNoPrice) {
		t.Fatalf("got %v", err)
	}
}
