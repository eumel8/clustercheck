package fluxcheck

import (
	"context"
	"fmt"

	"github.com/eumel8/clustercheck/pkg/common"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckFlux checks if all Flux HelmReleases and Kustomizations are in Ready state
func CheckFlux(namespace string, debug bool) error {
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

	// Get current context for display
	currentContext, err := common.GetCurrentContext()
	if err != nil {
		currentContext = "unknown"
	}

	fmt.Printf("\033[36mfluxcheck \033[0m on %s\n", currentContext)

	// Create a new scheme and add Flux types
	fluxScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(fluxScheme)
	_ = helmv2.AddToScheme(fluxScheme)
	_ = kustomizev1.AddToScheme(fluxScheme)

	// Create controller-runtime client
	k8sClient, err := client.New(config, client.Options{Scheme: fluxScheme})
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	ctx := context.Background()
	totalResources := 0
	readyResources := 0
	failedResources := []string{}

	// Check HelmReleases
	helmReleaseList := &helmv2.HelmReleaseList{}
	listOpts := []client.ListOption{}
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	if debug {
		if namespace == "" {
			fmt.Printf("  Operation: List HelmReleases (all namespaces)\n")
		} else {
			fmt.Printf("  Operation: List HelmReleases (namespace: %s)\n", namespace)
		}
	}

	err = k8sClient.List(ctx, helmReleaseList, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list HelmReleases: %v", err)
	}

	if debug {
		fmt.Printf("[DEBUG] Kubernetes API Response:\n")
		fmt.Printf("  HelmReleases found: %d\n", len(helmReleaseList.Items))
	}

	fmt.Printf("\n\033[1mHelmReleases:\033[0m\n")
	for _, hr := range helmReleaseList.Items {
		totalResources++
		resourceName := fmt.Sprintf("%s/%s", hr.Namespace, hr.Name)
		ready := false

		// Check Ready condition
		for _, condition := range hr.Status.Conditions {
			if condition.Type == "Ready" {
				if condition.Status == metav1.ConditionTrue {
					ready = true
					readyResources++
					fmt.Printf("%s \033[32m🟢 Ready\033[0m (revision: %s)\n", resourceName, hr.Status.LastAttemptedRevision)
				} else {
					failedResources = append(failedResources, fmt.Sprintf("HelmRelease %s: %s", resourceName, condition.Message))
					fmt.Printf("%s \033[31m🔴 Not Ready\033[0m - %s\n", resourceName, condition.Message)
				}
				break
			}
		}

		if !ready && len(hr.Status.Conditions) == 0 {
			failedResources = append(failedResources, fmt.Sprintf("HelmRelease %s: No conditions set", resourceName))
			fmt.Printf("%s \033[33m⚠️  Unknown\033[0m - No conditions set\n", resourceName)
		}
	}

	// Check Kustomizations
	kustomizationList := &kustomizev1.KustomizationList{}

	if debug {
		if namespace == "" {
			fmt.Printf("  Operation: List Kustomizations (all namespaces)\n")
		} else {
			fmt.Printf("  Operation: List Kustomizations (namespace: %s)\n", namespace)
		}
	}

	err = k8sClient.List(ctx, kustomizationList, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list Kustomizations: %v", err)
	}

	if debug {
		fmt.Printf("[DEBUG] Kubernetes API Response:\n")
		fmt.Printf("  Kustomizations found: %d\n\n", len(kustomizationList.Items))
	}

	fmt.Printf("\n\033[1mKustomizations:\033[0m\n")
	for _, ks := range kustomizationList.Items {
		totalResources++
		resourceName := fmt.Sprintf("%s/%s", ks.Namespace, ks.Name)
		ready := false

		// Check Ready condition
		for _, condition := range ks.Status.Conditions {
			if condition.Type == "Ready" {
				if condition.Status == metav1.ConditionTrue {
					ready = true
					readyResources++
					fmt.Printf("%s \033[32m🟢 Ready\033[0m (revision: %s)\n", resourceName, ks.Status.LastAppliedRevision)
				} else {
					failedResources = append(failedResources, fmt.Sprintf("Kustomization %s: %s", resourceName, condition.Message))
					fmt.Printf("%s \033[31m🔴 Not Ready\033[0m - %s\n", resourceName, condition.Message)
				}
				break
			}
		}

		if !ready && len(ks.Status.Conditions) == 0 {
			failedResources = append(failedResources, fmt.Sprintf("Kustomization %s: No conditions set", resourceName))
			fmt.Printf("%s \033[33m⚠️  Unknown\033[0m - No conditions set\n", resourceName)
		}
	}

	fmt.Printf("\n\033[1mSummary:\033[0m %d/%d resources Ready\n", readyResources, totalResources)

	if len(failedResources) > 0 {
		fmt.Printf("\033[31m\nFailed resources:\033[0m\n")
		for _, resource := range failedResources {
			fmt.Printf("  - %s\n", resource)
		}
		return fmt.Errorf("%d resources not Ready", len(failedResources))
	}

	if totalResources == 0 {
		fmt.Printf("\033[33mNo Flux resources found\033[0m\n")
	}

	return nil
}
