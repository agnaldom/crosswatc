package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestNewJSONAndLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "json", "warn")
	l.Info("hidden")
	l.Warn("shown", "correlation.id", "x")
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("expected single json line, got %q: %v", buf.String(), err)
	}
	if m["msg"] != "shown" || m["correlation.id"] != "x" {
		t.Fatalf("unexpected record: %v", m)
	}
}

func TestContextRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "text", "bogus")
	ctx := WithLogger(context.Background(), l)
	if FromContext(ctx) != l {
		t.Fatal("logger not returned from context")
	}
	if FromContext(context.Background()) == nil {
		t.Fatal("expected default logger")
	}
}
