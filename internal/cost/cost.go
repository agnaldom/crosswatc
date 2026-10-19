// Package cost estimates monthly cost before and after a change (spec §60 Costs).
//
// Resource types opt in with the XRD annotation crosswatc.io/cost (YAML/JSON list):
//
//   - name: vm
//     service: Virtual Machines          # Azure Retail Prices serviceName
//     sku: spec.parameters.vmSize        # path read from the Claim (→ armSkuName)
//     region: spec.parameters.location   # path; regionDefault when missing
//     regionDefault: brazilsouth
//     quantity: spec.parameters.count    # optional path, default 1
//     unit: hour                         # hour (× hoursPerMonth) | month | unit
//     hoursPerMonth: 730
//     quantityPerUnit: spec.parameters.diskGb  # optional multiplier (e.g. GB for per-GB meters)
package cost

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"
)

// Annotation on the XRD.
const Annotation = "crosswatc.io/cost"

// Component maps Claim parameters to a priced meter.
type Component struct {
	Name            string  `json:"name"`
	Service         string  `json:"service"`
	SKU             string  `json:"sku"`
	Region          string  `json:"region,omitempty"`
	RegionDefault   string  `json:"regionDefault,omitempty"`
	Quantity        string  `json:"quantity,omitempty"`
	QuantityPerUnit string  `json:"quantityPerUnit,omitempty"`
	Unit            string  `json:"unit,omitempty"`
	HoursPerMonth   float64 `json:"hoursPerMonth,omitempty"`
}

// ParseComponents reads the annotation; empty means the type is not priced.
func ParseComponents(raw string) ([]Component, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var cs []Component
	if err := yaml.Unmarshal([]byte(raw), &cs); err != nil {
		return nil, fmt.Errorf("cost: %s annotation: %w", Annotation, err)
	}
	return cs, nil
}

// PriceQuery identifies a meter.
type PriceQuery struct {
	Service string
	SKU     string
	Region  string
}

// Price is a unit price.
type Price struct {
	Unit     float64
	Currency string
	UnitName string // e.g. "1 Hour"
}

// Prices resolves meter prices (Azure Retail Prices, a fake in tests…).
type Prices interface {
	Price(ctx context.Context, q PriceQuery) (Price, error)
}

// Line is one component estimate.
type Line struct {
	Component string  `json:"component"`
	SKU       string  `json:"sku"`
	Region    string  `json:"region"`
	UnitPrice float64 `json:"unitPrice"`
	Quantity  float64 `json:"quantity"`
	Monthly   float64 `json:"monthly"`
}

// Estimate is the monthly cost of one configuration.
type Estimate struct {
	Monthly  float64  `json:"monthly"`
	Currency string   `json:"currency"`
	Lines    []Line   `json:"lines"`
	Unknown  []string `json:"unknown,omitempty"` // components that could not be priced
}

// Diff compares current and proposed configurations.
type Diff struct {
	Current      Estimate `json:"current"`
	New          Estimate `json:"new"`
	MonthlyDelta float64  `json:"monthlyDelta"`
	Currency     string   `json:"currency"`
	Partial      bool     `json:"partial"` // some component could not be priced
}

// Estimator computes estimates.
type Estimator struct {
	Prices Prices
}

// Estimate prices a Claim spec (with "parameters").
func (e *Estimator) Estimate(ctx context.Context, cs []Component, spec map[string]any) Estimate {
	out := Estimate{Lines: []Line{}}
	for _, c := range cs {
		sku := str(spec, c.SKU)
		region := str(spec, c.Region)
		if region == "" {
			region = c.RegionDefault
		}
		if sku == "" || region == "" {
			out.Unknown = append(out.Unknown, c.Name+": sku/região ausente")
			continue
		}
		p, err := e.Prices.Price(ctx, PriceQuery{Service: c.Service, SKU: sku, Region: region})
		if err != nil {
			out.Unknown = append(out.Unknown, fmt.Sprintf("%s: %v", c.Name, err))
			continue
		}
		qty := num(spec, c.Quantity, 1)
		if c.QuantityPerUnit != "" {
			qty *= num(spec, c.QuantityPerUnit, 1)
		}
		monthly := p.Unit * qty
		switch strings.ToLower(c.Unit) {
		case "", "hour":
			h := c.HoursPerMonth
			if h <= 0 {
				h = 730
			}
			monthly *= h
		case "month", "unit":
		}
		monthly = round(monthly)
		if out.Currency == "" {
			out.Currency = p.Currency
		}
		out.Lines = append(out.Lines, Line{Component: c.Name, SKU: sku, Region: region, UnitPrice: p.Unit, Quantity: qty, Monthly: monthly})
		out.Monthly = round(out.Monthly + monthly)
	}
	return out
}

// Compare estimates current vs proposed specs.
func (e *Estimator) Compare(ctx context.Context, cs []Component, current, proposed map[string]any) Diff {
	cur, next := e.Estimate(ctx, cs, current), e.Estimate(ctx, cs, proposed)
	cur2 := cur.Currency
	if cur2 == "" {
		cur2 = next.Currency
	}
	return Diff{Current: cur, New: next, MonthlyDelta: round(next.Monthly - cur.Monthly), Currency: cur2,
		Partial: len(cur.Unknown) > 0 || len(next.Unknown) > 0}
}

func round(f float64) float64 { return math.Round(f*100) / 100 }

// lookup reads a dot path (e.g. spec.parameters.vmSize) from a Claim spec map rooted at "spec".
func lookup(spec map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	cur := any(map[string]any{"spec": spec})
	for _, p := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = m[p]; !ok {
			return nil, false
		}
	}
	return cur, true
}

func str(spec map[string]any, path string) string {
	v, ok := lookup(spec, path)
	if !ok {
		return ""
	}
	return fmt.Sprint(v)
}

func num(spec map[string]any, path string, def float64) float64 {
	v, ok := lookup(spec, path)
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f
		}
	}
	return def
}
