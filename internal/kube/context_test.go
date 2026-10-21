package kube

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContextManager(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	initialYAML := `apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-ctx-1
- context:
    cluster: test-cluster
    user: test-user
  name: test-ctx-2
current-context: test-ctx-1
kind: Config
preferences: {}
users:
- name: test-user
  user:
    token: fake-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(initialYAML), 0644); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	cm := &DefaultContextManager{
		kubeconfigPath: kubeconfigPath,
		inCluster:      false,
	}

	contexts, current, inCluster, err := cm.GetContexts()
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if inCluster {
		t.Errorf("expected not in-cluster")
	}
	if current != "test-ctx-1" {
		t.Errorf("expected current context test-ctx-1, got %s", current)
	}
	if len(contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(contexts))
	}

	// Switch context
	if err := cm.SetContext("test-ctx-2"); err != nil {
		t.Fatalf("SetContext failed: %v", err)
	}
	_, current, _, _ = cm.GetContexts()
	if current != "test-ctx-2" {
		t.Errorf("expected current context test-ctx-2, got %s", current)
	}

	// Add context via YAML
	addYAML := `apiVersion: v1
clusters:
- cluster:
    server: https://192.168.1.100:6443
  name: extra-cluster
contexts:
- context:
    cluster: extra-cluster
    user: extra-user
  name: test-ctx-3
kind: Config
users:
- name: extra-user
  user:
    token: another-token
`
	added, err := cm.AddKubeConfig(addYAML)
	if err != nil {
		t.Fatalf("AddKubeConfig failed: %v", err)
	}
	if len(added) != 1 || added[0] != "test-ctx-3" {
		t.Errorf("expected test-ctx-3 to be added, got %v", added)
	}

	contexts, _, _, _ = cm.GetContexts()
	if len(contexts) != 3 {
		t.Errorf("expected 3 contexts, got %d", len(contexts))
	}

	// Remove context
	if err := cm.RemoveContext("test-ctx-3"); err != nil {
		t.Fatalf("RemoveContext failed: %v", err)
	}
	contexts, _, _, _ = cm.GetContexts()
	if len(contexts) != 2 {
		t.Errorf("expected 2 contexts after removal, got %d", len(contexts))
	}
}
