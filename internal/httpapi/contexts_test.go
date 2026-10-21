package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
)

type fakeContextManager struct {
	contexts []string
	current  string
}

func (f *fakeContextManager) GetContexts() ([]string, string, bool, error) {
	return f.contexts, f.current, false, nil
}

func (f *fakeContextManager) SetContext(name string) error {
	f.current = name
	return nil
}

func (f *fakeContextManager) CheckConnection(name string) (bool, error) {
	return true, nil
}

func (f *fakeContextManager) AddKubeConfig(yaml string) ([]string, error) {
	f.contexts = append(f.contexts, "new-cluster")
	return []string{"new-cluster"}, nil
}

func (f *fakeContextManager) RemoveContext(name string) error {
	var remaining []string
	for _, c := range f.contexts {
		if c != name {
			remaining = append(remaining, c)
		}
	}
	f.contexts = remaining
	return nil
}

func TestContextManagementEndpoints(t *testing.T) {
	fakeCM := &fakeContextManager{
		contexts: []string{"ctx-a", "ctx-b"},
		current:  "ctx-a",
	}
	deps := Deps{
		Store:    &fakeStore{},
		Contexts: fakeCM,
		Auth:     auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "platform-sre"}},
	}
	engine := New(deps)

	// 1. GET /api/v1/contexts
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contexts", nil)
	req.Header.Set("X-Dev-User", "admin")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var res map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["currentContext"] != "ctx-a" {
		t.Errorf("expected currentContext ctx-a, got %v", res["currentContext"])
	}

	// 2. POST /api/v1/contexts/current
	body := map[string]string{"context": "ctx-b"}
	b, _ := json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/contexts/current", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if fakeCM.current != "ctx-b" {
		t.Errorf("expected current to be ctx-b, got %s", fakeCM.current)
	}

	// 3. GET /api/v1/contexts/connection
	req = httptest.NewRequest(http.MethodGet, "/api/v1/contexts/connection?context=ctx-b", nil)
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var connRes map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &connRes)
	if connRes["connected"] != true {
		t.Errorf("expected connected=true, got %v", connRes["connected"])
	}

	// 4. POST /api/v1/contexts/add
	addBody := map[string]string{"kubeConfig": "fake-yaml"}
	b, _ = json.Marshal(addBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/contexts/add", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. DELETE /api/v1/contexts
	delBody := map[string]string{"context": "new-cluster"}
	b, _ = json.Marshal(delBody)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/contexts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
