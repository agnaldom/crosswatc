package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v93/github"
)

// ErrNotFound is returned when a file, branch or ref does not exist.
var ErrNotFound = errors.New("github: not found")

func notFound(resp *gh.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	return err
}

// GetFile returns the content and blob SHA of path at ref.
func (a *Adapter) GetFile(ctx context.Context, repository, path, ref string) ([]byte, string, error) {
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return nil, "", err
	}
	fc, _, resp, err := a.client.Repositories.GetContents(ctx, owner, repo, path, &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, "", fmt.Errorf("github: get %s@%s: %w", path, ref, notFound(resp, err))
	}
	if fc == nil {
		return nil, "", fmt.Errorf("github: %s is a directory", path)
	}
	content, err := fc.GetContent()
	if err != nil {
		return nil, "", err
	}
	return []byte(content), fc.GetSHA(), nil
}

// BranchHead returns the commit SHA at the tip of branch.
func (a *Adapter) BranchHead(ctx context.Context, repository, branch string) (string, error) {
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return "", err
	}
	ref, resp, err := a.client.Git.GetRef(ctx, owner, repo, "heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("github: ref %s: %w", branch, notFound(resp, err))
	}
	return ref.GetObject().GetSHA(), nil
}

// CreateBranch creates refs/heads/name at sha.
func (a *Adapter) CreateBranch(ctx context.Context, repository, name, sha string) error {
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return err
	}
	if _, _, err := a.client.Git.CreateRef(ctx, owner, repo, gh.CreateRef{Ref: "refs/heads/" + name, SHA: sha}); err != nil {
		return fmt.Errorf("github: create branch %s: %w", name, err)
	}
	return nil
}

// CommitFile writes content to path on branch, replacing the blob fileSHA. Returns the commit SHA.
func (a *Adapter) CommitFile(ctx context.Context, repository, branch, path string, content []byte, fileSHA, message string) (string, error) {
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return "", err
	}
	res, _, err := a.client.Repositories.UpdateFile(ctx, owner, repo, path, &gh.RepositoryContentFileOptions{
		Message: &message, Content: content, SHA: &fileSHA, Branch: &branch,
	})
	if err != nil {
		return "", fmt.Errorf("github: commit %s: %w", path, err)
	}
	return res.GetSHA(), nil
}

// OpenPullRequest opens head → base and returns its number and URL.
func (a *Adapter) OpenPullRequest(ctx context.Context, repository, head, base, title, body string) (int, string, error) {
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return 0, "", err
	}
	pr, _, err := a.client.PullRequests.Create(ctx, owner, repo, gh.CreatePullRequest{Title: &title, Head: head, Base: base, Body: &body})
	if err != nil {
		return 0, "", fmt.Errorf("github: open PR: %w", err)
	}
	return pr.GetNumber(), pr.GetHTMLURL(), nil
}

// ClosePullRequest comments and closes a PR without merging.
func (a *Adapter) ClosePullRequest(ctx context.Context, repository string, number int, comment string) error {
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return err
	}
	if comment != "" {
		if _, _, err := a.client.Issues.CreateComment(ctx, owner, repo, number, gh.IssueCommentRequest{Body: comment}); err != nil {
			return fmt.Errorf("github: comment PR #%d: %w", number, err)
		}
	}
	closed := "closed"
	if _, _, err := a.client.PullRequests.Edit(ctx, owner, repo, number, &gh.PullRequest{State: &closed}); err != nil {
		return fmt.Errorf("github: close PR #%d: %w", number, err)
	}
	return nil
}
