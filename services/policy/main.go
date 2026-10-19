// Command crosswatc-policy evaluates Crosswatc Rego policies on a Claim manifest change.
// Intended for the CI of the claims (GitOps) repository, so changes made outside the portal
// follow the same rules (spec §31 stage 3).
//
//	crosswatc-policy -base old.yaml -head new.yaml -name pgsql-payments -environment prd [-policies dir]
//
// Exit codes: 0 no violations, 1 violations, 2 usage or evaluation error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/drift"
	"github.com/agnaldom/crosswatc/internal/gitops"
	"github.com/agnaldom/crosswatc/internal/policy"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	fs := flag.NewFlagSet("crosswatc-policy", flag.ContinueOnError)
	base := fs.String("base", "", "manifest before the change (omit for new Claims)")
	head := fs.String("head", "", "manifest after the change")
	name := fs.String("name", "", "Claim metadata.name")
	env := fs.String("environment", "", "environment label (dev, hlg, prd…)")
	owner := fs.String("owner", "", "owning squad")
	kind := fs.String("kind", "", "Claim kind")
	dir := fs.String("policies", "", "directory with extra *.rego (package crosswatc)")
	asJSON := fs.Bool("json", false, "print violations as JSON")
	if err := fs.Parse(args); err != nil || *head == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "usage: crosswatc-policy -head new.yaml -name <claim> [-base old.yaml] [-environment prd] [-policies dir]")
		return 2
	}
	headParams, err := params(*head, *name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	baseParams := map[string]any{}
	if *base != "" {
		if baseParams, err = params(*base, *name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	var cs []changes.SpecChange
	for _, d := range drift.Compare(headParams, baseParams) {
		cs = append(cs, changes.SpecChange{Field: d.Field, Path: gitopsPath(d.Field), From: d.Observed, To: d.Desired, Impact: "UNKNOWN"})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Field < cs[j].Field })

	ctx := context.Background()
	engine, err := policy.Load(ctx, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	in := changes.Input{ResourceName: *name, Environment: *env, Owner: *owner, Kind: *kind, Phase: "ci",
		CurrentSpec: map[string]any{"parameters": baseParams}}
	vs, err := engine.Evaluate(ctx, in, cs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"changes": cs, "violations": vs})
	} else {
		for _, v := range vs {
			fmt.Printf("DENY %-28s %-30s %s\n", v.Rule, v.Field, v.Message)
		}
		fmt.Printf("%d change(s), %d violation(s)\n", len(cs), len(vs))
	}
	if len(vs) > 0 {
		return 1
	}
	return 0
}

func params(path, name string) (map[string]any, error) {
	b, err := os.ReadFile(filepath.Clean(path)) //nolint:gosec // CLI: reading the manifest the caller points to is the purpose
	if err != nil {
		return nil, err
	}
	return gitops.ClaimParameters(b, name)
}

func gitopsPath(field string) string { return "spec.parameters." + field }
