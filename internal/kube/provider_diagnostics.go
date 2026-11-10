package kube

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/agnaldom/crosswatc/internal/domain"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ponytail: runtime/log reads are restricted to the default Crossplane namespace.
// Add an explicit namespace setting for installations elsewhere; never accept a namespace from the browser.
const providerRuntimeNamespace = "crossplane-system"

var providerRevisionGVR = schema.GroupVersionResource{Group: "pkg.crossplane.io", Version: "v1", Resource: "providerrevisions"}
var providerPodsGVR = schema.GroupVersionResource{Version: "v1", Resource: "pods"}

type ProviderContainer struct {
	Name     string `json:"name"`
	Ready    bool   `json:"ready"`
	Restarts int64  `json:"restarts"`
	State    string `json:"state"`
	Reason   string `json:"reason,omitempty"`
	Message  string `json:"message,omitempty"`
}
type ProviderPod struct {
	Name       string              `json:"name"`
	Namespace  string              `json:"namespace"`
	Phase      string              `json:"phase"`
	Containers []ProviderContainer `json:"containers"`
}
type ProviderDiagnostics struct {
	Provider           Package           `json:"provider"`
	RevisionConditions domain.Conditions `json:"revisionConditions"`
	Pods               []ProviderPod     `json:"pods"`
	ControllerPods     []ProviderPod     `json:"controllerPods"`
	Events             []Event           `json:"events"`
	Warnings           []string          `json:"warnings"`
}
type ProviderLog struct {
	Pod       string `json:"pod"`
	Namespace string `json:"namespace"`
	Container string `json:"container"`
	Source    string `json:"source"`
	Previous  bool   `json:"previous"`
	Text      string `json:"text"`
	Error     string `json:"error,omitempty"`
}

// ProviderDiagnostics preserves successful reads when optional events/runtime reads fail.
func (c *Client) ProviderDiagnostics(ctx context.Context, name string) (ProviderDiagnostics, error) {
	u, err := c.Get(ctx, packageGVRs["Provider"][0], "", name)
	if err != nil {
		return ProviderDiagnostics{}, err
	}
	out := ProviderDiagnostics{Provider: parsePackage(u, "Provider"), RevisionConditions: domain.Conditions{}, Pods: []ProviderPod{}, ControllerPods: []ProviderPod{}, Events: []Event{}, Warnings: []string{}}
	addEvents := func(obj *unstructured.Unstructured) {
		if obj.GetUID() == "" {
			return
		}
		ref := Ref{APIVersion: obj.GetAPIVersion(), Kind: obj.GetKind(), Name: obj.GetName(), Namespace: obj.GetNamespace()}
		items, err := c.List(ctx, eventsGVR, ListOptions{Namespace: eventsNamespace(ref), FieldSelector: "involvedObject.uid=" + string(obj.GetUID())})
		if err != nil {
			out.Warnings = append(out.Warnings, "Events: "+err.Error())
			return
		}
		for i := range items {
			out.Events = append(out.Events, parseEvent(&items[i], ref))
		}
	}
	addEvents(u)
	if rev := out.Provider.CurrentRevision; rev != "" {
		revision, err := c.Get(ctx, providerRevisionGVR, "", rev)
		if err != nil {
			out.Warnings = append(out.Warnings, "Revision: "+err.Error())
		} else if ownedByProvider(revision, u) {
			out.RevisionConditions = Conditions(revision)
			addEvents(revision)
		} else {
			out.Warnings = append(out.Warnings, "Current revision is not owned by this Provider")
		}
	}
	pods, err := c.providerRuntimePods(ctx, u)
	if err != nil {
		out.Warnings = append(out.Warnings, "Runtime pods: "+err.Error())
	}
	for i := range pods {
		out.Pods = append(out.Pods, summarizeProviderPod(&pods[i]))
		addEvents(&pods[i])
	}
	controllers, err := c.crossplaneControllerPods(ctx)
	if err != nil {
		out.Warnings = append(out.Warnings, "Crossplane controller pods: "+err.Error())
	}
	for i := range controllers {
		out.ControllerPods = append(out.ControllerPods, summarizeProviderPod(&controllers[i]))
	}
	SortEvents(out.Events)
	if len(out.Events) > 50 {
		out.Events = out.Events[:50]
	}
	return out, nil
}
func ownedByProvider(revision, provider *unstructured.Unstructured) bool {
	for _, owner := range revision.GetOwnerReferences() {
		if owner.Kind == "Provider" && owner.Name == provider.GetName() && owner.UID == provider.GetUID() {
			return true
		}
	}
	return false
}
func (c *Client) providerRuntimePods(ctx context.Context, provider *unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	pods, err := c.List(ctx, providerPodsGVR, ListOptions{Namespace: providerRuntimeNamespace, LabelSelector: "pkg.crossplane.io/provider=" + provider.GetName()})
	if err != nil {
		return nil, err
	}
	revision := str(provider.Object, "status", "currentRevision")
	out := []unstructured.Unstructured{}
	for _, pod := range pods {
		labels := pod.GetLabels()
		if pod.GetNamespace() != providerRuntimeNamespace || labels["pkg.crossplane.io/provider"] != provider.GetName() {
			continue
		}
		if revision != "" && labels["pkg.crossplane.io/revision"] != revision {
			continue
		}
		out = append(out, pod)
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out, nil
}
func summarizeProviderPod(pod *unstructured.Unstructured) ProviderPod {
	out := ProviderPod{Name: pod.GetName(), Namespace: pod.GetNamespace(), Phase: str(pod.Object, "status", "phase"), Containers: []ProviderContainer{}}
	for _, field := range []string{"initContainerStatuses", "containerStatuses"} {
		for _, raw := range slice(pod.Object, "status", field) {
			state, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			restarts, _, _ := unstructured.NestedInt64(state, "restartCount")
			item := ProviderContainer{Name: str(state, "name"), Ready: boolean(state, "ready"), Restarts: restarts, State: "Unknown"}
			for _, kind := range []string{"waiting", "terminated", "running"} {
				if detail, ok := obj(state, "state")[kind].(map[string]any); ok {
					item.State = kind
					item.Reason = str(detail, "reason")
					item.Message = str(detail, "message")
					break
				}
			}
			out.Containers = append(out.Containers, item)
		}
	}
	return out
}

// ProviderLogs accepts no pod, namespace or container input. Existing Kubernetes RBAC
// is authoritative; no extra permissions are granted. Shared controller lines are filtered.
func (c *Client) ProviderLogs(ctx context.Context, name string) ([]ProviderLog, error) {
	provider, err := c.Get(ctx, packageGVRs["Provider"][0], "", name)
	if err != nil {
		return nil, err
	}
	if c.Core == nil {
		return nil, fmt.Errorf("controller log client is unavailable")
	}
	out := []ProviderLog{}
	runtimePods, err := c.providerRuntimePods(ctx, provider)
	if err != nil {
		out = append(out, ProviderLog{Source: "provider", Error: err.Error()})
	}
	patterns := []*regexp.Regexp{}
	for _, term := range []string{name, str(provider.Object, "status", "currentRevision"), providerImageName(str(provider.Object, "spec", "package"))} {
		if term != "" {
			patterns = append(patterns, regexp.MustCompile(`(^|[^A-Za-z0-9_.-])`+regexp.QuoteMeta(term)+`($|[^A-Za-z0-9_.-])`))
		}
	}
	read := func(pod unstructured.Unstructured, source string, filter bool) {
		if pod.GetNamespace() != providerRuntimeNamespace {
			return
		}
		containers := slice(pod.Object, "spec", "containers")
		if len(containers) > 2 {
			containers = containers[:2]
		}
		for _, raw := range containers {
			container, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			lines, limit := int64(200), int64(64<<10)
			options := &corev1.PodLogOptions{Container: str(container, "name"), TailLines: &lines, LimitBytes: &limit, Timestamps: true}
			restarted := false
			for _, status := range summarizeProviderPod(&pod).Containers {
				if status.Name == options.Container && status.Restarts > 0 {
					restarted = true
				}
			}
			attempts := []bool{false}
			if source == "crossplane" && restarted {
				attempts = append(attempts, true)
			}
			for _, previous := range attempts {
				options.Previous = previous
				data, err := c.Core.Pods(providerRuntimeNamespace).GetLogs(pod.GetName(), options).DoRaw(ctx)
				entry := ProviderLog{Pod: pod.GetName(), Namespace: providerRuntimeNamespace, Container: options.Container, Source: source, Previous: previous}
				if err != nil && !previous && source != "crossplane" && restarted {
					options.Previous = true
					entry.Previous = true
					data, err = c.Core.Pods(providerRuntimeNamespace).GetLogs(pod.GetName(), options).DoRaw(ctx)
				}
				if err != nil {
					entry.Error = err.Error()
				} else if filter {
					selected := []string{}
					for _, line := range strings.Split(string(data), "\n") {
						// Global controller failures explain why all package installations stop.
						match := strings.Contains(line, "leader election lost") || strings.Contains(line, "failed to renew lease") || strings.Contains(line, "cannot start controller manager") || strings.Contains(line, "panic:") || strings.Contains(line, "OOMKilled")
						for _, pattern := range patterns {
							if pattern.MatchString(line) {
								match = true
								break
							}
						}
						if match {
							selected = append(selected, line)
						}
					}
					entry.Text = strings.Join(selected, "\n")
				} else {
					entry.Text = string(data)
				}
				out = append(out, entry)
			}
		}
	}
	for _, pod := range runtimePods {
		read(pod, "provider", false)
	}
	controllers, err := c.crossplaneControllerPods(ctx)
	if err != nil {
		out = append(out, ProviderLog{Source: "crossplane", Error: err.Error()})
	}
	for _, pod := range controllers {
		read(pod, "crossplane", true)
	}
	return out, nil
}
func providerImageName(ref string) string {
	ref = strings.SplitN(ref, "@", 2)[0]
	return strings.SplitN(ref[strings.LastIndex(ref, "/")+1:], ":", 2)[0]
}

func (c *Client) crossplaneControllerPods(ctx context.Context) ([]unstructured.Unstructured, error) {
	pods, err := c.List(ctx, providerPodsGVR, ListOptions{Namespace: providerRuntimeNamespace, LabelSelector: "app=crossplane"})
	if err != nil {
		return nil, err
	}
	out := []unstructured.Unstructured{}
	for _, pod := range pods {
		if pod.GetNamespace() == providerRuntimeNamespace && pod.GetLabels()["app"] == "crossplane" {
			out = append(out, pod)
		}
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out, nil
}
