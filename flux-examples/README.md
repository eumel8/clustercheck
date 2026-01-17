# Flux Installation Summary

## Installation Details

Flux v2.4.0 has been successfully installed on the cluster with all controllers running:
- helm-controller
- kustomize-controller
- notification-controller
- source-controller

## Example Resources Created

### 1. HelmRepositories
Created two HelmRepository resources:

**Bitnami Repository** (`helmrepository.yaml`):
- URL: https://charts.bitnami.com/bitnami
- Namespace: flux-system
- Status: Ready

**Grafana Repository** (`helmrepository-grafana.yaml`):
- URL: https://grafana.github.io/helm-charts
- Namespace: flux-system
- Status: Ready

### 2. HelmRelease
**Grafana Test Release** (`helmrelease.yaml`):
- Chart: grafana v10.5.8
- Source: grafana HelmRepository
- Target Namespace: default
- Status: Successfully deployed
- Pod: Running

### 3. GitRepository
**Podinfo Repository** (`gitrepository.yaml`):
- URL: https://github.com/stefanprodan/podinfo
- Branch: master
- Namespace: flux-system
- Status: Ready

### 4. Kustomization
**Podinfo Kustomization** (`kustomization.yaml`):
- Source: podinfo GitRepository
- Path: ./kustomize
- Target Namespace: default
- Status: Applied successfully
- Pods: 2 replicas running

## Usage

### Check Flux status:
```bash
~/bin/flux check
```

### View all sources:
```bash
~/bin/flux get sources all
```

### View HelmReleases:
```bash
~/bin/flux get helmreleases
```

### View Kustomizations:
```bash
~/bin/flux get kustomizations -A
```

### Apply resources:
```bash
kubectl apply -f flux-examples/
```

## Deployed Applications

1. **Grafana** (via HelmRelease): Accessible in the `default` namespace
2. **Podinfo** (via Kustomization): 2 replicas running in the `default` namespace

## Verification Commands

### Using Flux CLI:
```bash
# Check all Flux resources
~/bin/flux get all -A

# Check pods deployed by Flux
kubectl get pods -n default | grep -E '(grafana|podinfo)'

# Get HelmRelease details
~/bin/flux get helmreleases grafana-test

# Get Kustomization details
~/bin/flux get kustomizations podinfo -n flux-system
```

### Using clustercheck tool:
```bash
# Check all Flux HelmReleases and Kustomizations across all namespaces
./clustercheck --check-flux

# Check Flux resources in a specific namespace
./clustercheck --check-flux --namespace flux-system

# Check all pods in the cluster
./clustercheck --check-pods

# Run Prometheus-based monitoring checks (default)
./clustercheck
```

The `--check-flux` flag verifies that all HelmReleases and Kustomizations have a Ready status of "True". It provides color-coded output showing the status of each resource along with revision information.
