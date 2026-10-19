package gitops

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agnaldom/crosswatc/internal/changes"
)

type fakeRepo struct {
	files    map[string][]byte
	branches map[string]string
	commits  map[string][]byte // branch → content
	deleted  map[string]string // branch → path
	prs      []string
	closed   []int
	failOn   string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{files: map[string][]byte{"claims/dev/vm.yaml": []byte(claim)}, branches: map[string]string{"main": "base1"}, commits: map[string][]byte{}, deleted: map[string]string{}}
}

func (f *fakeRepo) err(op string) error {
	if f.failOn == op {
		return errors.New(op + " failed")
	}
	return nil
}
func (f *fakeRepo) GetFile(_ context.Context, _, path, _ string) ([]byte, string, error) {
	return f.files[path], "blob1", f.err("get")
}
func (f *fakeRepo) BranchHead(_ context.Context, _, b string) (string, error) {
	return f.branches[b], f.err("head")
}
func (f *fakeRepo) CreateBranch(_ context.Context, _, name, sha string) error {
	if _, ok := f.branches[name]; ok {
		return errors.New("exists")
	}
	f.branches[name] = sha
	return f.err("branch")
}
func (f *fakeRepo) CommitFile(_ context.Context, _, branch, _ string, content []byte, fileSHA, msg string) (string, error) {
	if fileSHA != "blob1" || !strings.Contains(msg, "Correlation-Id: crosswatc-req-1") {
		return "", errors.New("bad commit args")
	}
	f.commits[branch] = content
	return "commit1", f.err("commit")
}
func (f *fakeRepo) DeleteFile(_ context.Context, _, branch, path, fileSHA, _ string) (string, error) {
	if fileSHA != "blob1" {
		return "", errors.New("bad sha")
	}
	f.deleted[branch] = path
	return "commit-del", f.err("delete")
}
func (f *fakeRepo) OpenPullRequest(_ context.Context, _, head, base, title, body string) (int, string, error) {
	f.prs = append(f.prs, head+"→"+base+"|"+title+"|"+body)
	return 173, "https://github.com/org/argocd/pull/173", f.err("pr")
}
func (f *fakeRepo) ClosePullRequest(_ context.Context, _ string, n int, _ string) error {
	f.closed = append(f.closed, n)
	return nil
}

func req() Request {
	return Request{Repository: "org/argocd", Path: "claims/dev/vm.yaml", ClaimName: "vm-payments", Kind: "VirtualMachine",
		Environment: "dev", ResourceName: "vm-payments", Rev: 8, CorrelationID: "crosswatc-req-1", RequestedBy: "agnaldo",
		Justification: "Aumento de capacidade.", Approval: true,
		Changes: []changes.SpecChange{{Field: "vmSize", Path: "spec.parameters.vmSize", From: "Standard_D2as_v5", To: "Standard_D4as_v5", Impact: changes.ImpactInPlace}}}
}

func TestSubmit(t *testing.T) {
	repo := newFakeRepo()
	o := &Orchestrator{Repo: repo}
	res, err := o.Submit(context.Background(), req())
	if err != nil {
		t.Fatal(err)
	}
	want := Result{Branch: "update/virtualmachine/dev/vm-payments/rev-8", CommitSHA: "commit1", PRNumber: 173, PRURL: "https://github.com/org/argocd/pull/173"}
	if res != want {
		t.Fatalf("%+v", res)
	}
	if repo.branches[want.Branch] != "base1" {
		t.Fatal("branch must start at base head")
	}
	body := string(repo.commits[want.Branch])
	if !strings.Contains(body, "vmSize: Standard_D4as_v5 # sized for dev") || !strings.Contains(body, `crosswatc.io/revision: "8"`) {
		t.Fatalf("manifest:\n%s", body)
	}
	pr := repo.prs[0]
	for _, s := range []string{"update/virtualmachine/dev/vm-payments/rev-8→main", "vm-payments rev 8: vmSize", "crosswatc-req-1", "Aumento de capacidade.", "obrigatória", "`Standard_D2as_v5` | `Standard_D4as_v5`"} {
		if !strings.Contains(pr, s) {
			t.Errorf("PR missing %q", s)
		}
	}

	if _, err := o.Submit(context.Background(), req()); err == nil {
		t.Fatal("same revision twice must fail on existing branch")
	}
}

func TestSubmitFailures(t *testing.T) {
	for _, op := range []string{"get", "head", "commit", "pr"} {
		repo := newFakeRepo()
		repo.failOn = op
		if _, err := (&Orchestrator{Repo: repo, BaseBranch: "main"}).Submit(context.Background(), req()); err == nil {
			t.Errorf("%s failure must propagate", op)
		}
	}
	repo := newFakeRepo()
	r := req()
	r.ClaimName = "other"
	repo.files["claims/dev/vm.yaml"] = []byte("a: 1\n---\nb: 2\n")
	if _, err := (&Orchestrator{Repo: repo}).Submit(context.Background(), r); !errors.Is(err, ErrClaimNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestCancelAndManifest(t *testing.T) {
	repo := newFakeRepo()
	o := &Orchestrator{Repo: repo}
	if err := o.Cancel(context.Background(), "org/argocd", 173, "agnaldo", "não precisa mais"); err != nil || repo.closed[0] != 173 {
		t.Fatal(err)
	}
	if b, err := o.Manifest(context.Background(), "org/argocd", "claims/dev/vm.yaml"); err != nil || !strings.Contains(string(b), "vm-payments") {
		t.Fatal(err)
	}
}

func TestBranchName(t *testing.T) {
	if got := BranchName("update", "PostgreSQL Instance", "PRD", "pg_payments/../x", 3); got != "update/postgresql-instance/prd/pg_payments-..-x/rev-3" {
		t.Fatalf("%q", got)
	}
}

func delReq() DeleteRequest {
	return DeleteRequest{Repository: "org/argocd", Path: "claims/dev/vm.yaml", ClaimName: "vm-payments", Kind: "VirtualMachine",
		Environment: "dev", ResourceName: "vm-payments", Rev: 9, CorrelationID: "crosswatc-req-1", RequestedBy: "agnaldo",
		Justification: "Serviço desativado", Deleted: []string{"LinuxVirtualMachine/vm-1"}, Orphaned: []string{"ManagedDisk/data"}}
}

func TestSubmitDeleteSingleDocDeletesFile(t *testing.T) {
	repo := newFakeRepo()
	res, err := (&Orchestrator{Repo: repo}).SubmitDelete(context.Background(), delReq())
	if err != nil {
		t.Fatal(err)
	}
	if res.Branch != "delete/virtualmachine/dev/vm-payments/rev-9" || res.CommitSHA != "commit-del" || repo.deleted[res.Branch] != "claims/dev/vm.yaml" {
		t.Fatalf("%+v %v", res, repo.deleted)
	}
	pr := repo.prs[0]
	for _, s := range []string{"DECOMMISSION vm-payments", "CAUTION", "LinuxVirtualMachine/vm-1", "ManagedDisk/data", "Serviço desativado"} {
		if !strings.Contains(pr, s) {
			t.Errorf("PR missing %q", s)
		}
	}
}

func TestSubmitDeleteMultiDocKeepsOthers(t *testing.T) {
	repo := newFakeRepo()
	repo.files["claims/dev/vm.yaml"] = []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: keep-me # comment\n---\n" + claim)
	res, err := (&Orchestrator{Repo: repo}).SubmitDelete(context.Background(), delReq())
	if err != nil {
		t.Fatal(err)
	}
	body := string(repo.commits[res.Branch])
	if !strings.Contains(body, "keep-me # comment") || strings.Contains(body, "vm-payments") || repo.deleted[res.Branch] != "" {
		t.Fatalf("%s", body)
	}
	r := delReq()
	r.ClaimName = "ghost"
	if _, err := (&Orchestrator{Repo: newFakeRepoMulti()}).SubmitDelete(context.Background(), r); !errors.Is(err, ErrClaimNotFound) {
		t.Fatalf("got %v", err)
	}
}

func newFakeRepoMulti() *fakeRepo {
	r := newFakeRepo()
	r.files["claims/dev/vm.yaml"] = []byte("metadata: {name: a}\n---\nmetadata: {name: b}\n")
	return r
}

func TestPRBodyMasksSensitiveValues(t *testing.T) {
	r := req()
	r.Changes = append(r.Changes, changes.SpecChange{Field: "adminPassword", Path: "spec.parameters.adminPassword", From: "[REDACTED]", To: "n3w"})
	if strings.Contains(prBody(r), "n3w") {
		t.Fatal("PR body must not contain the secret")
	}
}
