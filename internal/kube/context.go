package kube

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ContextManager defines the interface for managing Kubernetes contexts.
type ContextManager interface {
	GetContexts() ([]string, string, bool, error)
	SetContext(name string) error
	CheckConnection(name string) (bool, error)
	AddKubeConfig(yamlData string) ([]string, error)
	RemoveContext(name string) error
}

// DefaultContextManager implements ContextManager using client-go clientcmd.
type DefaultContextManager struct {
	mu             sync.RWMutex
	kubeconfigPath string
	inCluster      bool
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isInClusterEnvironment() bool {
	serviceAccountPath := "/var/run/secrets/kubernetes.io/serviceaccount"
	return fileExists(filepath.Join(serviceAccountPath, "token")) &&
		fileExists(filepath.Join(serviceAccountPath, "ca.crt"))
}

// NewContextManager creates a ContextManager with the specified kubeconfig path.
func NewContextManager(kubeconfigPath string) ContextManager {
	if kubeconfigPath == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			kubeconfigPath = env
		} else if home, err := os.UserHomeDir(); err == nil {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}
	return &DefaultContextManager{
		kubeconfigPath: kubeconfigPath,
		inCluster:      isInClusterEnvironment(),
	}
}

func (m *DefaultContextManager) loadConfig() (*clientcmdapi.Config, error) {
	if m.kubeconfigPath == "" {
		return nil, errors.New("kube: no kubeconfig path specified")
	}
	if !fileExists(m.kubeconfigPath) {
		return clientcmdapi.NewConfig(), nil
	}
	return clientcmd.LoadFromFile(m.kubeconfigPath)
}

func (m *DefaultContextManager) saveConfig(cfg *clientcmdapi.Config) error {
	if m.kubeconfigPath == "" {
		return errors.New("kube: no kubeconfig path specified")
	}
	dir := filepath.Dir(m.kubeconfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("kube: create directory %s: %w", dir, err)
	}
	return clientcmd.WriteToFile(*cfg, m.kubeconfigPath)
}

// GetContexts returns available contexts, the active context, and whether running in-cluster.
func (m *DefaultContextManager) GetContexts() ([]string, string, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.inCluster {
		return []string{"in-cluster"}, "in-cluster", true, nil
	}

	cfg, err := m.loadConfig()
	if err != nil {
		return nil, "", false, err
	}

	contexts := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		contexts = append(contexts, name)
	}
	if len(contexts) == 0 {
		return []string{"default"}, "default", false, nil
	}

	current := cfg.CurrentContext
	if current == "" && len(contexts) > 0 {
		current = contexts[0]
	}

	return contexts, current, false, nil
}

// SetContext changes the active context in the kubeconfig file.
func (m *DefaultContextManager) SetContext(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inCluster {
		return errors.New("cannot change context in in-cluster mode")
	}

	cfg, err := m.loadConfig()
	if err != nil {
		return err
	}

	if _, exists := cfg.Contexts[name]; !exists {
		return fmt.Errorf("context '%s' not found", name)
	}

	cfg.CurrentContext = name
	return m.saveConfig(cfg)
}

// CheckConnection tests connectivity to the specified context.
func (m *DefaultContextManager) CheckConnection(name string) (bool, error) {
	m.mu.RLock()
	inCluster := m.inCluster
	path := m.kubeconfigPath
	m.mu.RUnlock()

	var restCfg *rest.Config
	var err error

	if inCluster {
		restCfg, err = rest.InClusterConfig()
		if err != nil {
			return false, err
		}
	} else {
		if path == "" || !fileExists(path) {
			return false, errors.New("kubeconfig not found")
		}
		cfg, errLoad := clientcmd.LoadFromFile(path)
		if errLoad != nil {
			return false, errLoad
		}
		targetContext := name
		if targetContext == "" {
			targetContext = cfg.CurrentContext
		}
		if _, exists := cfg.Contexts[targetContext]; !exists {
			return false, fmt.Errorf("context '%s' not found", targetContext)
		}
		restCfg, err = clientcmd.NewDefaultClientConfig(*cfg, &clientcmd.ConfigOverrides{
			CurrentContext: targetContext,
		}).ClientConfig()
		if err != nil {
			return false, err
		}
	}

	restCfg.Timeout = 5 * time.Second
	dc, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = dc.RESTClient().Get().AbsPath("/healthz").DoRaw(ctx)
	if err == nil {
		return true, nil
	}
	_, err = dc.ServerVersion()
	if err == nil {
		return true, nil
	}
	return false, err
}

// AddKubeConfig merges a kubeconfig YAML snippet into the local kubeconfig.
func (m *DefaultContextManager) AddKubeConfig(yamlData string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inCluster {
		return nil, errors.New("cannot add contexts in in-cluster mode")
	}

	newCfg, err := clientcmd.Load([]byte(yamlData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig YAML: %w", err)
	}

	if len(newCfg.Contexts) == 0 {
		return nil, errors.New("kubeconfig does not contain any contexts")
	}

	existingCfg, err := m.loadConfig()
	if err != nil {
		return nil, err
	}

	var added []string
	for name, ctx := range newCfg.Contexts {
		if _, exists := existingCfg.Contexts[name]; !exists {
			added = append(added, name)
		}
		existingCfg.Contexts[name] = ctx
	}
	for name, cluster := range newCfg.Clusters {
		existingCfg.Clusters[name] = cluster
	}
	for name, authInfo := range newCfg.AuthInfos {
		existingCfg.AuthInfos[name] = authInfo
	}

	if existingCfg.CurrentContext == "" && len(added) > 0 {
		existingCfg.CurrentContext = added[0]
	}

	if err := m.saveConfig(existingCfg); err != nil {
		return nil, err
	}

	return added, nil
}

// RemoveContext deletes a context and cleans up unused clusters and auth credentials.
func (m *DefaultContextManager) RemoveContext(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inCluster {
		return errors.New("cannot remove context in in-cluster mode")
	}

	existingCfg, err := m.loadConfig()
	if err != nil {
		return err
	}

	ctx, exists := existingCfg.Contexts[name]
	if !exists {
		return fmt.Errorf("context '%s' not found", name)
	}

	clusterName := ctx.Cluster
	authName := ctx.AuthInfo

	delete(existingCfg.Contexts, name)

	if existingCfg.CurrentContext == name {
		existingCfg.CurrentContext = ""
		for otherName := range existingCfg.Contexts {
			existingCfg.CurrentContext = otherName
			break
		}
	}

	clusterInUse := false
	authInUse := false
	for _, c := range existingCfg.Contexts {
		if c.Cluster == clusterName {
			clusterInUse = true
		}
		if c.AuthInfo == authName {
			authInUse = true
		}
	}
	if !clusterInUse && clusterName != "" {
		delete(existingCfg.Clusters, clusterName)
	}
	if !authInUse && authName != "" {
		delete(existingCfg.AuthInfos, authName)
	}

	return m.saveConfig(existingCfg)
}
