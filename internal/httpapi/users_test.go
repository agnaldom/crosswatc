package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/store"
)

func TestUserManagementEndpoints(t *testing.T) {
	memStore := newInMemoryUserStore()
	deps := Deps{
		Store: &fakeStore{},
		Users: memStore,
		Auth:  auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-", SREGroup: "platform-sre"}},
	}
	engine := New(deps)

	// 1. GET /api/v1/users as admin
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("X-Dev-User", "admin")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var users []store.User
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(users) != 1 || users[0].Username != "admin" {
		t.Fatalf("unexpected users: %+v", users)
	}

	// 2. Non-admin forbidden
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("X-Dev-User", "regular-dev")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin, got %d", rec.Code)
	}

	// 3. POST /api/v1/users to create a user
	createBody := map[string]string{
		"username": "developer1",
		"email":    "dev1@company.com",
		"password": "Password123!",
		"role":     "user",
	}
	b, _ := json.Marshal(createBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	var created store.User
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Username != "developer1" || created.Role != "user" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	// 4. PUT /api/v1/users/:id to update user
	updateBody := map[string]string{
		"username": "developer1-updated",
		"email":    "dev1-new@company.com",
		"role":     "admin",
	}
	b, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/2", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated store.User
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.Username != "developer1-updated" || updated.Role != "admin" {
		t.Fatalf("unexpected updated user: %+v", updated)
	}

	// 5. DELETE /api/v1/users/:id
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/2", nil)
	req.Header.Set("X-Dev-User", "admin")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
}
