package sentry

import (
	"fmt"
	"slices"
)

type Preset struct {
	Name    string
	Aliases []string
	Value   []byte
}

func GetPreset(name string) (*Preset, error) {
	for _, preset := range DefaultPresets() {
		if preset.Name == name || slices.Contains(preset.Aliases, name) {
			return &preset, nil
		}
	}

	return nil, fmt.Errorf("preset %s not found", name)
}

func DefaultPresets() []Preset {
	return []Preset{
		{
			Name:    "ethpandaops",
			Aliases: []string{"ethpandaops-production"},
			Value: []byte(`
preset: ethpandaops-production
ethereum:
  beaconSubscriptions:
  - block
  - blob_sidecar
  - chain_reorg
  - finalized_checkpoint
  - head
outputs:
- name: ethpandaops
  type: xatu
  config:
    address: xatu.primary.production.platform.ethpandaops.io:443
    tls: true
    retry:
      enabled: true
      scalar: 1s
      maxAttempts: 5
    maxExportBatchSize: 64
    batchTimeout: 10s
    workers: 10
    maxQueueSize: 20000
    headers:
      Authorization: "Bearer $MUST_BE_SET_BY_USER"
    eventFilter:
      eventNames:
      - BEACON_API_ETH_V2_BEACON_BLOCK_V2
      - BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR
      - BEACON_API_ETH_V1_EVENTS_BLOCK_V2
      - BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2
      - BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2
      - BEACON_API_ETH_V1_EVENTS_HEAD_V2
`,
			),
		},
	}
}
