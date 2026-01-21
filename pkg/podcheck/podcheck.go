package podcheck

import (
	"context"
	"fmt"

	"github.com/eumel8/clustercheck/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// CheckPods checks if all pods in the cluster are in Running or Succeeded state
func CheckPods(namespace string, debug bool) error {
	kubeconfigPath := common.GetKubeConfig()

	if debug {
		fmt.Printf("\n[DEBUG] Kubernetes API Request:\n")
		fmt.Printf("  Kubeconfig: %s\n", kubeconfigPath)
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build config: %v", err)
	}

	if debug {
		fmt.Printf("  API Server: %s\n", config.Host)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %v", err)
	}

	// Get current context for display
	currentContext, err := common.GetCurrentContext()
	if err != nil {
		currentContext = "unknown"
	}

	fmt.Printf("\033[36mpodcheck \033[0m on %s\n", currentContext)

	// List pods
	ctx := context.Background()
	listOptions := metav1.ListOptions{}

	if debug {
		if namespace == "" {
			fmt.Printf("  Operation: List Pods (all namespaces)\n")
		} else {
			fmt.Printf("  Operation: List Pods (namespace: %s)\n", namespace)
		}
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	totalPods := len(pods.Items)

	if debug {
		fmt.Printf("[DEBUG] Kubernetes API Response:\n")
		fmt.Printf("  Total Pods: %d\n\n", totalPods)
	}
	runningOrSucceeded := 0
	failedPods := []string{}

	for _, pod := range pods.Items {
		phase := string(pod.Status.Phase)
		podName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		if phase == "Running" || phase == "Succeeded" {
			runningOrSucceeded++
			fmt.Printf("%s \033[32m🟢 %s\033[0m\n", podName, phase)
		} else {
			failedPods = append(failedPods, fmt.Sprintf("%s (%s)", podName, phase))
			fmt.Printf("%s \033[31m🔴 %s\033[0m\n", podName, phase)
		}
	}

	fmt.Printf("\nSummary: %d/%d pods in Running or Succeeded state\n", runningOrSucceeded, totalPods)

	if len(failedPods) > 0 {
		fmt.Printf("\033[31mFailed pods:\033[0m\n")
		for _, pod := range failedPods {
			fmt.Printf("  - %s\n", pod)
		}
		return fmt.Errorf("%d pods not in Running or Succeeded state", len(failedPods))
	}

	return nil
}
