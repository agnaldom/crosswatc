// Package kube is the single-cluster Kubernetes access layer (spec §3.4, §21, §24, §26).
// There is no context switching, cluster registry or multi-kubeconfig support by design.
package kube

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DefaultPageLimit is the list page size (§24).
const DefaultPageLimit int64 = 500

// RESTConfig returns in-cluster config, or the current context of kubeconfigPath when set.
func RESTConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("kube: in-cluster config (set KUBECONFIG for local development): %w", err)
		}
		return cfg, nil
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("kube: kubeconfig %s: %w", kubeconfigPath, err)
	}
	return cfg, nil
}

// Client bundles the dynamic and metadata clients for the control cluster.
type Client struct {
	Dynamic   dynamic.Interface
	Metadata  metadata.Interface // metadata-only reads (Secrets: never fetch values, §49)
	PageLimit int64
	Mapper    meta.RESTMapper // Kind → resource, for objects known only by apiVersion/kind (optional)
}

// New builds a Client from a REST config.
func New(cfg *rest.Config, pageLimit int64) (*Client, error) {
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	md, err := metadata.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	return &Client{Dynamic: dyn, Metadata: md, PageLimit: pageLimit, Mapper: mapper}, nil
}

// ResourceTarget identifies a resource type (§21).
type ResourceTarget struct {
	Group      string
	Version    string
	Kind       string
	Plural     string
	Namespaced bool
}

// GVR returns the GroupVersionResource of the target.
func (t ResourceTarget) GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: t.Group, Version: t.Version, Resource: t.Plural}
}

// APIVersion returns group/version (or version for the core group).
func (t ResourceTarget) APIVersion() string {
	return schema.GroupVersion{Group: t.Group, Version: t.Version}.String()
}

// ListOptions narrows List.
type ListOptions struct {
	Namespace     string // empty: all namespaces
	LabelSelector string
	FieldSelector string
}

// List returns every object of gvr, paginating with limit/continue (§24).
func (c *Client) List(ctx context.Context, gvr schema.GroupVersionResource, opts ListOptions) ([]unstructured.Unstructured, error) {
	var out []unstructured.Unstructured
	err := c.ForEachPage(ctx, gvr, opts, func(items []unstructured.Unstructured) error {
		out = append(out, items...)
		return nil
	})
	return out, err
}

// ForEachPage calls fn for each page, never loading the whole collection in one request.
func (c *Client) ForEachPage(ctx context.Context, gvr schema.GroupVersionResource, opts ListOptions, fn func([]unstructured.Unstructured) error) error {
	limit := c.PageLimit
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	ri := c.Dynamic.Resource(gvr)
	var lister dynamic.ResourceInterface = ri
	if opts.Namespace != "" {
		lister = ri.Namespace(opts.Namespace)
	}
	cont := ""
	for {
		list, err := lister.List(ctx, metav1.ListOptions{Limit: limit, Continue: cont, LabelSelector: opts.LabelSelector, FieldSelector: opts.FieldSelector})
		if err != nil {
			return fmt.Errorf("kube: list %s: %w", gvr, err)
		}
		if err := fn(list.Items); err != nil {
			return err
		}
		next := list.GetContinue()
		if next == "" {
			return nil
		}
		if next == cont {
			return fmt.Errorf("kube: list %s: server repeated continue token", gvr)
		}
		cont = next
	}
}

// Get fetches a single object. namespace may be empty for cluster-scoped kinds.
func (c *Client) Get(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	ri := c.Dynamic.Resource(gvr)
	if namespace != "" {
		return ri.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	return ri.Get(ctx, name, metav1.GetOptions{})
}

var secretsGVR = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}

// SecretExists checks a Secret by metadata only; values are never read (§49).
func (c *Client) SecretExists(ctx context.Context, namespace, name string) (bool, error) {
	_, err := c.Metadata.Resource(secretsGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if IsNotFound(err) {
		return false, nil
	}
	return false, err
}
