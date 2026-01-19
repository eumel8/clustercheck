package podcheck

import (
	"os"
	"strings"
	"testing"
)

func TestCheckPodsWithInvalidConfig(t *testing.T) {
	// Save original env vars
	originalKubeConfig := os.Getenv("KUBECONFIG")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("KUBECONFIG", originalKubeConfig)
		os.Setenv("HOME", originalHome)
	}()

	// Set invalid kubeconfig path
	os.Setenv("KUBECONFIG", "/nonexistent/path/to/kubeconfig")

	err := CheckPods("", false)
	if err == nil {
		t.Error("Expected error for invalid kubeconfig, got nil")
	}

	if !strings.Contains(err.Error(), "failed to build config") {
		t.Errorf("Expected 'failed to build config' error, got: %v", err)
	}
}
