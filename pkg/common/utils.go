package common

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig returns the path to the kubeconfig file
func GetKubeConfig() string {
	// Check if KUBECONFIG env var is set
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	// Default to $HOME/.kube/config
	return filepath.Join(os.Getenv("HOME"), ".kube", "config")
}

// GetCurrentContext returns the current kubernetes context
func GetCurrentContext() (string, error) {
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}
