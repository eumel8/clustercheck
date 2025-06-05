# clustercheck
![Coverage](https://img.shields.io/badge/Coverage-53.8%25-yellow)

## intro

clustercheck is a little helper to check the current health of a Kubernetes cluster.

It checks pre-defined KPI while query Prometheus backend. For that some requirements are needed:

* access to Prometheus from command line, between https_proxy variable
* configured `kubeconfig`, clustercheck will use the current context


## download

We provide binaries for various platform. Go to the [release page](https://github.com/eumel8/clustercheck/releases).


## usage:

```
% clustercheck                              
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
