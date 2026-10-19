package promcheck

import (
	"strings"
	"testing"
)

func TestDefaultQueries(t *testing.T) {
	qs := DefaultQueries("c1.example", "c1")
	if len(qs) != 12 {
		t.Fatalf("expected 12 queries, got %d", len(qs))
	}
	for _, q := range qs {
		if q.Description == "" || q.Query == "" {
			t.Errorf("empty query entry: %+v", q)
		}
	}
	nwop := Select(qs, NetworkOperator)[0].Query
	if !strings.Contains(nwop, "nwop_vsr_routes_fib") || !strings.Contains(nwop, "nwop_netlink_routes_fib") {
		t.Errorf("NETWORKOPERATOR must cover FRR and VSR: %s", nwop)
	}
	if !strings.Contains(Select(qs, Cluster)[0].Query, `tenantcluster="c1"`) {
		t.Error("CLUSTER must use the short cluster name")
	}
}

func TestSelect(t *testing.T) {
	got := Select(DefaultQueries("c", "c"), "node", APIServer)
	if len(got) != 2 || got[0].Description != APIServer || got[1].Description != Node {
		t.Errorf("unexpected selection: %+v", got)
	}
}
