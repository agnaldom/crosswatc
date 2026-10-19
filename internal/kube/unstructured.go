package kube

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/agnaldom/crosswatc/internal/domain"
)

func str(u map[string]any, path ...string) string {
	v, _, _ := unstructured.NestedString(u, path...)
	return v
}

func boolean(u map[string]any, path ...string) bool {
	v, _, _ := unstructured.NestedBool(u, path...)
	return v
}

func slice(u map[string]any, path ...string) []any {
	v, _, _ := unstructured.NestedSlice(u, path...)
	return v
}

func obj(u map[string]any, path ...string) map[string]any {
	v, _, _ := unstructured.NestedMap(u, path...)
	return v
}

// Conditions extracts status.conditions.
func Conditions(u *unstructured.Unstructured) domain.Conditions {
	var out domain.Conditions
	for _, raw := range slice(u.Object, "status", "conditions") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, domain.Condition{
			Type: str(m, "type"), Status: str(m, "status"), Reason: str(m, "reason"), Message: str(m, "message"),
			LastTransitionTime: str(m, "lastTransitionTime"),
		})
	}
	return out
}
