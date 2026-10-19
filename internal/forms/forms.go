// Package forms derives editable form schemas from XRD OpenAPI schemas (spec §38).
//
// UX metadata may come from two places, the annotation taking precedence:
//   - XRD annotation crosswatc.io/ui-metadata: JSON or YAML map keyed by field path
//     ("spec.parameters.vmSize") with displayName, editable, order, safeInPlace, approvalRequired;
//   - OpenAPI vendor extension x-crosswatc on the property, with the same keys.
//
// Fields are read-only unless metadata marks them editable.
package forms

import (
	"fmt"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/agnaldom/crosswatc/internal/kube"
)

// MetadataAnnotation holds UX metadata on the XRD.
const MetadataAnnotation = "crosswatc.io/ui-metadata"

// ParametersPath is the root of user-facing parameters.
const ParametersPath = "spec.parameters"

// Metadata is the optional UX metadata of one field.
type Metadata struct {
	DisplayName      string          `json:"displayName,omitempty"`
	Editable         bool            `json:"editable,omitempty"`
	Order            int             `json:"order,omitempty"`
	SafeInPlace      *bool           `json:"safeInPlace,omitempty"`
	ApprovalRequired map[string]bool `json:"approvalRequired,omitempty"`
	Group            string          `json:"group,omitempty"`
}

// Field is one leaf of the form.
type Field struct {
	Key              string          `json:"key"`  // relative to spec.parameters, dot separated
	Path             string          `json:"path"` // full path, e.g. spec.parameters.vmSize
	Type             string          `json:"type"` // string | integer | number | boolean | array | object
	Title            string          `json:"title"`
	Description      string          `json:"description,omitempty"`
	Required         bool            `json:"required"`
	Editable         bool            `json:"editable"`
	Order            int             `json:"order"`
	Enum             []any           `json:"enum,omitempty"`
	Default          any             `json:"default,omitempty"`
	Minimum          *float64        `json:"minimum,omitempty"`
	Maximum          *float64        `json:"maximum,omitempty"`
	MinLength        *int64          `json:"minLength,omitempty"`
	MaxLength        *int64          `json:"maxLength,omitempty"`
	Pattern          string          `json:"pattern,omitempty"`
	Immutable        bool            `json:"immutable,omitempty"` // x-kubernetes-validations self == oldSelf
	SafeInPlace      bool            `json:"safeInPlace"`
	ApprovalRequired map[string]bool `json:"approvalRequired,omitempty"`
	Group            string          `json:"group,omitempty"`
}

// Form is the UI schema of a resource type.
type Form struct {
	Kind    string  `json:"kind"`
	Version string  `json:"version"`
	Fields  []Field `json:"fields"`
}

// FieldByKey returns the field for a parameters key.
func (f Form) FieldByKey(key string) (Field, bool) {
	for _, fl := range f.Fields {
		if fl.Key == key {
			return fl, true
		}
	}
	return Field{}, false
}

// Build derives the form of an XRD. Only spec.parameters is exposed.
func Build(x kube.XRD) (Form, error) {
	meta, err := parseAnnotation(x.Annotations[MetadataAnnotation])
	if err != nil {
		return Form{}, fmt.Errorf("forms: %s annotation: %w", MetadataAnnotation, err)
	}
	f := Form{Kind: userKind(x), Version: x.Version(), Fields: []Field{}}
	params := nested(x.Schema(""), "properties", "spec", "properties", "parameters")
	if params == nil {
		return f, nil
	}
	walk(params, "", nil, meta, &f.Fields)
	slices.SortStableFunc(f.Fields, func(a, b Field) int {
		if a.Order != b.Order {
			return a.Order - b.Order
		}
		return strings.Compare(a.Key, b.Key)
	})
	return f, nil
}

func userKind(x kube.XRD) string {
	if x.ClaimKind != "" {
		return x.ClaimKind
	}
	return x.Kind
}

func parseAnnotation(raw string) (map[string]Metadata, error) {
	out := map[string]Metadata{}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	if err := yaml.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func walk(schema map[string]any, prefix string, required []string, meta map[string]Metadata, out *[]Field) {
	props, _ := schema["properties"].(map[string]any)
	req := stringSlice(schema["required"])
	for name, raw := range props {
		prop, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		typ, _ := prop["type"].(string)
		if typ == "object" && prop["properties"] != nil {
			walk(prop, key, req, meta, out)
			continue
		}
		*out = append(*out, field(key, typ, prop, slices.Contains(req, name), meta))
	}
}

func field(key, typ string, prop map[string]any, required bool, meta map[string]Metadata) Field {
	path := ParametersPath + "." + key
	fl := Field{Key: key, Path: path, Type: typ, Title: key, Required: required, Order: 1000}
	fl.Description, _ = prop["description"].(string)
	fl.Enum, _ = prop["enum"].([]any)
	fl.Default = prop["default"]
	fl.Pattern, _ = prop["pattern"].(string)
	fl.Minimum, fl.Maximum = num(prop["minimum"]), num(prop["maximum"])
	fl.MinLength, fl.MaxLength = integer(prop["minLength"]), integer(prop["maxLength"])
	for _, v := range sliceOf(prop["x-kubernetes-validations"]) {
		if m, ok := v.(map[string]any); ok && strings.ReplaceAll(fmt.Sprint(m["rule"]), " ", "") == "self==oldSelf" {
			fl.Immutable = true
		}
	}

	md, ok := meta[path]
	if !ok {
		md, ok = meta[key]
	}
	if !ok {
		if ext, isMap := prop["x-crosswatc"].(map[string]any); isMap {
			b, _ := yaml.Marshal(ext)
			_ = yaml.Unmarshal(b, &md)
		}
	}
	if md.DisplayName != "" {
		fl.Title = md.DisplayName
	}
	if md.Order != 0 {
		fl.Order = md.Order
	}
	fl.Editable = md.Editable && !fl.Immutable
	fl.SafeInPlace = md.SafeInPlace == nil || *md.SafeInPlace // default: assume in-place unless told otherwise
	if fl.Immutable {
		fl.SafeInPlace = false
	}
	fl.ApprovalRequired, fl.Group = md.ApprovalRequired, md.Group
	return fl
}

// ApprovalRequiredIn reports whether changing the field needs approval in env (default: true outside dev).
func (f Field) ApprovalRequiredIn(env string) bool {
	if v, ok := f.ApprovalRequired[env]; ok {
		return v
	}
	return env != "dev"
}

func nested(m map[string]any, path ...string) map[string]any {
	cur := m
	for _, p := range path {
		next, ok := cur[p].(map[string]any)
		if !ok {
			return nil
		}
		cur = next
	}
	return cur
}

func stringSlice(v any) []string {
	var out []string
	for _, x := range sliceOf(v) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func sliceOf(v any) []any {
	s, _ := v.([]any)
	return s
}

func num(v any) *float64 {
	switch n := v.(type) {
	case float64:
		return &n
	case int64:
		f := float64(n)
		return &f
	case int:
		f := float64(n)
		return &f
	}
	return nil
}

func integer(v any) *int64 {
	if f := num(v); f != nil {
		i := int64(*f)
		return &i
	}
	return nil
}

// Values extracts current parameter values (flattened by key) from a Claim spec.
func Values(form Form, spec map[string]any) map[string]any {
	out := map[string]any{}
	params, _ := spec["parameters"].(map[string]any)
	for _, f := range form.Fields {
		if v, ok := Lookup(params, f.Key); ok {
			out[f.Key] = v
		}
	}
	return out
}

// Lookup reads a dot-separated key from a nested map.
func Lookup(m map[string]any, key string) (any, bool) {
	cur := any(m)
	for _, p := range strings.Split(key, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = mm[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}
