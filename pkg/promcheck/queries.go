// Package promcheck provides the shared Prometheus health queries and a small
// query client used by clustercheck and other applications.
//
// It only depends on the Go standard library.
package promcheck

import "strings"

// Query is a single named Prometheus health query. A healthy result is "1".
type Query struct {
	Description string `json:"description"`
	Query       string `json:"query"`
}

// Names of the default queries.
const (
	APIServer       = "APISERVER"
	Cluster         = "CLUSTER"
	FluentbitOK     = "FLUENTBIT_OK"
	FluentdOK       = "FLUENTD_OK"
	Goldpinger      = "GOLDPINGER"
	KubeDNS         = "KUBEDNS"
	Kubelet         = "KUBELET"
	NetworkOperator = "NETWORKOPERATOR"
	Node            = "NODE"
	StorageCheck    = "STORAGECHECK"
	PrometheusAgent = "PROMETHEUSAGENT"
	SystemPods      = "SYSTEMPODS"
)

// DefaultQueries returns the full set of cluster health queries.
//
// cluster is the value of the "cluster" label (usually the FQDN),
// shortCluster the value of the "tenantcluster" label (short context name).
func DefaultQueries(cluster, shortCluster string) []Query {
	c := cluster
	return []Query{
		{APIServer, `avg(up{job="kube-apiserver",cluster="` + c + `"})`},
		{Cluster, `capi_cluster_status_phase{phase="Provisioned", tenantcluster="` + shortCluster + `"} == 1`},
		{FluentbitOK, `count(max(fluentbit_output_errors_total{cluster="` + c + `"}) + 1)`},
		{FluentdOK, `count(max(fluentd_output_status_num_errors{cluster="` + c + `"}) + 1)`},
		{Goldpinger, `avg(goldpinger_cluster_health_total{cluster="` + c + `"})`},
		{KubeDNS, `avg(up{job="kube-dns", cluster="` + c + `"})`},
		{Kubelet, `clamp((count(up{job="kubelet", cluster="` + c + `"}) > 3),1,1)`},
		{NetworkOperator, `clamp(avg(nwop_netlink_routes_fib{protocol="bgp",vrf="main",cluster="` + c + `"}),1,1) or clamp(avg(nwop_vsr_routes_fib{protocol="bgp",vrf="main",cluster="` + c + `"}),1,1)`},
		{Node, `min(kube_node_status_condition{condition="Ready",status="true",cluster="` + c + `"})`},
		{StorageCheck, `clamp((increase(storage_check_success_total{cluster="` + c + `"}[1h]) >= 1),1,1)`},
		{PrometheusAgent, `avg(up{job="prometheus-agent",cluster="` + c + `"})`},
		{SystemPods, `clamp(sum(kube_pod_status_phase{namespace=~".*-system", phase!~"Running|Succeeded",cluster="` + c + `"} == 0),1,1)`},
	}
}

// Select returns the queries whose Description matches one of names
// (case-insensitive), in the order of the input slice.
func Select(queries []Query, names ...string) []Query {
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[strings.ToUpper(n)] = struct{}{}
	}
	var out []Query
	for _, q := range queries {
		if _, ok := want[strings.ToUpper(q.Description)]; ok {
			out = append(out, q)
		}
	}
	return out
}
