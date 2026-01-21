# Gate Check - Comprehensive Cluster Health Validation

## Overview

The `--gate-check` feature provides a comprehensive cluster health validation suitable for quality gate decisions before production deployments. It combines all three existing check modes (pod health, Flux resources, and Prometheus monitoring) into a single comprehensive assessment with an aggregated health score.

## What It Does

The gate check performs three types of validation:

1. **Pod Health Check**: Verifies all pods are in Running or Succeeded state
2. **Flux Resources Check**: Ensures all HelmReleases and Kustomizations are Ready
3. **Prometheus Monitoring Check**: Validates key cluster metrics via Prometheus

It then computes an overall health score as a percentage and provides a quality gate decision.

## Usage

### Basic Usage

```bash
./clustercheck --gate-check
```

### With Namespace Filter

```bash
# Check resources only in specific namespace
./clustercheck --gate-check --namespace monitoring
```

### With Prometheus Authentication

```bash
# Using Bitwarden
./clustercheck --gate-check --bw

# Using environment variables
export PROM_USER="username"
export PROM_PASS="password"
./clustercheck --gate-check
```

### With Custom Cluster FQDN

```bash
./clustercheck --gate-check --f example.com
```

### Environment Variables

```bash
# Set Prometheus URL (default: https://127.0.0.1:9090)
export PROMETHEUS_URL="http://127.0.0.1:9090"

# Set cluster name override
export CLUSTER="my-cluster"

# Set cluster FQDN
export CLUSTERCHECK_FQDN="prod.example.com"

# Run gate check
./clustercheck --gate-check
```

## Output Example

```
╔══════════════════════════════════════════════════╗
║         CLUSTER GATE CHECK - k3d-e2e             ║
╚══════════════════════════════════════════════════╝

[1/3] Pod Health Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
podcheck on k3d-e2e
default/app-pod-1 🟢 Running
default/app-pod-2 🟢 Running
...
Summary: 20/20 pods in Running or Succeeded state

[2/3] Flux Resources Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
fluxcheck on k3d-e2e

HelmReleases:
flux-system/my-app 🟢 Ready (revision: 1.0.0)

Kustomizations:
flux-system/config 🟢 Ready (revision: main@sha1:abc123)

Summary: 2/2 resources Ready

[3/3] Prometheus Monitoring Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  API Server ✓ OK
  Kubelet ✓ OK
  Node Status ✓ OK
  System Pods ✓ OK

✓ All Prometheus checks passed

╔══════════════════════════════════════════════════╗
║              GATE CHECK SUMMARY                  ║
╚══════════════════════════════════════════════════╝

✓ CLUSTER HEALTH: PASSED

Health Score: 100.0% (6 of 6 checks passed)

Detailed Results:
─────────────────────────────────────────────────
✓ Pod Health                     PASS
✓ Flux Resources                 PASS
✓ API Server                     PASS
✓ Kubelet                        PASS
✓ Node Status                    PASS
✓ System Pods                    PASS

Quality Gate Decision:
─────────────────────────────────────────────────
🟢 EXCELLENT - Ready for production
```

## Health Score Thresholds

The gate check uses the following health score thresholds:

| Score Range | Status | Decision | Exit Code |
|-------------|--------|----------|-----------|
| 90-100% | 🟢 EXCELLENT | Ready for production | 0 |
| 80-89% | 🟡 GOOD | Acceptable for go-live | 0 |
| 60-79% | 🟠 FAIR | Review failures before go-live | 1 |
| 0-59% | 🔴 POOR | Not ready for production | 1 |

**Pass Threshold**: 80% or higher

## Exit Codes

- **0**: Health check passed (score >= 80%)
- **1**: Health check failed (score < 80%)

This makes it suitable for CI/CD pipeline integration:

```bash
#!/bin/bash
if ./clustercheck --gate-check; then
    echo "✓ Cluster passed health check - deploying to production"
    # Deploy to production
else
    echo "✗ Cluster failed health check - blocking deployment"
    exit 1
fi
```

## Prometheus Checks

The gate check performs the following Prometheus validations:

1. **API Server**: Verifies Kubernetes API server is up
   - Query: `avg(up{job="kube-apiserver",cluster="<cluster>"})`

2. **Kubelet**: Ensures kubelet is running on nodes
   - Query: `clamp((count(up{job="kubelet",cluster="<cluster>"}) > 3),1,1)`

3. **Node Status**: Checks nodes are in Ready state
   - Query: `min(kube_node_status_condition{condition="Ready",status="true",cluster="<cluster>"})`

4. **System Pods**: Validates system pods are running
   - Query: `clamp(sum(kube_pod_status_phase{namespace=~".*-system",phase!~"Running|Succeeded",cluster="<cluster>"} == 0),1,1)`

**Note**: These queries expect metrics to have a `cluster` label. If your Prometheus setup doesn't include cluster labels, you may need to configure external labels or relabeling rules.

## CI/CD Integration

### GitHub Actions

```yaml
name: Production Deployment
on:
  push:
    branches: [main]

jobs:
  health-check:
    runs-on: ubuntu-latest
    steps:
      - name: Configure kubectl
        uses: azure/k8s-set-context@v1
        with:
          kubeconfig: ${{ secrets.KUBE_CONFIG }}

      - name: Port-forward Prometheus
        run: |
          kubectl port-forward -n monitoring svc/prometheus 9090:9090 &
          sleep 5

      - name: Run Gate Check
        run: |
          export PROMETHEUS_URL="http://127.0.0.1:9090"
          ./clustercheck --gate-check

      - name: Deploy if healthy
        if: success()
        run: |
          # Your deployment commands
          echo "Deploying to production"
```

### GitLab CI

```yaml
gate_check:
  stage: test
  script:
    - export PROMETHEUS_URL="http://prometheus.monitoring:9090"
    - ./clustercheck --gate-check
  only:
    - main
  allow_failure: false
```

### Jenkins

```groovy
stage('Gate Check') {
    steps {
        script {
            sh '''
                export PROMETHEUS_URL="http://127.0.0.1:9090"
                ./clustercheck --gate-check
            '''
        }
    }
}
```

## Troubleshooting

### All Prometheus Checks Failing

**Symptom**: All Prometheus checks return value "0"

**Cause**: Metrics don't have `cluster` label or Prometheus is not accessible

**Solution**:
1. Verify Prometheus is accessible:
   ```bash
   curl http://127.0.0.1:9090/api/v1/query?query=up
   ```

2. Check if metrics have cluster labels:
   ```bash
   curl 'http://127.0.0.1:9090/api/v1/query?query=up{job="kubelet"}'
   ```

3. Add cluster labels via Prometheus external labels:
   ```yaml
   prometheus:
     prometheusSpec:
       externalLabels:
         cluster: 'my-cluster'
   ```

### Pod Check Fails in Test Namespace

**Symptom**: Pod check shows failures for test/development pods

**Solution**: Use namespace filter to focus on production namespaces:
```bash
./clustercheck --gate-check --namespace production
```

### Flux Resources Not Found

**Symptom**: "No Flux resources found"

**Cause**: Flux is not installed or resources are in different namespace

**Solution**:
1. Check Flux installation:
   ```bash
   flux check
   ```

2. List Flux resources:
   ```bash
   flux get all -A
   ```

## Comparison with Individual Checks

| Feature | --check-pods | --check-flux | Default (Prometheus) | --gate-check |
|---------|--------------|--------------|----------------------|--------------|
| Pod Health | ✓ | - | - | ✓ |
| Flux Resources | - | ✓ | - | ✓ |
| Prometheus Metrics | - | - | ✓ | ✓ |
| Health Score | - | - | - | ✓ |
| Quality Gate Decision | - | - | - | ✓ |
| CI/CD Ready | Partial | Partial | Partial | ✓ |
| Exit Code on Failure | ✓ | ✓ | - | ✓ |

## Best Practices

1. **Run Regularly**: Include gate checks in your CI/CD pipeline for every production deployment

2. **Set Appropriate Thresholds**: The default 80% threshold works for most cases, but adjust based on your requirements

3. **Monitor Trends**: Track health scores over time to identify degradation patterns

4. **Document Exceptions**: If certain checks consistently fail but are acceptable, document why

5. **Combine with Other Checks**: Use gate check alongside:
   - Security scans
   - Load tests
   - Smoke tests
   - Integration tests

6. **Test in Staging First**: Always run gate checks in staging before production

## Examples

### Pre-deployment Validation

```bash
#!/bin/bash
set -e

echo "Running pre-deployment health checks..."

# Port-forward Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090 > /dev/null 2>&1 &
PF_PID=$!
sleep 5

# Run gate check
export PROMETHEUS_URL="http://127.0.0.1:9090"
if ./clustercheck --gate-check; then
    echo "✓ Cluster health check passed"

    # Deploy application
    kubectl apply -f manifests/

    # Wait for rollout
    kubectl rollout status deployment/my-app

    echo "✓ Deployment successful"
else
    echo "✗ Cluster health check failed - aborting deployment"
    kill $PF_PID
    exit 1
fi

# Cleanup
kill $PF_PID
```

### Production Readiness Report

```bash
#!/bin/bash
# Generate production readiness report

echo "Production Readiness Report - $(date)"
echo "======================================"
echo ""

export PROMETHEUS_URL="http://127.0.0.1:9090"
./clustercheck --gate-check > /tmp/gate-check-report.txt 2>&1

cat /tmp/gate-check-report.txt

# Extract health score
HEALTH_SCORE=$(grep "Health Score:" /tmp/gate-check-report.txt | awk '{print $3}')
echo ""
echo "Overall Health Score: $HEALTH_SCORE"

if [[ "$HEALTH_SCORE" > "90" ]]; then
    echo "Status: APPROVED FOR PRODUCTION"
else
    echo "Status: REQUIRES REVIEW"
fi
```

## Related Documentation

- [README.md](README.md) - Main project documentation
- [PROMETHEUS-SETUP.md](flux-examples/PROMETHEUS-SETUP.md) - Prometheus configuration
- [flux-examples/README.md](flux-examples/README.md) - Flux examples and setup

## Support

For issues or questions:
- Check logs with verbose output
- Verify all prerequisites are met
- Review troubleshooting section
- Test individual check modes first
