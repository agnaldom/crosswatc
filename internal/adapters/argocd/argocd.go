// Package argocd reads Argo CD Application status through the Kubernetes API (spec §35.1).
// No Argo CD HTTP API is used: Applications are CRs in the same control cluster.
package argocd

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/agnaldom/crosswatc/internal/contract"
	"github.com/agnaldom/crosswatc/internal/lifecycle"
)

// ApplicationGVR is the Argo CD Application resource.
var ApplicationGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}

const (
	trackingIDAnnotation = "argocd.argoproj.io/tracking-id"
	instanceLabel        = "app.kubernetes.io/instance"
)

// Getter fetches a single object (satisfied by *kube.Client).
type Getter interface {
	Get(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error)
}

// Adapter resolves and reads Applications.
type Adapter struct {
	kube             Getter
	defaultNamespace string
	isNotFound       func(error) bool
}

// New builds the adapter. defaultNamespace is where Applications live (usually "argocd").
func New(g Getter, defaultNamespace string, isNotFound func(error) bool) *Adapter {
	return &Adapter{kube: g, defaultNamespace: defaultNamespace, isNotFound: isNotFound}
}

// AppRef locates an Application.
type AppRef struct {
	Namespace string
	Name      string
}

func (r AppRef) String() string { return r.Namespace + "/" + r.Name }

// ResolveApp finds the Application managing a Claim, in priority order:
// crosswatc.io/argo-app annotation ("name" or "namespace/name"), Argo tracking-id annotation, instance label.
func (a *Adapter) ResolveApp(labels, annotations map[string]string) (AppRef, bool) {
	if v := annotations[contract.AnnotationArgoApp]; v != "" {
		if ns, name, ok := strings.Cut(v, "/"); ok {
			return AppRef{ns, name}, true
		}
		return AppRef{a.defaultNamespace, v}, true
	}
	if v := annotations[trackingIDAnnotation]; v != "" {
		app, _, _ := strings.Cut(v, ":")
		return a.parseAppName(app), app != ""
	}
	if v := labels[instanceLabel]; v != "" {
		return a.parseAppName(v), true
	}
	return AppRef{}, false
}

// parseAppName handles apps-in-any-namespace names ("ns_name").
func (a *Adapter) parseAppName(s string) AppRef {
	if ns, name, ok := strings.Cut(s, "_"); ok && ns != "" && name != "" {
		return AppRef{ns, name}
	}
	return AppRef{a.defaultNamespace, s}
}

// Status reads the Application and maps it to the lifecycle signal. A missing app yields Found=false.
func (a *Adapter) Status(ctx context.Context, ref AppRef) (lifecycle.ArgoSignal, error) {
	u, err := a.kube.Get(ctx, ApplicationGVR, ref.Namespace, ref.Name)
	if err != nil {
		if a.isNotFound != nil && a.isNotFound(err) {
			return lifecycle.ArgoSignal{}, nil
		}
		return lifecycle.ArgoSignal{}, err
	}
	return Parse(u), nil
}

// Parse extracts sync, health and operation state.
func Parse(u *unstructured.Unstructured) lifecycle.ArgoSignal {
	s := lifecycle.ArgoSignal{
		Found:          true,
		SyncStatus:     nested(u, "status", "sync", "status"),
		Health:         nested(u, "status", "health", "status"),
		OperationPhase: nested(u, "status", "operationState", "phase"),
		SyncedRevision: nested(u, "status", "sync", "revision"),
		Message:        nested(u, "status", "operationState", "message"),
	}
	if s.SyncedRevision == "" {
		if revs, _, _ := unstructured.NestedStringSlice(u.Object, "status", "sync", "revisions"); len(revs) > 0 {
			s.SyncedRevision = revs[0] // multi-source: first source is the claims repo by convention
		}
	}
	return s
}

func nested(u *unstructured.Unstructured, path ...string) string {
	v, _, _ := unstructured.NestedString(u.Object, path...)
	return v
}
