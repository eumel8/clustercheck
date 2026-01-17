package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eumel8/clustercheck/pkg/monitoringcheck"
)

func main() {
	bitwarden := flag.Bool("bw", false, "enable Bitwarden password store")
	fqdn := flag.String("f", "", "optional FQDN of cluster targets, e.g. example.com")
	checkPods := flag.Bool("check-pods", false, "check if all pods are in Running or Succeeded state")
	checkFlux := flag.Bool("check-flux", false, "check if all Flux HelmReleases and Kustomizations are Ready")
	namespace := flag.String("namespace", "", "namespace to check resources (empty for all namespaces)")
	flag.Parse()

	if *checkPods {
		if err := monitoringcheck.CheckPods(*namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Pod check failed: %v\n", err)
			os.Exit(1)
		}
	} else if *checkFlux {
		if err := monitoringcheck.CheckFlux(*namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Flux check failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		monitoringcheck.Run(*bitwarden, *fqdn)
	}
}
