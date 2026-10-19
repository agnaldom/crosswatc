package kube

import (
	"context"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var eventsGVR = schema.GroupVersionResource{Version: "v1", Resource: "events"}

// Event is a Kubernetes Event about an object (Crossplane controllers explain failures here).
type Event struct {
	Object    Ref       `json:"object"`
	Type      string    `json:"type"` // Normal | Warning
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int64     `json:"count"`
	Source    string    `json:"source,omitempty"`
	FirstSeen time.Time `json:"firstSeen,omitzero"`
	LastSeen  time.Time `json:"lastSeen,omitzero"`
}

// eventsNamespace: events of cluster-scoped objects (XRs and most MRs) are recorded in "default".
func eventsNamespace(ref Ref) string {
	if ref.Namespace == "" {
		return "default"
	}
	return ref.Namespace
}

// EventsFor lists the Events whose involvedObject is ref (field selector, paginated).
func (c *Client) EventsFor(ctx context.Context, ref Ref) ([]Event, error) {
	sel := []string{"involvedObject.kind=" + ref.Kind, "involvedObject.name=" + ref.Name}
	if ref.Namespace != "" {
		sel = append(sel, "involvedObject.namespace="+ref.Namespace)
	}
	items, err := c.List(ctx, eventsGVR, ListOptions{Namespace: eventsNamespace(ref), FieldSelector: strings.Join(sel, ",")})
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(items))
	for i := range items {
		out = append(out, parseEvent(&items[i], ref))
	}
	SortEvents(out)
	return out, nil
}

// SortEvents orders by last occurrence, newest first.
func SortEvents(es []Event) {
	sort.SliceStable(es, func(i, j int) bool { return es[i].LastSeen.After(es[j].LastSeen) })
}

func parseEvent(u *unstructured.Unstructured, ref Ref) Event {
	o := u.Object
	e := Event{Object: ref, Type: str(o, "type"), Reason: str(o, "reason"), Message: str(o, "message")}
	e.Count, _, _ = unstructured.NestedInt64(o, "count")
	if n, ok, _ := unstructured.NestedInt64(o, "series", "count"); ok && n > e.Count {
		e.Count = n
	}
	if e.Count == 0 {
		e.Count = 1
	}
	e.Source = str(o, "source", "component")
	if e.Source == "" {
		e.Source = str(o, "reportingComponent")
	}
	e.FirstSeen = ts(str(o, "firstTimestamp"), str(o, "eventTime"), u.GetCreationTimestamp().Format(time.RFC3339))
	e.LastSeen = ts(str(o, "lastTimestamp"), str(o, "series", "lastObservedTime"), str(o, "eventTime"), e.FirstSeen.Format(time.RFC3339Nano))
	return e
}

// ts returns the first parseable RFC3339 timestamp.
func ts(vals ...string) time.Time {
	for _, v := range vals {
		if v == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
	}
	return time.Time{}
}
