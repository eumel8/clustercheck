# Prometheus Installation for clustercheck

## Overview

Prometheus has been successfully installed on the cluster using Flux and the kube-prometheus-stack Helm chart. This provides a complete monitoring solution with Prometheus, Grafana, and various exporters.

## Installation Details

### Components Installed

1. **Prometheus Operator**: Manages Prometheus instances
2. **Prometheus Server**: Time-series database and monitoring system
3. **Grafana**: Visualization and dashboarding
4. **Node Exporter**: Hardware and OS metrics from nodes
5. **Kube State Metrics**: Kubernetes object state metrics

### Flux Resources

**HelmRepository**: `prometheus-community`
- URL: https://prometheus-community.github.io/helm-charts
- File: `helmrepository-prometheus.yaml`

**HelmRelease**: `kube-prometheus-stack`
- Chart: kube-prometheus-stack v81.0.0
- Namespace: monitoring
- File: `helmrelease-prometheus.yaml`

## Accessing Prometheus

### Port Forwarding

```bash
# Forward Prometheus to localhost
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090

# Access Prometheus UI
open http://localhost:9090

# Test API
curl http://localhost:9090/api/v1/query?query=up
```

### Grafana Access

```bash
# Forward Grafana to localhost
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-stack-grafana 3000:80

# Access Grafana UI
open http://localhost:3000

# Default credentials
Username: admin
Password: admin
```

## Available Metrics

The kube-prometheus-stack provides the following monitoring targets:

```bash
# Check all scrape targets
curl -s --get --data-urlencode 'query=up' http://127.0.0.1:9090/api/v1/query | jq -r '.data.result[] | .metric.job' | sort -u
```

Current jobs:
- `apiserver`: Kubernetes API server metrics
- `coredns`: CoreDNS metrics
- `kube-state-metrics`: Kubernetes object state
- `kubelet`: Kubelet metrics
- `node-exporter`: Node hardware/OS metrics
- `monitoring-kube-prometheus-operator`: Prometheus operator metrics
- `monitoring-kube-prometheus-prometheus`: Prometheus self-monitoring
- `monitoring-kube-prometheus-stack-grafana`: Grafana metrics

## Using clustercheck with Prometheus

The `clustercheck` tool's default mode performs Prometheus-based health checks. However, the tool expects metrics with a `cluster` label to identify the target cluster.

### Configuration Requirements

For production use, you need to:

1. **Set Prometheus URL**:
   ```bash
   export PROMETHEUS_URL="http://127.0.0.1:9090"
   ```

2. **Set Credentials** (if required):
   ```bash
   export PROM_USER="username"
   export PROM_PASS="password"
   ```

3. **Configure Cluster Labels**: Add relabeling rules to your Prometheus configuration to add `cluster` labels to metrics.

### Example Queries

The clustercheck tool runs queries like:

```promql
# API Server check
avg(up{job="kube-apiserver",cluster="k3d-e2e"})

# Node check
min(kube_node_status_condition{condition="Ready",status="true",cluster="k3d-e2e"})

# Kubelet check
clamp((count(up{job="kubelet", cluster="k3d-e2e"}) > 3),1,1)
```

### Testing Current Setup

Check what metrics are available:

```bash
# Port-forward Prometheus
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090 &

# Check API server metrics
curl -s --get --data-urlencode 'query=up{job="apiserver"}' http://127.0.0.1:9090/api/v1/query | jq '.'

# Check node metrics
curl -s --get --data-urlencode 'query=kube_node_status_condition{condition="Ready",status="true"}' http://127.0.0.1:9090/api/v1/query | jq '.'

# Check kubelet metrics
curl -s --get --data-urlencode 'query=up{job="kubelet"}' http://127.0.0.1:9090/api/v1/query | jq '.'
```

## Verifying Installation

### Check Prometheus Pods

```bash
kubectl get pods -n monitoring
```

Expected output:
```
NAME                                                              READY   STATUS
monitoring-kube-prometheus-operator-xxx                           1/1     Running
monitoring-kube-prometheus-stack-grafana-xxx                      3/3     Running
monitoring-kube-prometheus-stack-kube-state-metrics-xxx           1/1     Running
monitoring-kube-prometheus-stack-prometheus-node-exporter-xxx     1/1     Running
prometheus-monitoring-kube-prometheus-prometheus-0                2/2     Running
```

### Check Flux Resources

```bash
# Verify HelmRelease is Ready
./clustercheck --check-flux

# Or use flux CLI
~/bin/flux get helmreleases kube-prometheus-stack
```

Expected output:
```
fluxcheck on k3d-e2e

HelmReleases:
flux-system/kube-prometheus-stack 🟢 Ready (revision: 81.0.0)
...
```

## ServiceMonitors

The kube-prometheus-stack automatically creates ServiceMonitors for:
- Kubernetes API Server
- CoreDNS
- Kubelet
- Node Exporter
- Kube State Metrics

View all ServiceMonitors:
```bash
kubectl get servicemonitors -n monitoring
```

## Adding Custom Scrape Configs

To add custom Prometheus scrape configurations:

1. Create a ConfigMap with additional scrape configs
2. Reference it in the HelmRelease values
3. Apply changes via Flux

Example in `helmrelease-prometheus.yaml`:
```yaml
prometheus:
  prometheusSpec:
    additionalScrapeConfigs:
    - job_name: 'my-custom-app'
      static_configs:
      - targets: ['my-app.default.svc:8080']
        labels:
          cluster: 'k3d-e2e'
```

## Troubleshooting

### Prometheus not scraping targets

Check ServiceMonitor selectors:
```bash
kubectl describe prometheus -n monitoring
```

### Metrics missing cluster label

Add relabeling rules or configure external labels in Prometheus:
```yaml
prometheus:
  prometheusSpec:
    externalLabels:
      cluster: 'k3d-e2e'
```

### Port forwarding issues

Kill existing port-forwards:
```bash
pkill -f "port-forward.*prometheus"
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090
```

## Integration with clustercheck

The clustercheck tool has three operation modes:

1. **Prometheus Monitoring** (default):
   ```bash
   export PROMETHEUS_URL="http://127.0.0.1:9090"
   ./clustercheck
   ```

2. **Pod Health Check**:
   ```bash
   ./clustercheck --check-pods
   ```

3. **Flux Resources Check**:
   ```bash
   ./clustercheck --check-flux
   ```

All three modes can be used to comprehensively monitor your cluster health.
