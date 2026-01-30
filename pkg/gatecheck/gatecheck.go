package gatecheck

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eumel8/clustercheck/pkg/common"
	"github.com/eumel8/clustercheck/pkg/fluxcheck"
	"github.com/eumel8/clustercheck/pkg/monitoringcheck"
	"github.com/eumel8/clustercheck/pkg/podcheck"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
}

// GateCheckResult represents the overall gate check result
type GateCheckResult struct {
	TotalChecks   int
	PassedChecks  int
	FailedChecks  int
	HealthScore   float64
	CheckResults  []CheckResult
	OverallPassed bool
}

// GateCheck performs all health checks and computes an overall health score
func GateCheck(namespace string, bitwarden bool, fqdn string, debug bool) (*GateCheckResult, error) {
	result := &GateCheckResult{
		CheckResults: []CheckResult{},
	}

	// Get current context for display
	currentContext, err := common.GetCurrentContext()
	if err != nil {
		currentContext = "unknown"
	}

	fmt.Printf("\033[36m╔══════════════════════════════════════════════════╗\033[0m\n")
	fmt.Printf("\033[36m║         CLUSTER GATE CHECK - %s\033[0m\n", currentContext)
	fmt.Printf("\033[36m╚══════════════════════════════════════════════════╝\033[0m\n\n")

	// 1. Pod Health Check
	fmt.Printf("\033[1m[1/3] Pod Health Check\033[0m\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	podErr := podcheck.CheckPods(namespace, debug)
	if podErr == nil {
		result.CheckResults = append(result.CheckResults, CheckResult{
			Name:    "Pod Health",
			Passed:  true,
			Message: "All pods are in Running or Succeeded state",
		})
		result.PassedChecks++
	} else {
		result.CheckResults = append(result.CheckResults, CheckResult{
			Name:    "Pod Health",
			Passed:  false,
			Message: podErr.Error(),
		})
		result.FailedChecks++
	}
	result.TotalChecks++
	fmt.Println()

	// 2. Flux Resources Check
	fmt.Printf("\033[1m[2/3] Flux Resources Check\033[0m\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fluxErr := fluxcheck.CheckFlux(namespace, debug)
	if fluxErr == nil {
		result.CheckResults = append(result.CheckResults, CheckResult{
			Name:    "Flux Resources",
			Passed:  true,
			Message: "All HelmReleases and Kustomizations are Ready",
		})
		result.PassedChecks++
	} else {
		result.CheckResults = append(result.CheckResults, CheckResult{
			Name:    "Flux Resources",
			Passed:  false,
			Message: fluxErr.Error(),
		})
		result.FailedChecks++
	}
	result.TotalChecks++
	fmt.Println()

	// 3. Prometheus Monitoring Check
	fmt.Printf("\033[1m[3/3] Prometheus Monitoring Check\033[0m\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	monitoringChecks, monitoringPassed := runPrometheusChecks(bitwarden, fqdn, debug)

	for _, check := range monitoringChecks {
		result.CheckResults = append(result.CheckResults, check)
		result.TotalChecks++
		if check.Passed {
			result.PassedChecks++
		} else {
			result.FailedChecks++
		}
	}

	if monitoringPassed {
		fmt.Printf("\n\033[32m✓ All Prometheus checks passed\033[0m\n")
	} else {
		fmt.Printf("\n\033[31m✗ Some Prometheus checks failed\033[0m\n")
	}
	fmt.Println()

	// Calculate health score
	if result.TotalChecks > 0 {
		result.HealthScore = (float64(result.PassedChecks) / float64(result.TotalChecks)) * 100
	}
	result.OverallPassed = result.HealthScore >= 80.0

	// Print Summary
	fmt.Printf("\033[36m╔══════════════════════════════════════════════════╗\033[0m\n")
	fmt.Printf("\033[36m║              GATE CHECK SUMMARY                  ║\033[0m\n")
	fmt.Printf("\033[36m╚══════════════════════════════════════════════════╝\033[0m\n\n")

	if result.OverallPassed {
		fmt.Printf("\033[1;32m✓ CLUSTER HEALTH: PASSED\033[0m\n")
	} else {
		fmt.Printf("\033[1;31m✗ CLUSTER HEALTH: FAILED\033[0m\n")
	}

	fmt.Printf("\n\033[1mHealth Score: %.1f%% (%d of %d checks passed)\033[0m\n\n",
		result.HealthScore, result.PassedChecks, result.TotalChecks)

	// Detailed Results
	fmt.Println("Detailed Results:")
	fmt.Println("─────────────────────────────────────────────────")
	for _, check := range result.CheckResults {
		if check.Passed {
			fmt.Printf("✓ \033[32m%-30s\033[0m PASS\n", check.Name)
		} else {
			fmt.Printf("✗ \033[31m%-30s\033[0m FAIL - %s\n", check.Name, check.Message)
		}
	}
	fmt.Println()

	// Quality Gate Decision
	fmt.Println("Quality Gate Decision:")
	fmt.Println("─────────────────────────────────────────────────")
	if result.HealthScore >= 90 {
		fmt.Printf("\033[1;32m🟢 EXCELLENT - Ready for production\033[0m\n")
	} else if result.HealthScore >= 80 {
		fmt.Printf("\033[1;32m🟡 GOOD - Acceptable for go-live\033[0m\n")
	} else if result.HealthScore >= 60 {
		fmt.Printf("\033[1;33m🟠 FAIR - Review failures before go-live\033[0m\n")
	} else {
		fmt.Printf("\033[1;31m🔴 POOR - Not ready for production\033[0m\n")
	}
	fmt.Println()

	if !result.OverallPassed {
		return result, fmt.Errorf("cluster health check failed with score %.1f%%", result.HealthScore)
	}

	return result, nil
}

// runPrometheusChecks executes Prometheus monitoring checks and returns results
func runPrometheusChecks(bitwarden bool, fqdn string, debug bool) ([]CheckResult, bool) {
	results := []CheckResult{}

	// static Prometheus API endpoint
	prometheus := "https://127.0.0.1:9090"
	username := os.Getenv("PROM_USER")
	password := os.Getenv("PROM_PASS")
	clcBW := os.Getenv("CLUSTERCHECK_BW")
	clcFQDN := os.Getenv("CLUSTERCHECK_FQDN")

	if bitwarden == true || clcBW != "" {
		itemName := "Prometheus Agent RemoteWrite"
		jsonData, err := monitoringcheck.GetBitwardenItemJSON(itemName)
		if err != nil {
			results = append(results, CheckResult{
				Name:    "Prometheus Authentication",
				Passed:  false,
				Message: fmt.Sprintf("Failed to get Bitwarden credentials: %v", err),
			})
			return results, false
		}

		var item monitoringcheck.BitwardenItem
		err = json.Unmarshal(jsonData, &item)
		if err != nil {
			results = append(results, CheckResult{
				Name:    "Prometheus Authentication",
				Passed:  false,
				Message: fmt.Sprintf("Failed to parse Bitwarden JSON: %v", err),
			})
			return results, false
		}

		username = item.Login.Username
		password = item.Login.Password
	}

	cluster, err := common.GetCurrentContext()
	if err != nil {
		cluster = "unknown"
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

	queries := []monitoringcheck.PrometheusQueries{
		{
			Description: "APISERVER",
			Query:       `avg(up{job="kube-apiserver",cluster="` + cluster + `"})`,
		},
                {
                        Description: "CLUSTER",
                        Query:       `capi_cluster_status_phase{phase="Provisioned", tenantcluster="` + shortCluster + `"} == 1`,
                },
                {
                        Description: "FLUENTBIT_OK",
                        Query: `count(max(fluentbit_output_errors_total{cluster="` + cluster + `"}) + 1)`,
                },
                {
                        Description: "FLUENTD_OK",
                        Query: `count(max(fluentd_output_status_num_errors{cluster="` + cluster + `"}) + 1)`,
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

	allPassed := true
	for _, query := range queries {
		result, err := monitoringcheck.QueryPrometheus(prometheus, query.Query, username, password, debug)
		if err != nil {
			results = append(results, CheckResult{
				Name:    query.Description,
				Passed:  false,
				Message: fmt.Sprintf("Query error: %v", err),
			})
			allPassed = false
			fmt.Printf("  %s \033[31m✗ ERROR\033[0m - %v\n", query.Description, err)
		} else {
			if result == "1" {
				results = append(results, CheckResult{
					Name:    query.Description,
					Passed:  true,
					Message: "Healthy",
				})
				fmt.Printf("  %s \033[32m✓ OK\033[0m\n", query.Description)
			} else {
				results = append(results, CheckResult{
					Name:    query.Description,
					Passed:  false,
					Message: fmt.Sprintf("Value: %s (expected: 1)", result),
				})
				allPassed = false
				fmt.Printf("  %s \033[31m✗ FAIL\033[0m - Value: %s\n", query.Description, result)
			}
		}
	}

	return results, allPassed
}
