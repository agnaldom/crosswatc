// Package github is the read-only GitHub adapter (spec §35, §44, §49). Authentication uses a GitHub App.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	gh "github.com/google/go-github/v93/github"

	"github.com/agnaldom/crosswatc/internal/lifecycle"
)

// Config holds GitHub App credentials.
type Config struct {
	APIURL         string // https://api.github.com or GHES https://host/api/v3
	AppID          int64
	InstallationID int64
	PrivateKeyPath string
}

// Adapter reads pull requests, checks and reviews.
type Adapter struct {
	client *gh.Client
}

// New authenticates as a GitHub App installation.
func New(cfg Config) (*Adapter, error) {
	if cfg.AppID == 0 || cfg.InstallationID == 0 || cfg.PrivateKeyPath == "" {
		return nil, errors.New("github: app id, installation id and private key path are required")
	}
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("github: app transport: %w", err)
	}
	if cfg.APIURL != "" {
		tr.BaseURL = strings.TrimRight(cfg.APIURL, "/")
	}
	return NewWithHTTPClient(&http.Client{Transport: tr}, cfg.APIURL)
}

// NewWithHTTPClient builds an adapter on an existing HTTP client (tests, custom auth).
func NewWithHTTPClient(hc *http.Client, apiURL string) (*Adapter, error) {
	opts := []gh.ClientOptionsFunc{gh.WithHTTPClient(hc)}
	if apiURL != "" && apiURL != "https://api.github.com" {
		base := strings.TrimRight(apiURL, "/") + "/"
		opts = append(opts, gh.WithURLs(&base, nil))
	}
	c, err := gh.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("github: client: %w", err)
	}
	return &Adapter{client: c}, nil
}

// Lookup identifies the pull request of a resource.
type Lookup struct {
	Repository    string // owner/name
	Path          string // claim manifest path
	CorrelationID string
	PRNumber      int // known PR number short-circuits the search
}

// PullRequest finds the PR for a resource and returns its lifecycle signal.
// Resolution order: known number → correlation ID search → last commit touching Path.
func (a *Adapter) PullRequest(ctx context.Context, l Lookup) (lifecycle.PRSignal, error) {
	owner, repo, err := splitRepo(l.Repository)
	if err != nil {
		return lifecycle.PRSignal{}, err
	}
	number := l.PRNumber
	if number == 0 && l.CorrelationID != "" {
		if number, err = a.findByCorrelation(ctx, owner, repo, l.CorrelationID); err != nil {
			return lifecycle.PRSignal{}, err
		}
	}
	if number == 0 && l.Path != "" {
		if number, err = a.findByPath(ctx, owner, repo, l.Path); err != nil {
			return lifecycle.PRSignal{}, err
		}
	}
	if number == 0 {
		return lifecycle.PRSignal{}, nil
	}
	return a.signal(ctx, owner, repo, number)
}

func (a *Adapter) findByCorrelation(ctx context.Context, owner, repo, correlationID string) (int, error) {
	q := fmt.Sprintf("repo:%s/%s is:pr %q", owner, repo, correlationID)
	res, _, err := a.client.Search.Issues(ctx, q, &gh.SearchOptions{Sort: "created", Order: "desc", ListOptions: gh.ListOptions{PerPage: 1}})
	if err != nil {
		return 0, fmt.Errorf("github: search PR: %w", err)
	}
	if len(res.Issues) == 0 {
		return 0, nil
	}
	return res.Issues[0].GetNumber(), nil
}

func (a *Adapter) findByPath(ctx context.Context, owner, repo, path string) (int, error) {
	commits, _, err := a.client.Repositories.ListCommits(ctx, owner, repo, &gh.CommitsListOptions{Path: path, ListOptions: gh.ListOptions{PerPage: 1}})
	if err != nil {
		return 0, fmt.Errorf("github: list commits: %w", err)
	}
	if len(commits) == 0 {
		return 0, nil
	}
	prs, _, err := a.client.PullRequests.ListPullRequestsWithCommit(ctx, owner, repo, commits[0].GetSHA(), &gh.ListOptions{PerPage: 5})
	if err != nil {
		return 0, fmt.Errorf("github: PRs for commit: %w", err)
	}
	for _, pr := range prs {
		if pr.GetMerged() || pr.MergedAt != nil {
			return pr.GetNumber(), nil
		}
	}
	if len(prs) > 0 {
		return prs[0].GetNumber(), nil
	}
	return 0, nil
}

func (a *Adapter) signal(ctx context.Context, owner, repo string, number int) (lifecycle.PRSignal, error) {
	pr, _, err := a.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return lifecycle.PRSignal{}, fmt.Errorf("github: get PR #%d: %w", number, err)
	}
	s := lifecycle.PRSignal{
		Found:          true,
		Number:         pr.GetNumber(),
		URL:            pr.GetHTMLURL(),
		Open:           pr.GetState() == "open",
		Merged:         pr.GetMerged(),
		MergeCommitSHA: "",
	}
	if s.Merged {
		s.MergeCommitSHA = pr.GetMergeCommitSHA()
	}
	if s.Checks, err = a.checks(ctx, owner, repo, pr.GetHead().GetSHA()); err != nil {
		return lifecycle.PRSignal{}, err
	}
	if s.Approved, s.ChangesRequested, err = a.reviews(ctx, owner, repo, number); err != nil {
		return lifecycle.PRSignal{}, err
	}
	return s, nil
}

// checks aggregates check runs and commit statuses: any failure → failure, any pending → pending.
func (a *Adapter) checks(ctx context.Context, owner, repo, sha string) (lifecycle.CheckState, error) {
	if sha == "" {
		return lifecycle.ChecksNone, nil
	}
	var states []lifecycle.CheckState
	opts := &gh.ListCheckRunsOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		runs, resp, err := a.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, opts)
		if err != nil {
			return "", fmt.Errorf("github: check runs: %w", err)
		}
		for _, r := range runs.CheckRuns {
			states = append(states, checkRunState(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	combined, _, err := a.client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return "", fmt.Errorf("github: combined status: %w", err)
	}
	for _, st := range combined.Statuses {
		switch st.GetState() {
		case "success":
			states = append(states, lifecycle.ChecksSuccess)
		case "pending":
			states = append(states, lifecycle.ChecksPending)
		default:
			states = append(states, lifecycle.ChecksFailure)
		}
	}
	return aggregate(states), nil
}

func checkRunState(r *gh.CheckRun) lifecycle.CheckState {
	if r.GetStatus() != "completed" {
		return lifecycle.ChecksPending
	}
	switch r.GetConclusion() {
	case "success", "neutral", "skipped":
		return lifecycle.ChecksSuccess
	default: // failure, cancelled, timed_out, action_required, stale
		return lifecycle.ChecksFailure
	}
}

func aggregate(states []lifecycle.CheckState) lifecycle.CheckState {
	if len(states) == 0 {
		return lifecycle.ChecksNone
	}
	out := lifecycle.ChecksSuccess
	for _, s := range states {
		switch s {
		case lifecycle.ChecksFailure:
			return lifecycle.ChecksFailure
		case lifecycle.ChecksPending:
			out = lifecycle.ChecksPending
		}
	}
	return out
}

// reviews considers the latest decisive review of each reviewer.
func (a *Adapter) reviews(ctx context.Context, owner, repo string, number int) (approved, changesRequested bool, err error) {
	latest := map[string]string{}
	opts := &gh.ListOptions{PerPage: 100}
	for {
		rs, resp, err := a.client.PullRequests.ListReviews(ctx, owner, repo, number, opts)
		if err != nil {
			return false, false, fmt.Errorf("github: reviews: %w", err)
		}
		for _, r := range rs {
			switch st := r.GetState(); st {
			case "APPROVED", "CHANGES_REQUESTED", "DISMISSED":
				latest[r.GetUser().GetLogin()] = st
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	for _, st := range latest {
		switch st {
		case "APPROVED":
			approved = true
		case "CHANGES_REQUESTED":
			changesRequested = true
		}
	}
	return approved && !changesRequested, changesRequested, nil
}

func splitRepo(full string) (string, string, error) {
	owner, repo, ok := strings.Cut(full, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("github: invalid repository %q (want owner/name)", full)
	}
	return owner, repo, nil
}
