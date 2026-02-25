package main

import "strings"

// prefixRoute maps a table name prefix to a Go package subdirectory.
// Only tables matching a known prefix are generated; everything else
// (materialized views, legacy tables, observoor, etc.) is skipped.
//
// Adding a new domain = one line here.
var prefixRoutes = []struct {
	Prefix  string
	Package string
}{
	{"beacon_api_eth_", "beacon"},
	{"canonical_beacon_", "canonical"},
	{"consensus_engine_", "execution"},
	{"execution_block_metrics", "execution"},
	{"execution_engine_", "execution"},
	{"execution_state_size", "execution"},
	{"mempool_transaction", "execution"},
	{"libp2p_", "libp2p"},
	{"mev_", "mev"},
	{"node_record_", "node"},
}

// resolvePackage returns the Go package for a logical table name.
// Returns ("", false) if no prefix matches — the caller should skip the table.
func resolvePackage(table string) (string, bool) {
	for _, r := range prefixRoutes {
		if strings.HasPrefix(table, r.Prefix) {
			return r.Package, true
		}
	}

	return "", false
}
