# clustercheck
![Coverage](https://img.shields.io/badge/Coverage-20.9%25-red)

## intro

clustercheck is a comprehensive Kubernetes cluster health validation tool with multiple operational modes.

### Features

- **Prometheus Monitoring** (default): Query Prometheus for cluster health metrics
- **Pod Health Check** (`--check-pods`): Verify all pods are Running or Succeeded
- **Flux Resources Check** (`--check-flux`): Ensure HelmReleases and Kustomizations are Ready
- **Gate Check** (`--gate-check`): Comprehensive health validation with scoring for quality gates

### Requirements

* Configured `kubeconfig` - clustercheck will use the current context
* For Prometheus checks: access to Prometheus API endpoint
* For Flux checks: Flux CD installed on the cluster


## download

We provide binaries for various platform. Go to the [release page](https://github.com/eumel8/clustercheck/releases).


## usage

### Operational Modes

#### 1. Prometheus Monitoring (Default)

Query Prometheus for cluster health metrics:

```bash
./clustercheck
```

Output:
```
APISERVER 🟢 OK (1)
CLUSTER 🟢 OK (1)
FLUENTBITERRORS 🔴 FAIL (0)
FLUENTDERRORS 🟢 OK (1)
GOLDPINGER 🔴 FAIL (0)
KUBEDNS 🟢 OK (1)
KUBELET 🟢 OK (1)
NETWORKOPERATOR 🟢 OK (1)
NODE 🟢 OK (1)
STORAGECHECK 🟢 OK (1)
PROMETHEUSAGENT 🔴 FAIL (0)
SYSTEMPODS 🟢 OK (1)
```

#### 2. Pod Health Check

Verify all pods are in Running or Succeeded state:

```bash
# Check all pods in all namespaces
./clustercheck --check-pods

# Check pods in specific namespace
./clustercheck --check-pods --namespace production
```

Output:
```
podcheck on k3d-e2e
default/app-1 🟢 Running
default/app-2 🟢 Running
...
Summary: 20/20 pods in Running or Succeeded state
```

#### 3. Flux Resources Check

Ensure all HelmReleases and Kustomizations are Ready:

```bash
# Check all Flux resources
./clustercheck --check-flux

# Check Flux resources in specific namespace
./clustercheck --check-flux --namespace flux-system
```

Output:
```
fluxcheck on k3d-e2e

HelmReleases:
flux-system/my-app 🟢 Ready (revision: 1.0.0)

Kustomizations:
flux-system/config 🟢 Ready (revision: main@sha1:abc123)

Summary: 2/2 resources Ready
```

#### 4. Gate Check (Comprehensive)

Comprehensive cluster health validation with scoring for quality gate decisions:

```bash
./clustercheck --gate-check
```

Output:
```
╔══════════════════════════════════════════════════╗
║         CLUSTER GATE CHECK - k3d-e2e             ║
╚══════════════════════════════════════════════════╝

[1/3] Pod Health Check
...

[2/3] Flux Resources Check
...

[3/3] Prometheus Monitoring Check
...

╔══════════════════════════════════════════════════╗
║              GATE CHECK SUMMARY                  ║
╚══════════════════════════════════════════════════╝

✓ CLUSTER HEALTH: PASSED

Health Score: 100.0% (6 of 6 checks passed)

Quality Gate Decision:
─────────────────────────────────────────────────
🟢 EXCELLENT - Ready for production
```

Exit codes:
- `0`: Health check passed (score >= 80%)
- `1`: Health check failed (score < 80%)

For detailed gate check documentation, see [GATE-CHECK.md](GATE-CHECK.md).

### Command-Line Flags

```bash
Usage of ./clustercheck:
  -bw
        enable Bitwarden password store
  -check-flux
        check if all Flux HelmReleases and Kustomizations are Ready
  -check-pods
        check if all pods are in Running or Succeeded state
  -f string
        optional FQDN of cluster targets, e.g. example.com
  -gate-check
        comprehensive cluster health check for quality gate validation
  -namespace string
        namespace to check resources (empty for all namespaces)
```

## tips & tricks

### remove quarantine flag on Mac

```
xattr -d com.apple.quarantine $HOME/bin/clustercheck
```

### overwriting cluster name

```
export CLUSTER="my-cluster"
```

### overwrite prometheus url

```
export PROMETHEUS_URL="https://my-prometheus.instance"
```

### set basic auth credentials to access Prometheus

```
export PROM_USER="user"
export PROM_PASS="pass"
```

## Bitwarden feature

Start the programm with `-bw` or set env var

```
export CLUSTERCHECK_BW=1
```

In this version the programm expect an item on a Bitwarden service containing username/password for HTTP Basic Auth on
Prometheus API

```
bw get item "Prometheus Agent RemoteWrite
```


### set FQDN

If your cluster has a FQDN which is specific to set start the programm with `-f` together with the FQDN or set env var

```
CLUSTERCHECK_FQDN=example.com
```

### Prometheus TLS connection

we skip SSL verification and allow insecure connection by default, take care.

### Proxy Settings

we respect env vars like `http_proxy` or `https_proxy` for Prometheus endpoint connection from your computer.

