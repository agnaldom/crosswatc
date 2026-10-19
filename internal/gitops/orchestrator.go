package gitops

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/agnaldom/crosswatc/internal/changes"
	"github.com/agnaldom/crosswatc/internal/contract"
)

// Repo is the Git hosting API used by the orchestrator (satisfied by *github.Adapter).
type Repo interface {
	GetFile(ctx context.Context, repository, path, ref string) ([]byte, string, error)
	BranchHead(ctx context.Context, repository, branch string) (string, error)
	CreateBranch(ctx context.Context, repository, name, sha string) error
	CommitFile(ctx context.Context, repository, branch, path string, content []byte, fileSHA, message string) (string, error)
	DeleteFile(ctx context.Context, repository, branch, path, fileSHA, message string) (string, error)
	OpenPullRequest(ctx context.Context, repository, head, base, title, body string) (int, string, error)
	ClosePullRequest(ctx context.Context, repository string, number int, comment string) error
}

// Orchestrator implements readManifest → createBranch → updateManifest → commit → createPullRequest (§44).
type Orchestrator struct {
	Repo       Repo
	BaseBranch string
}

// Request describes one update revision.
type Request struct {
	Repository    string
	Path          string
	ClaimName     string
	Kind          string
	Environment   string
	ResourceName  string
	Rev           int
	CorrelationID string
	RequestedBy   string
	Justification string
	Changes       []changes.SpecChange
	Approval      bool
}

// Result is what was created on the Git host.
type Result struct {
	Branch    string
	CommitSHA string
	PRNumber  int
	PRURL     string
}

// Manifest returns the Claim manifest at the base branch (desired state, §43).
func (o *Orchestrator) Manifest(ctx context.Context, repository, path string) ([]byte, error) {
	b, _, err := o.Repo.GetFile(ctx, repository, path, o.base())
	return b, err
}

func (o *Orchestrator) base() string {
	if o.BaseBranch == "" {
		return "main"
	}
	return o.BaseBranch
}

// Submit opens the Pull Request for req.
func (o *Orchestrator) Submit(ctx context.Context, req Request) (Result, error) {
	base := o.base()
	src, fileSHA, err := o.Repo.GetFile(ctx, req.Repository, req.Path, base)
	if err != nil {
		return Result{}, err
	}
	edits := make([]Edit, len(req.Changes))
	for i, c := range req.Changes {
		edits[i] = Edit{Path: c.Path, Value: c.To}
	}
	updated, err := UpdateManifest(src, req.ClaimName, edits, map[string]string{
		contract.AnnotationRevision:  strconv.Itoa(req.Rev),
		contract.AnnotationRequestID: req.CorrelationID,
	})
	if err != nil {
		return Result{}, err
	}

	head, err := o.Repo.BranchHead(ctx, req.Repository, base)
	if err != nil {
		return Result{}, err
	}
	branch := BranchName("update", req.Kind, req.Environment, req.ResourceName, req.Rev)
	if err := o.Repo.CreateBranch(ctx, req.Repository, branch, head); err != nil {
		return Result{}, err
	}
	sha, err := o.Repo.CommitFile(ctx, req.Repository, branch, req.Path, updated, fileSHA, commitMessage(req))
	if err != nil {
		return Result{}, err
	}
	num, url, err := o.Repo.OpenPullRequest(ctx, req.Repository, branch, base, prTitle(req), prBody(req))
	if err != nil {
		return Result{}, err
	}
	return Result{Branch: branch, CommitSHA: sha, PRNumber: num, PRURL: url}, nil
}

// DeleteRequest describes a decommission (spec §60).
type DeleteRequest struct {
	Repository    string
	Path          string
	ClaimName     string
	Kind          string
	Environment   string
	ResourceName  string
	Rev           int
	CorrelationID string
	RequestedBy   string
	Justification string
	Deleted       []string // cloud resources that will be destroyed (kind/name)
	Orphaned      []string // cloud resources kept (deletionPolicy Orphan)
}

// SubmitDelete opens a PR that removes the Claim from Git; Argo CD prunes it and Crossplane deletes the infra.
func (o *Orchestrator) SubmitDelete(ctx context.Context, req DeleteRequest) (Result, error) {
	base := o.base()
	src, fileSHA, err := o.Repo.GetFile(ctx, req.Repository, req.Path, base)
	if err != nil {
		return Result{}, err
	}
	rest, remaining, err := RemoveClaim(src, req.ClaimName)
	if err != nil {
		return Result{}, err
	}
	head, err := o.Repo.BranchHead(ctx, req.Repository, base)
	if err != nil {
		return Result{}, err
	}
	branch := BranchName("delete", req.Kind, req.Environment, req.ResourceName, req.Rev)
	if err := o.Repo.CreateBranch(ctx, req.Repository, branch, head); err != nil {
		return Result{}, err
	}
	msg := fmt.Sprintf("crosswatc: decommission %s (rev %d)\n\nRequested-by: %s\nCorrelation-Id: %s", req.ResourceName, req.Rev, req.RequestedBy, req.CorrelationID)
	var sha string
	if remaining == 0 {
		sha, err = o.Repo.DeleteFile(ctx, req.Repository, branch, req.Path, fileSHA, msg)
	} else {
		sha, err = o.Repo.CommitFile(ctx, req.Repository, branch, req.Path, rest, fileSHA, msg)
	}
	if err != nil {
		return Result{}, err
	}
	title := fmt.Sprintf("[crosswatc] DECOMMISSION %s (%s)", req.ResourceName, req.Environment)
	num, url, err := o.Repo.OpenPullRequest(ctx, req.Repository, branch, base, title, deleteBody(req))
	if err != nil {
		return Result{}, err
	}
	return Result{Branch: branch, CommitSHA: sha, PRNumber: num, PRURL: url}, nil
}

func deleteBody(r DeleteRequest) string {
	var b strings.Builder
	b.WriteString("> [!CAUTION]\n> Este PR **remove o Claim**. Após o merge, o Argo CD faz prune e o Crossplane **apaga os recursos de cloud** abaixo.\n\n")
	fmt.Fprintf(&b, "| | |\n|---|---|\n| Recurso | `%s` (%s, %s) |\n| Revision | %d |\n| Solicitante | %s |\n| Correlation ID | `%s` |\n| Aprovação | obrigatória |\n",
		r.ResourceName, r.Kind, r.Environment, r.Rev, r.RequestedBy, r.CorrelationID)
	b.WriteString("\n### Será apagado na cloud\n\n")
	if len(r.Deleted) == 0 {
		b.WriteString("_nenhum recurso gerenciado conhecido_\n")
	}
	for _, d := range r.Deleted {
		fmt.Fprintf(&b, "- `%s`\n", d)
	}
	if len(r.Orphaned) > 0 {
		b.WriteString("\n### Mantido (deletionPolicy Orphan)\n\n")
		for _, d := range r.Orphaned {
			fmt.Fprintf(&b, "- `%s`\n", d)
		}
	}
	fmt.Fprintf(&b, "\n### Justificativa\n\n%s\n", r.Justification)
	return b.String()
}

// Cancel closes an open revision PR.
func (o *Orchestrator) Cancel(ctx context.Context, repository string, number int, by, reason string) error {
	msg := fmt.Sprintf("Cancelado no Crosswatc por **%s**.", by)
	if reason != "" {
		msg += "\n\n> " + reason
	}
	return o.Repo.ClosePullRequest(ctx, repository, number, msg)
}

var unsafeRef = regexp.MustCompile(`[^a-z0-9._-]+`)

// BranchName follows §44: {op}/{kind}/{environment}/{resource}/rev-{n}.
func BranchName(op, kind, env, resource string, rev int) string {
	clean := func(s string) string { return strings.Trim(unsafeRef.ReplaceAllString(strings.ToLower(s), "-"), "-.") }
	return fmt.Sprintf("%s/%s/%s/%s/rev-%d", op, clean(kind), clean(env), clean(resource), rev)
}

func prTitle(r Request) string {
	fields := make([]string, len(r.Changes))
	for i, c := range r.Changes {
		fields[i] = c.Field
	}
	return fmt.Sprintf("[crosswatc] %s rev %d: %s", r.ResourceName, r.Rev, strings.Join(fields, ", "))
}

func commitMessage(r Request) string {
	return fmt.Sprintf("crosswatc: update %s (rev %d)\n\nRequested-by: %s\nCorrelation-Id: %s", r.ResourceName, r.Rev, r.RequestedBy, r.CorrelationID)
}

func prBody(r Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Mudança solicitada pelo **Crosswatc**.\n\n")
	fmt.Fprintf(&b, "| | |\n|---|---|\n| Recurso | `%s` (%s, %s) |\n| Revision | %d |\n| Solicitante | %s |\n| Correlation ID | `%s` |\n",
		r.ResourceName, r.Kind, r.Environment, r.Rev, r.RequestedBy, r.CorrelationID)
	if r.Approval {
		b.WriteString("| Aprovação | obrigatória |\n")
	}
	b.WriteString("\n### Alterações\n\n| Campo | Antes | Depois | Impacto |\n|---|---|---|---|\n")
	for _, c := range r.Changes {
		fmt.Fprintf(&b, "| `%s` | `%v` | `%v` | %s |\n", c.Field, c.From, c.To, c.Impact)
	}
	if r.Justification != "" {
		fmt.Fprintf(&b, "\n### Justificativa\n\n%s\n", r.Justification)
	}
	return b.String()
}
