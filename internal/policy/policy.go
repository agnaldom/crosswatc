// Package policy evaluates organisation rules written in Rego (OPA) against change requests (spec §40, §58).
package policy

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/policy/defaults"
	"github.com/agnaldom/crosswatc/internal/redact"
)

// Query collects violations. Custom files must declare `package crosswatc`; their `deny contains …`
// rules merge with the built-in ones.
const Query = "data.crosswatc.deny"

// Engine is a prepared Rego query. Safe for concurrent use.
type Engine struct {
	query   rego.PreparedEvalQuery
	modules []string
}

// Load compiles the built-in policies plus every *.rego under dir (dir may be empty).
// Test files (*_test.rego) are skipped.
func Load(ctx context.Context, dir string) (*Engine, error) {
	mods := map[string]string{}
	if err := collect(defaults.FS, ".", "builtin/", mods); err != nil {
		return nil, err
	}
	if dir != "" {
		if err := collect(os.DirFS(dir), ".", dir+"/", mods); err != nil {
			return nil, fmt.Errorf("policy: load %s: %w", dir, err)
		}
	}
	opts := []func(*rego.Rego){rego.Query(Query)}
	names := make([]string, 0, len(mods))
	for name, src := range mods {
		opts = append(opts, rego.Module(name, src))
		names = append(names, name)
	}
	sort.Strings(names)
	q, err := rego.New(opts...).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("policy: compile: %w", err)
	}
	return &Engine{query: q, modules: names}, nil
}

func collect(fsys fs.FS, root, prefix string, out map[string]string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".rego") || strings.HasSuffix(path, "_test.rego") {
			return err
		}
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(prefix+path)] = string(b)
		return nil
	})
}

// Modules lists the loaded policy files.
func (e *Engine) Modules() []string { return e.modules }

// Document builds the Rego input (exported for the CLI and tests).
func Document(in changes.Input, cs []changes.SpecChange) map[string]any {
	params, _ := in.CurrentSpec["parameters"].(map[string]any)
	list := make([]map[string]any, len(cs))
	for i, c := range cs {
		list[i] = map[string]any{"field": c.Field, "path": c.Path, "from": c.From, "to": c.To, "impact": string(c.Impact)}
	}
	phase := in.Phase
	if phase == "" {
		phase = "preview"
	}
	return map[string]any{
		"phase": phase,
		"resource": map[string]any{"name": in.ResourceName, "owner": in.Owner, "environment": in.Environment,
			"provider": in.Provider, "kind": in.Kind, "state": in.ResourceState},
		"changes":       list,
		"current":       redact.Map(params),
		"justification": in.Justification,
	}
}

// Evaluate implements changes.Policy.
func (e *Engine) Evaluate(ctx context.Context, in changes.Input, cs []changes.SpecChange) ([]changes.Violation, error) {
	return e.EvaluateDocument(ctx, Document(in, cs))
}

// EvaluateDocument runs the query on a prepared input document.
func (e *Engine) EvaluateDocument(ctx context.Context, doc map[string]any) ([]changes.Violation, error) {
	rs, err := e.query.Eval(ctx, rego.EvalInput(doc))
	if err != nil {
		return nil, fmt.Errorf("policy: eval: %w", err)
	}
	var out []changes.Violation
	for _, r := range rs {
		for _, ex := range r.Expressions {
			items, _ := ex.Value.([]any)
			for _, it := range items {
				out = append(out, toViolation(it))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rule != out[j].Rule {
			return out[i].Rule < out[j].Rule
		}
		return out[i].Field < out[j].Field
	})
	return out, nil
}

func toViolation(v any) changes.Violation {
	switch t := v.(type) {
	case string:
		return changes.Violation{Rule: "policy", Message: t}
	case map[string]any:
		s := func(k string) string { x, _ := t[k].(string); return x }
		rule := s("rule")
		if rule == "" {
			rule = "policy"
		}
		return changes.Violation{Rule: rule, Field: s("field"), Message: s("message")}
	}
	return changes.Violation{Rule: "policy", Message: fmt.Sprint(v)}
}
