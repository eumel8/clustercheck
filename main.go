package main

// Kubernetes Healthy Cluster Speed Check
// This script checks the health of a Kubernetes cluster by querying Prometheus for various metrics.
// It uses the Bitwarden CLI to retrieve Prometheus credentials and performs checks on various components.

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"
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

// get current kubernetes context
func getCurrentContext() (string, error) {
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// Query Prometheus
func queryPrometheus(prometheus string, query string, username string, password string) (string, error) {
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

func main() {
	bitwarden := flag.Bool("bw", false, "enable Bitwarden password store")
	fqdn := flag.String("f", "", "optional FQDN of cluster targets, e.g. example.com")
	flag.Parse()


	// static Prometheus API endpoint
	prometheus := "https://127.0.0.1:9090"
	username := os.Getenv("PROM_USER")
	password := os.Getenv("PROM_PASS")
	clcBW := os.Getenv("CLUSTERCHECK_BW")
	clcFQDN := os.Getenv("CLUSTERCHECK_FQDN")

	if *bitwarden == true || clcBW != "" {
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

	cluster, err := getCurrentContext()
	if err != nil {
		fmt.Printf("Failed to get current kube context: %v\n", err)
	}

	if *fqdn != "" {
		cluster =  cluster + "." + *fqdn
	}

	if clcFQDN != "" {
		cluster =  cluster + "." + clcFQDN
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
			Query:       `avg(up{application="apiserver",cluster="` + cluster + `"})`,
		},
		{
			Description: "CLUSTER",
			Query:       `capi_cluster_status_phase{phase="Provisioned", tenantcluster="` + cluster + `"} == 1`,
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
			Query:       `clamp((storage_check_success_total{cluster="` + cluster + `"} > 0 AND storage_check_failure_total{cluster="` + cluster + `"} == 0),1,1)`,
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
		result, err := queryPrometheus(prometheus, query.Query, username, password)
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
