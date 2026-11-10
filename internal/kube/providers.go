package kube

import (
	"context"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ProviderConfigSummary exposes configuration metadata, never credential values or Secrets.
type ProviderConfigSummary struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace,omitempty"`
	Kind             string `json:"kind"`
	APIVersion       string `json:"apiVersion"`
	Provider         string `json:"provider"`
	CredentialSource string `json:"credentialSource,omitempty"`
}

// ProviderConfigs discovers config APIs owned by the current revision of healthy, installed providers.
func (c *Client) ProviderConfigs(ctx context.Context) ([]ProviderConfigSummary, error) {
	providers, err := c.Packages(ctx, "Provider")
	if err != nil {
		return nil, err
	}
	active := map[string]string{}
	for _, p := range providers {
		if p.CurrentRevision != "" && p.Conditions.IsTrue("Installed") && p.Conditions.IsTrue("Healthy") {
			active[p.CurrentRevision] = p.Name
		}
	}
	out := []ProviderConfigSummary{}
	if len(active) == 0 {
		return out, nil
	}
	var crds []unstructured.Unstructured
	if c.Metadata != nil {
		metaList, err := c.Metadata.Resource(crdGVR).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, item := range metaList.Items {
				if !strings.HasPrefix(item.Name, "providerconfigs.") && !strings.HasPrefix(item.Name, "clusterproviderconfigs.") {
					continue
				}
				hasActiveOwner := false
				for _, owner := range item.GetOwnerReferences() {
					if owner.APIVersion == "pkg.crossplane.io/v1" && owner.Kind == "ProviderRevision" && active[owner.Name] != "" {
						hasActiveOwner = true
						break
					}
				}
				if !hasActiveOwner {
					continue
				}
				crd, err := c.Get(ctx, crdGVR, "", item.Name)
				if err != nil {
					continue
				}
				crds = append(crds, *crd)
			}
		} else {
			crds, err = c.List(ctx, crdGVR, ListOptions{})
			if err != nil {
				return nil, err
			}
		}
	} else {
		crds, err = c.List(ctx, crdGVR, ListOptions{})
		if err != nil {
			return nil, err
		}
	}
	for i := range crds {
		crd := &crds[i]
		kind := str(crd.Object, "spec", "names", "kind")
		if kind != "ProviderConfig" && kind != "ClusterProviderConfig" {
			continue
		}
		provider := ""
		for _, owner := range crd.GetOwnerReferences() {
			if owner.APIVersion == "pkg.crossplane.io/v1" && owner.Kind == "ProviderRevision" {
				provider = active[owner.Name]
				if provider != "" {
					break
				}
			}
		}
		if provider == "" || !Conditions(crd).IsTrue("Established") {
			continue
		}
		target, ok := parseCRDLike(crd, false)
		if !ok {
			continue
		}
		// Only read a served version; storage versions may no longer be served.
		target.Version = ""
		for _, raw := range slice(crd.Object, "spec", "versions") {
			if v, ok := raw.(map[string]any); ok && boolean(v, "served") {
				target.Version = str(v, "name")
				if boolean(v, "storage") {
					break
				}
			}
		}
		if target.Version == "" {
			continue
		}
		configs, err := c.List(ctx, target.GVR(), ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, config := range configs {
			if config.GetDeletionTimestamp() != nil {
				continue
			}
			out = append(out, ProviderConfigSummary{Name: config.GetName(), Namespace: config.GetNamespace(), Kind: kind, APIVersion: target.APIVersion(), Provider: provider, CredentialSource: str(config.Object, "spec", "credentials", "source")})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		return a.Provider+"/"+a.APIVersion+"/"+a.Namespace+"/"+a.Name < b.Provider+"/"+b.APIVersion+"/"+b.Namespace+"/"+b.Name
	})
	return out, nil
}

// InstallProvider creates a package only. It never changes existing packages, configs or cloud resources.
func (c *Client) InstallProvider(ctx context.Context, name, ref, actor string) (Package, error) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "pkg.crossplane.io/v1", "kind": "Provider",
		"metadata": map[string]any{"name": name, "labels": map[string]any{"app.kubernetes.io/managed-by": "crosswatc"}, "annotations": map[string]any{"crosswatc.io/requested-by": actor}},
		"spec":     map[string]any{"package": ref, "packagePullPolicy": "IfNotPresent", "revisionActivationPolicy": "Automatic"},
	}}
	created, err := c.Dynamic.Resource(packageGVRs["Provider"][0]).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return Package{}, err
	}
	return parsePackage(created, "Provider"), nil
}
