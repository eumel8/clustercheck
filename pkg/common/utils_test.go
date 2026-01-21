package common

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetCurrentContext(t *testing.T) {
	// Create a temporary kubeconfig file for testing
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

	// Create a test kubeconfig
	config := &clientcmdapi.Config{
		CurrentContext: "test-context",
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster:  "test-cluster",
				AuthInfo: "test-user",
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": {
				Server: "https://test-server",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test-user": {
				Token: "test-token",
			},
		},
	}

	err := clientcmd.WriteToFile(*config, kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create .kube directory
	kubeDir := filepath.Join(tempDir, ".kube")
	err = os.MkdirAll(kubeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .kube directory: %v", err)
	}

	// Copy config to .kube/config
	kubePath := filepath.Join(kubeDir, "config")
	err = clientcmd.WriteToFile(*config, kubePath)
	if err != nil {
		t.Fatalf("Failed to write kubeconfig to .kube/config: %v", err)
	}

	context, err := GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext() returned error: %v", err)
	}

	if context != "test-context" {
		t.Errorf("Expected context 'test-context', got '%s'", context)
	}
}

func TestGetCurrentContextError(t *testing.T) {
	// Test with non-existent kubeconfig
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", "/nonexistent")

	_, err := GetCurrentContext()
	if err == nil {
		t.Error("Expected error for non-existent kubeconfig, got nil")
	}
}

func TestGetKubeConfig(t *testing.T) {
	// Save original env vars
	originalKubeConfig := os.Getenv("KUBECONFIG")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("KUBECONFIG", originalKubeConfig)
		os.Setenv("HOME", originalHome)
	}()

	t.Run("KUBECONFIG env var set", func(t *testing.T) {
		expected := "/custom/path/to/kubeconfig"
		os.Setenv("KUBECONFIG", expected)

		result := GetKubeConfig()
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("KUBECONFIG not set, use HOME", func(t *testing.T) {
		os.Setenv("KUBECONFIG", "")
		os.Setenv("HOME", "/test/home")

		expected := "/test/home/.kube/config"
		result := GetKubeConfig()
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
}
