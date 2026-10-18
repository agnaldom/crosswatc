// Package redact removes sensitive values before data leaves the cluster boundary (spec §49).
package redact

import "regexp"

// Placeholder replaces sensitive values.
const Placeholder = "[REDACTED]"

var sensitive = regexp.MustCompile(`(?i)(password|passwd|secret|token|credential|private.?key|access.?key|api.?key|connection.?string)`)

// Sensitive reports whether a key name suggests secret material.
// Kubernetes references to secrets (…SecretRef, …SecretName) are names, not values, and are kept.
func Sensitive(key string) bool {
	if refLike.MatchString(key) {
		return false
	}
	return sensitive.MatchString(key)
}

var refLike = regexp.MustCompile(`(?i)(secret|credential)s?(ref|name|selector|namespace)$|^writeConnectionSecret`)

// Map returns a deep copy of m with sensitive values replaced.
func Map(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if Sensitive(k) {
			if _, isMap := v.(map[string]any); !isMap {
				out[k] = Placeholder
				continue
			}
		}
		out[k] = value(v)
	}
	return out
}

func value(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return Map(t)
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = value(x)
		}
		return out
	default:
		return v
	}
}
