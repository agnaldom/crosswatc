package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	base := write(t, dir, "base.yaml", "metadata: {name: vm-payments}\nspec:\n  parameters:\n    vmSize: D2\n    tags: {cost-center: cc1}\n")
	okHead := write(t, dir, "ok.yaml", "metadata: {name: vm-payments}\nspec:\n  parameters:\n    vmSize: D4\n    tags: {cost-center: cc1}\n")
	badHead := write(t, dir, "bad.yaml", "metadata: {name: vm-payments}\nspec:\n  parameters:\n    vmSize: D4\n    publicNetworkAccess: Enabled\n    tags: {cost-center: \"\"}\n")

	if code := run([]string{"-base", base, "-head", okHead, "-name", "vm-payments", "-environment", "prd"}); code != 0 {
		t.Fatalf("clean change: exit %d", code)
	}
	if code := run([]string{"-base", base, "-head", badHead, "-name", "vm-payments", "-json"}); code != 1 {
		t.Fatalf("violations: exit %d", code)
	}
	if code := run([]string{"-head", okHead, "-name", "Bad_Name"}); code != 1 {
		t.Fatalf("new Claim with invalid name must violate naming: exit %d", code)
	}
	if code := run([]string{"-head", filepath.Join(dir, "missing.yaml"), "-name", "x"}); code != 2 {
		t.Fatalf("unreadable manifest: exit %d", code)
	}
	if code := run([]string{}); code != 2 {
		t.Fatalf("usage: exit %d", code)
	}
}
