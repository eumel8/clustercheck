package monitoringcheck

// Kubernetes Healthy Cluster Speed Check
// This script checks the health of a Kubernetes cluster by querying Prometheus for various metrics.
// It uses the Bitwarden CLI to retrieve Prometheus credentials and performs checks on various components.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Struct to hold Bitwarden login fields
type BitwardenItem struct {
	Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"login"`
}

// Get BW_SESSION from env
func getSessionToken() string {
	return os.Getenv("BW_SESSION")
}

// Run Bitwarden CLI to get the item JSON
func getBitwardenItemJSON(itemName string) ([]byte, error) {
	cmd := exec.Command("bw", "get", "item", itemName)
	cmd.Env = append(os.Environ(), "BW_SESSION="+getSessionToken())

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// Prometheus response struct
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Prometheus query struct
type PrometheusQueries struct {
	Description string `json:"description"`
	Query       string `json:"query"`
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

// QueryPrometheus queries Prometheus with the given parameters
func QueryPrometheus(prometheus string, query string, username string, password string) (string, error) {
	value := "0"
	params := url.Values{}
	params.Add("query", query)
	url := fmt.Sprintf("%s/api/v1/query?%s", prometheus, params.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return value, err
	}
	req.SetBasicAuth(username, password)

	// skip TLS verification
	insecureClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := insecureClient.Do(req)
	if err != nil {
		return value, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return value, err
	}

	// Define a structure matching the Prometheus response
	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  [2]interface{}    `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	// Parse JSON response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return value, err
	}

	// Extract the value (second element in the Value array)
	if len(result.Data.Result) > 0 {
		value = result.Data.Result[0].Value[1].(string)
	}

	return value, nil
}

// getKubeConfig returns the path to the kubeconfig file
func getKubeConfig() string {
	// Check if KUBECONFIG env var is set
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	// Default to $HOME/.kube/config
	return filepath.Join(os.Getenv("HOME"), ".kube", "config")
}

// CheckPods checks if all pods in the cluster are in Running or Succeeded state
func CheckPods(namespace string) error {
	kubeconfigPath := getKubeConfig()

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build config: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %v", err)
	}

	// Get current context for display
	currentContext, err := GetCurrentContext()
	if err != nil {
		currentContext = "unknown"
	}

	fmt.Printf("\033[36mpodcheck \033[0m on %s\n", currentContext)

	// List pods
	ctx := context.Background()
	listOptions := metav1.ListOptions{}
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	totalPods := len(pods.Items)
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

// CheckFlux checks if all Flux HelmReleases and Kustomizations are in Ready state
func CheckFlux(namespace string) error {
	kubeconfigPath := getKubeConfig()

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build config: %v", err)
	}

	// Get current context for display
	currentContext, err := GetCurrentContext()
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

	err = k8sClient.List(ctx, helmReleaseList, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list HelmReleases: %v", err)
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
	err = k8sClient.List(ctx, kustomizationList, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list Kustomizations: %v", err)
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

// Run executes the cluster health check with the given options
func Run(bitwarden bool, fqdn string) {
	// static Prometheus API endpoint
	prometheus := "https://127.0.0.1:9090"
	username := os.Getenv("PROM_USER")
	password := os.Getenv("PROM_PASS")
	clcBW := os.Getenv("CLUSTERCHECK_BW")
	clcFQDN := os.Getenv("CLUSTERCHECK_FQDN")

	if bitwarden == true || clcBW != "" {
		// doing bitwarden stuff here to get prometheus credentials
		itemName := "Prometheus Agent RemoteWrite"
		jsonData, err := getBitwardenItemJSON(itemName)
		if err != nil {
			fmt.Printf("Failed to get item from Bitwarden: %v\n", err)
		}

		var item BitwardenItem
		err = json.Unmarshal(jsonData, &item)
		if err != nil {
			fmt.Printf("Failed to parse Bitwarden JSON: %v\n", err)
		}

		username = item.Login.Username
		password = item.Login.Password
	}

	cluster, err := GetCurrentContext()
	if err != nil {
		fmt.Printf("Failed to get current kube context: %v\n", err)
	}

	shortCluster := cluster

	if fqdn != "" {
		cluster = cluster + "." + fqdn
	}

	if clcFQDN != "" {
		cluster = cluster + "." + clcFQDN
	}

	if os.Getenv("PROMETHEUS_URL") != "" {
		prometheus = os.Getenv("PROMETHEUS_URL")
	}

	if os.Getenv("CLUSTER") != "" {
		cluster = os.Getenv("CLUSTER")
	}

	queries := []PrometheusQueries{
		{
			Description: "APISERVER",
			Query:       `avg(up{job="kube-apiserver",cluster="` + cluster + `"})`,
		},
		{
			Description: "CLUSTER",
			Query:       `capi_cluster_status_phase{phase="Provisioned", tenantcluster="` + shortCluster + `"} == 1`,
		},
		{
			Description: "FLUENTBITERRORS",
			Query:       `rate(fluentbit_output_errors_total{cluster="` + cluster + `"}[1h])) > 0`,
		},
		{
			Description: "FLUENTDERRORS",
			Query:       `avg(fluentd_output_status_num_errors{cluster="` + cluster + `"}) > 0`,
		},
		{
			Description: "GOLDPINGER",
			Query:       `avg(goldpinger_cluster_health_total{cluster="` + cluster + `"})`,
		},
		{
			Description: "KUBEDNS",
			Query:       `avg(up{job="kube-dns", cluster="` + cluster + `"})`,
		},
		{
			Description: "KUBELET",
			Query:       `clamp((count(up{job="kubelet", cluster="` + cluster + `"}) > 3),1,1)`,
		},
		{
			Description: "NETWORKOPERATOR",
			Query:       `clamp(avg(nwop_netlink_routes_fib{protocol="bgp",vrf="main",cluster="` + cluster + `"}),1,1)`,
		},
		{
			Description: "NODE",
			Query:       `min(kube_node_status_condition{condition="Ready",status="true",cluster="` + cluster + `"})`,
		},
		{
			Description: "STORAGECHECK",
			Query:       `clamp((increase(storage_check_success_total{cluster="` + cluster + `"}[1h]) > 1),1,1) OR (storage_check_failure_total{cluster="` + cluster + `"} > 0)`,
		},
		{
			Description: "PROMETHEUSAGENT",
			Query:       `avg(up{job="prometheus-agent",cluster="` + cluster + `"})`,
		},
		{
			Description: "SYSTEMPODS",
			Query:       `clamp(sum(kube_pod_status_phase{namespace=~".*-system", phase!~"Running|Succeeded",cluster="` + cluster + `"} == 0),1,1)`,
		},
	}

	fmt.Printf("\033[36mclustercheck \033[0m on %s\n", cluster)
	for _, query := range queries {
		result, err := QueryPrometheus(prometheus, query.Query, username, password)
		if err != nil {
			fmt.Println("Error query :", query.Description, err)
		} else {
			if result == "1" {
				if strings.HasPrefix(query.Description, "FLUENT") {
					fmt.Printf("%s \033[31m🔴 FAIL (0)\033[0m \n", query.Description)
				} else {
					fmt.Printf("%s \033[32m🟢 OK (1)\033[0m \n", query.Description)
				}
			} else {
				if strings.HasPrefix(query.Description, "FLUENT") {
					fmt.Printf("%s \033[32m🟢 OK (1)\033[0m \n", query.Description)
				} else {
					fmt.Printf("%s \033[31m🔴 FAIL (0)\033[0m \n", query.Description)
				}
			}
		}
	}
}
