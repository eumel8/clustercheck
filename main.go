package main

import (
	"flag"

	"github.com/eumel8/clustercheck/pkg/monitoringcheck"
)

func main() {
	bitwarden := flag.Bool("bw", false, "enable Bitwarden password store")
	fqdn := flag.String("f", "", "optional FQDN of cluster targets, e.g. example.com")
	flag.Parse()

	monitoringcheck.Run(*bitwarden, *fqdn)
}
