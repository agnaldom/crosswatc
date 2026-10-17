// Package backstage polls Backstage Scaffolder tasks (spec §31, stage 1).
package backstage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/agnaldom/crosswatc/internal/lifecycle"
)

// ErrTaskNotFound means the task id does not exist in Backstage.
var ErrTaskNotFound = errors.New("backstage: task not found")

// Terminal Scaffolder statuses.
var terminal = map[string]bool{"completed": true, "failed": true, "cancelled": true}

// Adapter fetches task status with a per-task cache: terminal statuses are cached forever,
// non-terminal ones for PollInterval.
type Adapter struct {
	baseURL      string
	token        string
	pollInterval time.Duration
	http         *http.Client
	now          func() time.Time

	mu    sync.Mutex
	cache map[string]entry
}

type entry struct {
	status    string
	fetchedAt time.Time
}

// New builds an adapter. baseURL is the Backstage backend root (no trailing slash).
func New(baseURL, token string, pollInterval time.Duration, hc *http.Client) *Adapter {
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{baseURL: baseURL, token: token, pollInterval: pollInterval, http: hc, now: time.Now, cache: map[string]entry{}}
}

// Enabled reports whether a Backstage URL is configured.
func (a *Adapter) Enabled() bool { return a != nil && a.baseURL != "" }

// Task returns the lifecycle signal for taskID. An empty taskID yields an empty signal (stage SKIPPED).
func (a *Adapter) Task(ctx context.Context, taskID string) (lifecycle.BackstageSignal, error) {
	if taskID == "" || !a.Enabled() {
		return lifecycle.BackstageSignal{}, nil
	}
	a.mu.Lock()
	e, ok := a.cache[taskID]
	a.mu.Unlock()
	if ok && (terminal[e.status] || a.now().Sub(e.fetchedAt) < a.pollInterval) {
		return lifecycle.BackstageSignal{TaskID: taskID, Status: e.status}, nil
	}

	status, err := a.fetch(ctx, taskID)
	if err != nil {
		return lifecycle.BackstageSignal{}, err
	}
	a.mu.Lock()
	a.cache[taskID] = entry{status: status, fetchedAt: a.now()}
	a.mu.Unlock()
	return lifecycle.BackstageSignal{TaskID: taskID, Status: status}, nil
}

func (a *Adapter) fetch(ctx context.Context, taskID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/scaffolder/v2/tasks/"+url.PathEscape(taskID), nil)
	if err != nil {
		return "", err
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("backstage: get task %s: %w", taskID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return "", ErrTaskNotFound
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("backstage: get task %s: HTTP %d", taskID, resp.StatusCode)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("backstage: decode task %s: %w", taskID, err)
	}
	return body.Status, nil
}
