package flattener

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// All returns all registered routes used by consumoor.
func All() []Route {
	routes := make([]Route, 0, 96)
	routes = append(routes, beaconRoutes()...)
	routes = append(routes, executionRoutes()...)
	routes = append(routes, mevAndNodeRoutes()...)
	routes = append(routes, libp2pRoutes()...)

	return routes
}

func peerConvergenceMutator(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error) {
	peerID := firstNonEmpty(row, "remote_peer", "peer_id")
	if peerID == "" {
		return nil, nil
	}

	row["peer_id"] = peerID
	row["unique_key"] = hashKey(peerID + asString(row["meta_network_name"]))
	row["updated_date_time"] = time.Now().Unix()

	return []map[string]any{row}, nil
}

func syncCommitteeMutator(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error) {
	setAlias(row, "epoch", "epoch_number")

	aggsRaw, ok := row["sync_committee_validator_aggregates"]
	if !ok {
		return []map[string]any{row}, nil
	}

	aggs, ok := aggsRaw.([]any)
	if !ok {
		return []map[string]any{row}, nil
	}

	validatorAggregates := make([][]uint64, 0, len(aggs))

	for _, agg := range aggs {
		aggMap, ok := agg.(map[string]any)
		if !ok {
			continue
		}

		validators, _ := uint64Slice(aggMap["validators"])
		validatorAggregates = append(validatorAggregates, validators)
	}

	row["validator_aggregates"] = validatorAggregates

	return []map[string]any{row}, nil
}

func syncAggregateMutator(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error) {
	setAlias(row, "slot", "block_slot_number")
	setAlias(row, "epoch", "block_epoch_number")

	if validators, ok := uint64Slice(row["validators_participated"]); ok {
		row["validators_participated"] = validators
	}

	if validators, ok := uint64Slice(row["validators_missed"]); ok {
		row["validators_missed"] = validators
	}

	return []map[string]any{row}, nil
}

func uint64Slice(value any) ([]uint64, bool) {
	items, ok := value.([]any)
	if !ok {
		if values, okk := value.([]uint64); okk {
			return values, true
		}

		return nil, false
	}

	out := make([]uint64, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case uint64:
			out = append(out, v)
		case int64:
			if v >= 0 {
				out = append(out, uint64(v))
			}
		case float64:
			if v >= 0 {
				out = append(out, uint64(v))
			}
		case string:
			parsed, err := parseUint(v)
			if err == nil {
				out = append(out, parsed)
			}
		}
	}

	return out, true
}

func parseUint(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty")
	}

	return strconv.ParseUint(value, 10, 64)
}
