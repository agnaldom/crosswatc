package redact

import (
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	in := map[string]any{
		"parameters": map[string]any{
			"vmSize":        "D2",
			"adminPassword": "hunter2",
			"apiKey":        "k",
			"nested":        []any{map[string]any{"clientSecret": "s", "name": "x"}},
			"credentials":   map[string]any{"source": "Secret", "secretRef": map[string]any{"name": "creds"}},
		},
		"writeConnectionSecretToRef": map[string]any{"name": "pg-conn"},
		"passwordSecretName":         "admin-pw",
	}
	want := map[string]any{
		"parameters": map[string]any{
			"vmSize":        "D2",
			"adminPassword": Placeholder,
			"apiKey":        Placeholder,
			"nested":        []any{map[string]any{"clientSecret": Placeholder, "name": "x"}},
			"credentials":   map[string]any{"source": "Secret", "secretRef": map[string]any{"name": "creds"}},
		},
		"writeConnectionSecretToRef": map[string]any{"name": "pg-conn"},
		"passwordSecretName":         "admin-pw",
	}
	got := Map(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
	if in["parameters"].(map[string]any)["adminPassword"] != "hunter2" {
		t.Fatal("input must not be mutated")
	}
	if Map(nil) != nil {
		t.Fatal("nil")
	}
}
