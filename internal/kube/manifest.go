package kube

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/agnaldom/crosswatc/internal/redact"
)

// ErrForbiddenKind protects kinds whose content must never leave the cluster (§49).
var ErrForbiddenKind = errors.New("kube: kind not viewable")

var forbiddenKinds = map[string]bool{"Secret": true, "ConfigMap": true, "ServiceAccount": true}

// Manifest returns the live object as sanitized YAML: no managedFields or last-applied annotation,
// and values of sensitive keys redacted in spec/status.
func (c *Client) Manifest(ctx context.Context, ref Ref) ([]byte, error) {
	if forbiddenKinds[ref.Kind] {
		return nil, ErrForbiddenKind
	}
	if c.Mapper == nil {
		return nil, errors.New("kube: no REST mapper")
	}
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("kube: apiVersion %q: %w", ref.APIVersion, err)
	}
	m, err := c.Mapper.RESTMapping(schema.GroupKind{Group: gv.Group, Kind: ref.Kind}, gv.Version)
	if err != nil {
		return nil, fmt.Errorf("kube: resolve %s %s: %w", ref.APIVersion, ref.Kind, err)
	}
	ns := ""
	if m.Scope.Name() == meta.RESTScopeNameNamespace {
		ns = ref.Namespace
	}
	u, err := c.Get(ctx, m.Resource, ns, ref.Name)
	if err != nil {
		return nil, err
	}
	return SanitizedYAML(u)
}

// SanitizedYAML strips noise and secret material from an object before display.
func SanitizedYAML(u *unstructured.Unstructured) ([]byte, error) {
	o := u.DeepCopy().Object
	unstructured.RemoveNestedField(o, "metadata", "managedFields")
	unstructured.RemoveNestedField(o, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
	for _, k := range []string{"spec", "status", "data", "stringData"} {
		if m, ok := o[k].(map[string]any); ok {
			o[k] = redact.Map(m)
		} else if _, present := o[k]; present && (k == "data" || k == "stringData") {
			o[k] = redact.Placeholder
		}
	}
	return yaml.Marshal(o)
}
