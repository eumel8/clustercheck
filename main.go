package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eumel8/clustercheck/pkg/fluxcheck"
	"github.com/eumel8/clustercheck/pkg/gatecheck"
	"github.com/eumel8/clustercheck/pkg/monitoringcheck"
	"github.com/eumel8/clustercheck/pkg/podcheck"
)

func main() {
	bitwarden := flag.Bool("bw", false, "enable Bitwarden password store")
	fqdn := flag.String("f", "", "optional FQDN of cluster targets, e.g. example.com")
	checkPods := flag.Bool("check-pods", false, "check if all pods are in Running or Succeeded state")
	checkFlux := flag.Bool("check-flux", false, "check if all Flux HelmReleases and Kustomizations are Ready")
	gateCheck := flag.Bool("gate-check", false, "comprehensive cluster health check for quality gate validation")
	namespace := flag.String("namespace", "", "namespace to check resources (empty for all namespaces)")
	debug := flag.Bool("debug", false, "enable debug output for API requests and responses")
	flag.Parse()

	if *gateCheck {
		_, err := gatecheck.GateCheck(*namespace, *bitwarden, *fqdn, *debug)
		if err != nil {
			os.Exit(1)
		}
	} else if *checkPods {
		if err := podcheck.CheckPods(*namespace, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Pod check failed: %v\n", err)
			os.Exit(1)
		}
	} else if *checkFlux {
		if err := fluxcheck.CheckFlux(*namespace, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Flux check failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		monitoringcheck.Run(*bitwarden, *fqdn, *debug)
	}
}
