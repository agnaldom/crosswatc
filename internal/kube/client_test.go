package kube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	mdfake "k8s.io/client-go/metadata/fake"
	"k8s.io/client-go/rest"
)

var widgetGVR = schema.GroupVersionResource{Group: "example.io", Version: "v1", Resource: "widgets"}

func widget(i int) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("example.io/v1")
	u.SetKind("Widget")
	u.SetName(fmt.Sprintf("w-%d", i))
	return u
}

// pagedServer serves total widgets honoring limit/continue and records each request query.
func pagedServer(t *testing.T, total int, seen *[]url.Values) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis/example.io/v1/widgets" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		*seen = append(*seen, q)
		limit, _ := strconv.Atoi(q.Get("limit"))
		start, _ := strconv.Atoi(q.Get("continue"))
		end := min(start+limit, total)
		list := &unstructured.UnstructuredList{}
		list.SetAPIVersion("example.io/v1")
		list.SetKind("WidgetList")
		for i := start; i < end; i++ {
			list.Items = append(list.Items, widget(i))
		}
		if end < total {
			list.SetContinue(strconv.Itoa(end))
		}
		w.Header().Set("Content-Type", "application/json")
		b, _ := list.MarshalJSON()
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	dyn, err := dynamic.NewForConfig(&rest.Config{Host: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{Dynamic: dyn}
}

func TestListPaginates(t *testing.T) {
	var seen []url.Values
	c := pagedServer(t, 7, &seen)
	c.PageLimit = 3
	items, err := c.List(context.Background(), widgetGVR, ListOptions{LabelSelector: "crosswatc.io/managed=true"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 7 || items[6].GetName() != "w-6" {
		t.Fatalf("got %d items", len(items))
	}
	if len(seen) != 3 {
		t.Fatalf("want 3 page requests, got %d", len(seen))
	}
	for _, q := range seen {
		if q.Get("limit") != "3" || q.Get("labelSelector") != "crosswatc.io/managed=true" {
			t.Fatalf("bad query %v", q)
		}
	}
}

func TestDefaultPageLimit(t *testing.T) {
	var seen []url.Values
	c := pagedServer(t, 1, &seen)
	if _, err := c.List(context.Background(), widgetGVR, ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if seen[0].Get("limit") != strconv.FormatInt(DefaultPageLimit, 10) {
		t.Fatalf("limit = %s", seen[0].Get("limit"))
	}
}

func TestForEachPageStopsOnError(t *testing.T) {
	var seen []url.Values
	c := pagedServer(t, 10, &seen)
	c.PageLimit = 2
	stop := errors.New("stop")
	err := c.ForEachPage(context.Background(), widgetGVR, ListOptions{}, func([]unstructured.Unstructured) error { return stop })
	if !errors.Is(err, stop) || len(seen) != 1 {
		t.Fatalf("err=%v pages=%d", err, len(seen))
	}
}

func TestForEachPageRejectsRepeatedContinue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apiVersion":"example.io/v1","kind":"WidgetList","metadata":{"continue":"same"},"items":[]}`))
	}))
	defer srv.Close()
	dyn, _ := dynamic.NewForConfig(&rest.Config{Host: srv.URL})
	c := &Client{Dynamic: dyn}
	if _, err := c.List(context.Background(), widgetGVR, ListOptions{}); err == nil {
		t.Fatal("expected error on repeated continue token")
	}
}

func TestSecretExistsUsesMetadataOnly(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = metav1.AddMetaToScheme(scheme)
	sec := &metav1.PartialObjectMetadata{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "pg-conn"},
	}
	c := &Client{Metadata: mdfake.NewSimpleMetadataClient(scheme, sec)}
	ok, err := c.SecretExists(context.Background(), "team-a", "pg-conn")
	if err != nil || !ok {
		t.Fatalf("exists: %v %v", ok, err)
	}
	ok, err = c.SecretExists(context.Background(), "team-a", "missing")
	if err != nil || ok {
		t.Fatalf("missing: %v %v", ok, err)
	}
}

func TestRESTConfig(t *testing.T) {
	if _, err := RESTConfig(""); err == nil {
		t.Fatal("expected in-cluster error outside a pod")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	kc := `apiVersion: v1
kind: Config
clusters: [{name: c, cluster: {server: "https://127.0.0.1:6443"}}]
users: [{name: u, user: {token: t}}]
contexts: [{name: x, context: {cluster: c, user: u}}]
current-context: x
`
	if err := os.WriteFile(path, []byte(kc), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := RESTConfig(path)
	if err != nil || cfg.Host != "https://127.0.0.1:6443" {
		t.Fatalf("cfg=%v err=%v", cfg, err)
	}
	if _, err := New(cfg, 10); err != nil {
		t.Fatal(err)
	}
}

func TestResourceTarget(t *testing.T) {
	rt := ResourceTarget{Group: "example.io", Version: "v1", Kind: "Widget", Plural: "widgets"}
	if rt.GVR() != widgetGVR || rt.APIVersion() != "example.io/v1" {
		t.Fatalf("%v %s", rt.GVR(), rt.APIVersion())
	}
	if (ResourceTarget{Version: "v1"}).APIVersion() != "v1" {
		t.Fatal("core group apiVersion")
	}
}
