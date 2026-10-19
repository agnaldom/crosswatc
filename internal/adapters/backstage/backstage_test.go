package backstage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func server(t *testing.T, status *string, hits *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/scaffolder/v2/tasks/t1":
			_, _ = w.Write([]byte(`{"id":"t1","status":"` + *status + `"}`))
		case "/api/scaffolder/v2/tasks/boom":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTaskCaching(t *testing.T) {
	status, hits := "processing", int32(0)
	srv := server(t, &status, &hits)
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	a := New(srv.URL, "tok", 30*time.Second, srv.Client())
	a.now = func() time.Time { return now }
	ctx := context.Background()

	s, err := a.Task(ctx, "t1")
	if err != nil || s.Status != "processing" || s.TaskID != "t1" {
		t.Fatalf("%+v %v", s, err)
	}
	status = "completed"
	if s, _ = a.Task(ctx, "t1"); s.Status != "processing" || hits != 1 {
		t.Fatalf("within interval must use cache: %+v hits=%d", s, hits)
	}
	now = now.Add(31 * time.Second)
	if s, _ = a.Task(ctx, "t1"); s.Status != "completed" || hits != 2 {
		t.Fatalf("after interval must refetch: %+v hits=%d", s, hits)
	}
	now = now.Add(time.Hour)
	if s, _ = a.Task(ctx, "t1"); s.Status != "completed" || hits != 2 {
		t.Fatalf("terminal status must never refetch: hits=%d", hits)
	}
}

func TestTaskErrorsAndDisabled(t *testing.T) {
	status, hits := "open", int32(0)
	srv := server(t, &status, &hits)
	a := New(srv.URL, "tok", time.Second, srv.Client())
	ctx := context.Background()
	if _, err := a.Task(ctx, "nope"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
	if _, err := a.Task(ctx, "boom"); err == nil {
		t.Fatal("expected HTTP error")
	}
	if s, err := a.Task(ctx, ""); err != nil || s.TaskID != "" {
		t.Fatal("empty task id must be a no-op")
	}
	disabled := New("", "", time.Second, nil)
	if disabled.Enabled() {
		t.Fatal("must be disabled without URL")
	}
	if s, err := disabled.Task(ctx, "t1"); err != nil || s.TaskID != "" {
		t.Fatal("disabled adapter must be a no-op")
	}
	if _, err := New(srv.URL, "wrong", time.Second, srv.Client()).Task(ctx, "t1"); err == nil {
		t.Fatal("expected 401 error")
	}
}
