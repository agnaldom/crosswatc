// Package changes validates and analyses configuration changes before a Pull Request is opened (spec §40).
package changes

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/agnaldom/crosswatc/internal/cost"
	"github.com/agnaldom/crosswatc/internal/forms"
	"github.com/agnaldom/crosswatc/internal/redact"
)

// Impact of a single field change.
type Impact string

const (
	ImpactInPlace     Impact = "IN_PLACE"
	ImpactReplacement Impact = "REPLACEMENT"
)

// SpecChange is one field transition (spec §13 changes[]).
type SpecChange struct {
	Field  string `json:"field"`
	Path   string `json:"path"`
	From   any    `json:"from"`
	To     any    `json:"to"`
	Impact Impact `json:"impact"`
}

// Violation is a policy failure.
type Violation struct {
	Field   string `json:"field,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// PolicyResult aggregates violations.
type PolicyResult struct {
	Result     string      `json:"result"` // PASS | FAIL
	Violations []Violation `json:"violations"`
}

// ImpactSummary is the overall impact (spec §40).
type ImpactSummary struct {
	ReplacementRequired bool `json:"replacementRequired"`
	PossibleDowntime    bool `json:"possibleDowntime"`
	ApprovalRequired    bool `json:"approvalRequired"`
}

// Result is the preview response.
type Result struct {
	Valid    bool          `json:"valid"`
	Changes  []SpecChange  `json:"changes"`
	Policy   PolicyResult  `json:"policy"`
	Impact   ImpactSummary `json:"impact"`
	Warnings []string      `json:"warnings"`
	Cost     *cost.Diff    `json:"cost,omitempty"` // spec §60: current vs new monthly estimate
}

// Input is everything needed to preview a change.
type Input struct {
	Form          forms.Form
	CurrentSpec   map[string]any // Claim spec (applied state)
	Changes       map[string]any // key (relative to spec.parameters) → new value
	Environment   string
	ResourceState string
	ResourceName  string
	Owner         string
	Provider      string
	Kind          string
	Phase         string // "preview" | "submit" (policies may be stricter on submit)
	Justification string
	// CostComponents come from the XRD crosswatc.io/cost annotation (empty: not priced).
	CostComponents []cost.Component
}

// Policy adds organisation rules (naming, tags, security — spec §31 stage 3). Implemented by P4.5.
type Policy interface {
	Evaluate(ctx context.Context, in Input, changes []SpecChange) ([]Violation, error)
}

// Preview validates in against the form and policies.
func Preview(ctx context.Context, in Input, policies ...Policy) (Result, error) {
	res := Result{Changes: []SpecChange{}, Warnings: []string{}, Policy: PolicyResult{Violations: []Violation{}}}
	violate := func(field, rule, msg string) {
		res.Policy.Violations = append(res.Policy.Violations, Violation{Field: field, Rule: rule, Message: msg})
	}

	if in.ResourceState != "OPERATIONAL" {
		violate("", "resource-not-operational", fmt.Sprintf("Recurso precisa estar OPERATIONAL (atual: %s)", in.ResourceState))
	}
	if len(in.Changes) == 0 {
		violate("", "no-changes", "Nenhuma alteração informada")
	}

	params, _ := in.CurrentSpec["parameters"].(map[string]any)
	keys := make([]string, 0, len(in.Changes))
	for k := range in.Changes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		raw := in.Changes[key]
		f, ok := in.Form.FieldByKey(key)
		if !ok {
			violate(key, "unknown-field", "Campo inexistente no schema")
			continue
		}
		if !f.Editable {
			violate(key, "not-editable", "Campo não pode ser editado pelo portal")
			continue
		}
		to, err := coerce(f, raw)
		if err != nil {
			violate(key, "type", err.Error())
			continue
		}
		if msg := constraint(f, to); msg != "" {
			violate(key, "constraint", msg)
			continue
		}
		from, _ := forms.Lookup(params, key)
		if equal(from, to) {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: valor igual ao atual, ignorado", key))
			continue
		}
		impact := ImpactInPlace
		if !f.SafeInPlace {
			impact = ImpactReplacement
			res.Impact.ReplacementRequired, res.Impact.PossibleDowntime = true, true
		}
		if f.ApprovalRequiredIn(in.Environment) {
			res.Impact.ApprovalRequired = true
		}
		res.Changes = append(res.Changes, SpecChange{Field: key, Path: f.Path, From: from, To: to, Impact: impact})
	}
	if len(in.Changes) > 0 && len(res.Changes) == 0 && len(res.Policy.Violations) == 0 {
		violate("", "no-changes", "Todas as alterações são iguais aos valores atuais")
	}
	if res.Impact.ReplacementRequired {
		res.Warnings = append(res.Warnings, "A mudança exige recriação do recurso e pode causar indisponibilidade")
	}

	if len(res.Policy.Violations) == 0 {
		for _, p := range policies {
			vs, err := p.Evaluate(ctx, in, res.Changes)
			if err != nil {
				return Result{}, fmt.Errorf("changes: policy: %w", err)
			}
			res.Policy.Violations = append(res.Policy.Violations, vs...)
		}
	}
	res.Valid = len(res.Policy.Violations) == 0
	res.Policy.Result = "PASS"
	if !res.Valid {
		res.Policy.Result = "FAIL"
	}
	return res, nil
}

// Redacted returns a copy of cs with From/To masked for sensitive fields (password, token, keys…).
// Use it for anything persisted or shown (revision.changes, audit, PR body, API responses); the
// manifest edit itself needs the real value.
func Redacted(cs []SpecChange) []SpecChange {
	out := make([]SpecChange, len(cs))
	for i, c := range cs {
		key := c.Field
		if j := strings.LastIndex(key, "."); j >= 0 {
			key = key[j+1:]
		}
		if redact.Sensitive(key) {
			c.From, c.To = redact.Placeholder, redact.Placeholder
		}
		out[i] = c
	}
	return out
}

// ProposedSpec returns a deep copy of current with the accepted changes applied at their paths.
func ProposedSpec(current map[string]any, cs []SpecChange) map[string]any {
	out := deepCopy(current)
	for _, c := range cs {
		parts := strings.Split(strings.TrimPrefix(c.Path, "spec."), ".")
		m := out
		for _, p := range parts[:len(parts)-1] {
			next, ok := m[p].(map[string]any)
			if !ok {
				next = map[string]any{}
				m[p] = next
			}
			m = next
		}
		m[parts[len(parts)-1]] = c.To
	}
	return out
}

func deepCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if mm, ok := v.(map[string]any); ok {
			out[k] = deepCopy(mm)
		} else {
			out[k] = v
		}
	}
	return out
}

// coerce converts JSON input to the field type (forms send numbers as strings or float64).
func coerce(f forms.Field, v any) (any, error) {
	switch f.Type {
	case "integer":
		switch n := v.(type) {
		case float64:
			if n != float64(int64(n)) {
				return nil, fmt.Errorf("deve ser inteiro")
			}
			return int64(n), nil
		case int64:
			return n, nil
		case int:
			return int64(n), nil
		case string:
			i, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("deve ser inteiro")
			}
			return i, nil
		}
		return nil, fmt.Errorf("deve ser inteiro")
	case "number":
		switch n := v.(type) {
		case float64:
			return n, nil
		case int64:
			return float64(n), nil
		case string:
			x, err := strconv.ParseFloat(n, 64)
			if err != nil {
				return nil, fmt.Errorf("deve ser numérico")
			}
			return x, nil
		}
		return nil, fmt.Errorf("deve ser numérico")
	case "boolean":
		switch b := v.(type) {
		case bool:
			return b, nil
		case string:
			x, err := strconv.ParseBool(b)
			if err != nil {
				return nil, fmt.Errorf("deve ser booleano")
			}
			return x, nil
		}
		return nil, fmt.Errorf("deve ser booleano")
	case "string", "":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("deve ser texto")
		}
		return s, nil
	}
	return nil, fmt.Errorf("tipo %q não suportado no portal", f.Type)
}

func constraint(f forms.Field, v any) string {
	if len(f.Enum) > 0 && !slices.ContainsFunc(f.Enum, func(e any) bool { return equal(e, v) }) {
		return fmt.Sprintf("valor deve ser um de %v", f.Enum)
	}
	if n, ok := toFloat(v); ok {
		if f.Minimum != nil && n < *f.Minimum {
			return fmt.Sprintf("mínimo %v", *f.Minimum)
		}
		if f.Maximum != nil && n > *f.Maximum {
			return fmt.Sprintf("máximo %v", *f.Maximum)
		}
	}
	if s, ok := v.(string); ok {
		if f.Required && s == "" {
			return "obrigatório"
		}
		if f.MinLength != nil && int64(len(s)) < *f.MinLength {
			return fmt.Sprintf("mínimo %d caracteres", *f.MinLength)
		}
		if f.MaxLength != nil && int64(len(s)) > *f.MaxLength {
			return fmt.Sprintf("máximo %d caracteres", *f.MaxLength)
		}
		if f.Pattern != "" {
			re, err := regexp.Compile(f.Pattern)
			if err != nil || !re.MatchString(s) {
				return fmt.Sprintf("formato inválido (%s)", f.Pattern)
			}
		}
	}
	return ""
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}

// equal compares values across JSON/YAML numeric representations.
func equal(a, b any) bool {
	if fa, ok := toFloat(a); ok {
		if fb, ok := toFloat(b); ok {
			return fa == fb
		}
	}
	return reflect.DeepEqual(a, b)
}
