package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/tester"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/policy/defaults"
)

// TestRegoUnitTests runs the *_test.rego files shipped with the defaults.
func TestRegoUnitTests(t *testing.T) {
	ctx := context.Background()
	src := map[string]string{}
	if err := collectAll(src); err != nil {
		t.Fatal(err)
	}
	mods := map[string]*ast.Module{}
	for name, s := range src {
		m, err := ast.ParseModule(name, s)
		if err != nil {
			t.Fatal(err)
		}
		mods[name] = m
	}
	results, err := tester.NewRunner().Run(ctx, mods)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for res := range results {
		n++
		if res.Fail || res.Error != nil {
			t.Errorf("%s: fail=%v err=%v", res.Name, res.Fail, res.Error)
		}
	}
	if n < 6 {
		t.Fatalf("expected the rego test suite to run, got %d tests", n)
	}
}

func collectAll(out map[string]string) error {
	entries, err := defaults.FS.ReadDir(".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		b, err := defaults.FS.ReadFile(e.Name())
		if err != nil {
			return err
		}
		out[e.Name()] = string(b)
	}
	return nil
}

func input(env string, cs ...changes.SpecChange) changes.Input {
	return changes.Input{ResourceName: "pgsql-payments", Owner: "payments", Environment: env, Provider: "azure",
		Kind: "PostgreSQLInstance", ResourceState: "OPERATIONAL",
		CurrentSpec: map[string]any{"parameters": map[string]any{"adminPassword": "x", "vmSize": "D2"}}}
}

func TestEngine(t *testing.T) {
	e, err := Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Modules()) != 1 {
		t.Fatalf("modules %v (test files must be skipped)", e.Modules())
	}
	ok := []changes.SpecChange{{Field: "vmSize", Path: "spec.parameters.vmSize", From: "D2", To: "D4", Impact: changes.ImpactInPlace}}
	vs, err := e.Evaluate(context.Background(), input("dev"), ok)
	if err != nil || len(vs) != 0 {
		t.Fatalf("%+v %v", vs, err)
	}

	repl := []changes.SpecChange{{Field: "diskGb", From: 64, To: 128, Impact: changes.ImpactReplacement},
		{Field: "network.publicNetworkAccess", From: "Disabled", To: "Enabled", Impact: changes.ImpactInPlace}}
	in := input("prd")
	in.Phase, in.Justification = "submit", "curta"
	vs, err = e.Evaluate(context.Background(), in, repl)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, v := range vs {
		got[v.Rule] = v.Field
	}
	if _, ok := got["prd/no-replacement"]; !ok || got["security/no-public-access"] != "network.publicNetworkAccess" || len(got) != 3 {
		t.Fatalf("%+v", vs)
	}

	doc := Document(in, nil)
	if doc["current"].(map[string]any)["adminPassword"] != "[REDACTED]" {
		t.Fatal("policies must never see secret values")
	}
}

func TestCustomPolicyDir(t *testing.T) {
	dir := t.TempDir()
	custom := `package crosswatc

import rego.v1

deny contains "VM D8 não aprovada" if {
	some c in input.changes
	c.to == "D8"
}
`
	if err := os.WriteFile(filepath.Join(dir, "org.rego"), []byte(custom), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := Load(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	vs, err := e.EvaluateDocument(context.Background(), map[string]any{
		"resource": map[string]any{"name": "vm-aa", "environment": "dev"},
		"changes":  []any{map[string]any{"field": "vmSize", "to": "D8", "impact": "IN_PLACE"}},
	})
	if err != nil || len(vs) != 1 || vs[0].Message != "VM D8 não aprovada" || vs[0].Rule != "policy" {
		t.Fatalf("%+v %v", vs, err)
	}

	bad := t.TempDir()
	_ = os.WriteFile(filepath.Join(bad, "x.rego"), []byte("package crosswatc\ndeny contains"), 0o600)
	if _, err := Load(context.Background(), bad); err == nil {
		t.Fatal("invalid rego must fail to load")
	}
}
