package changes

import (
	"context"
	"errors"
	"testing"

	"github.com/agnaldom/crosswatc/internal/forms"
)

func f64(v float64) *float64 { return &v }

var form = forms.Form{Kind: "VirtualMachine", Fields: []forms.Field{
	{Key: "vmSize", Path: "spec.parameters.vmSize", Type: "string", Editable: true, SafeInPlace: true,
		Enum: []any{"Standard_D2as_v5", "Standard_D4as_v5"}, ApprovalRequired: map[string]bool{"dev": false, "prd": true}},
	{Key: "diskGb", Path: "spec.parameters.diskGb", Type: "integer", Editable: true, SafeInPlace: false, Minimum: f64(32), Maximum: f64(1024)},
	{Key: "subnet", Path: "spec.parameters.network.subnet", Type: "string", Editable: true, SafeInPlace: true, Pattern: "^snet-"},
	{Key: "location", Path: "spec.parameters.location", Type: "string", Editable: false},
}}

func input(env string, ch map[string]any) Input {
	return Input{Form: form, Environment: env, ResourceState: "OPERATIONAL", Changes: ch,
		CurrentSpec: map[string]any{"parameters": map[string]any{"vmSize": "Standard_D2as_v5", "diskGb": int64(64), "location": "brazilsouth"}}}
}

func TestPreviewInPlace(t *testing.T) {
	r, err := Preview(context.Background(), input("dev", map[string]any{"vmSize": "Standard_D4as_v5"}))
	if err != nil {
		t.Fatal(err)
	}
	if !r.Valid || r.Policy.Result != "PASS" || len(r.Changes) != 1 || r.Impact != (ImpactSummary{}) {
		t.Fatalf("%+v", r)
	}
	c := r.Changes[0]
	if c.From != "Standard_D2as_v5" || c.To != "Standard_D4as_v5" || c.Impact != ImpactInPlace {
		t.Fatalf("%+v", c)
	}
	r, _ = Preview(context.Background(), input("prd", map[string]any{"vmSize": "Standard_D4as_v5"}))
	if !r.Impact.ApprovalRequired {
		t.Fatal("prd requires approval")
	}
}

func TestPreviewReplacement(t *testing.T) {
	r, _ := Preview(context.Background(), input("dev", map[string]any{"diskGb": "128"}))
	if !r.Valid || !r.Impact.ReplacementRequired || !r.Impact.PossibleDowntime || r.Changes[0].To != int64(128) || len(r.Warnings) != 1 {
		t.Fatalf("%+v", r)
	}
	if r.Impact.ApprovalRequired {
		t.Fatal("without metadata, dev does not require approval")
	}
}

func TestPreviewViolations(t *testing.T) {
	cases := map[string]struct {
		ch   map[string]any
		rule string
	}{
		"unknown":      {map[string]any{"nope": 1}, "unknown-field"},
		"not editable": {map[string]any{"location": "eastus"}, "not-editable"},
		"enum":         {map[string]any{"vmSize": "Standard_D8"}, "constraint"},
		"min":          {map[string]any{"diskGb": 8}, "constraint"},
		"integer":      {map[string]any{"diskGb": 40.5}, "type"},
		"pattern":      {map[string]any{"subnet": "vnet-a"}, "constraint"},
		"wrong type":   {map[string]any{"vmSize": 4}, "type"},
		"empty":        {map[string]any{}, "no-changes"},
		"no-op":        {map[string]any{"diskGb": 64.0}, "no-changes"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := Preview(context.Background(), input("dev", tc.ch))
			if err != nil {
				t.Fatal(err)
			}
			if r.Valid || r.Policy.Result != "FAIL" || r.Policy.Violations[0].Rule != tc.rule {
				t.Fatalf("%+v", r.Policy)
			}
		})
	}
	in := input("dev", map[string]any{"vmSize": "Standard_D4as_v5"})
	in.ResourceState = "UPDATING"
	if r, _ := Preview(context.Background(), in); r.Valid || r.Policy.Violations[0].Rule != "resource-not-operational" {
		t.Fatalf("%+v", r.Policy)
	}
}

type denyPolicy struct{ err error }

func (d denyPolicy) Evaluate(context.Context, Input, []SpecChange) ([]Violation, error) {
	return []Violation{{Rule: "org/naming", Message: "nope"}}, d.err
}

func TestPreviewPolicies(t *testing.T) {
	r, err := Preview(context.Background(), input("dev", map[string]any{"vmSize": "Standard_D4as_v5"}), denyPolicy{})
	if err != nil || r.Valid || r.Policy.Violations[0].Rule != "org/naming" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := Preview(context.Background(), input("dev", map[string]any{"vmSize": "Standard_D4as_v5"}), denyPolicy{errors.New("x")}); err == nil {
		t.Fatal("policy error must propagate")
	}
	// schema violations short-circuit org policies
	r, _ = Preview(context.Background(), input("dev", map[string]any{"nope": 1}), denyPolicy{})
	if len(r.Policy.Violations) != 1 {
		t.Fatalf("%+v", r.Policy)
	}
}
