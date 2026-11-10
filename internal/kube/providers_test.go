package kube

import (
	"context"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestProviderConfigsAndCreateOnlyInstall(t *testing.T) {
	ctx := context.Background()
	provider := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "pkg.crossplane.io/v1", "kind": "Provider", "metadata": map[string]any{"name": "existing-family"},
		"spec":   map[string]any{"package": "xpkg.upbound.io/upbound/provider-family-azure:v1.0.0"},
		"status": map[string]any{"currentRevision": "active-revision", "conditions": []any{map[string]any{"type": "Installed", "status": "True"}, map[string]any{"type": "Healthy", "status": "True"}}},
	}}
	crd := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
		"metadata": map[string]any{"name": "providerconfigs.azure.example.com", "ownerReferences": []any{map[string]any{"apiVersion": "pkg.crossplane.io/v1", "kind": "ProviderRevision", "name": "active-revision", "uid": "revision-uid"}}},
		"spec":     map[string]any{"group": "azure.example.com", "scope": "Namespaced", "names": map[string]any{"kind": "ProviderConfig", "plural": "providerconfigs"}, "versions": []any{map[string]any{"name": "v1beta1", "served": true, "storage": true}}},
		"status":   map[string]any{"conditions": []any{map[string]any{"type": "Established", "status": "True"}}},
	}}
	config := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "azure.example.com/v1beta1", "kind": "ProviderConfig", "metadata": map[string]any{"name": "production", "namespace": "team-a"}, "spec": map[string]any{"credentials": map[string]any{"source": "Secret", "secretRef": map[string]any{"name": "never-exposed", "key": "creds"}}}}}
	gvr := schema.GroupVersionResource{Group: "azure.example.com", Version: "v1beta1", Resource: "providerconfigs"}
	c := &Client{Dynamic: fakeDynamic(map[schema.GroupVersionResource]string{packageGVRs["Provider"][0]: "ProviderList", crdGVR: "CustomResourceDefinitionList", gvr: "ProviderConfigList"}, provider, crd, config)}
	configs, err := c.ProviderConfigs(ctx)
	if err != nil || len(configs) != 1 || configs[0].Name != "production" || configs[0].Namespace != "team-a" || configs[0].Provider != "existing-family" || configs[0].CredentialSource != "Secret" {
		t.Fatalf("configs=%+v err=%v", configs, err)
	}
	// Configs belonging to old/inactive revisions must disappear, even though their CRDs remain served.
	provider.Object["status"].(map[string]any)["currentRevision"] = "different-revision"
	if _, err = c.Dynamic.Resource(packageGVRs["Provider"][0]).Update(ctx, provider, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	configs, err = c.ProviderConfigs(ctx)
	if err != nil || len(configs) != 0 {
		t.Fatalf("inactive configs=%+v err=%v", configs, err)
	}
	installed, err := c.InstallProvider(ctx, "crosswatc-helm", "xpkg.crossplane.io/crossplane-contrib/provider-helm:v1.5.0", "issuer#subject")
	if err != nil || !strings.Contains(installed.Package, "provider-helm:v1.5.0") {
		t.Fatalf("%+v %v", installed, err)
	}
	obj, err := c.Get(ctx, packageGVRs["Provider"][0], "", "crosswatc-helm")
	if err != nil || obj.GetAnnotations()["crosswatc.io/requested-by"] != "issuer#subject" || str(obj.Object, "spec", "revisionActivationPolicy") != "Automatic" {
		t.Fatalf("%+v %v", obj, err)
	}
	if _, err = c.InstallProvider(ctx, "crosswatc-helm", "xpkg.crossplane.io/crossplane-contrib/provider-helm:v9.9.9", "issuer#subject"); !apierrors.IsAlreadyExists(err) {
		t.Fatalf("must not upgrade existing: %v", err)
	}
}
