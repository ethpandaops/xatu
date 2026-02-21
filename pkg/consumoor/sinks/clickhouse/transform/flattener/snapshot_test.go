package flattener_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	libp2p "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	gossipsub "github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	mevrelay "github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	noderecord "github.com/ethpandaops/xatu/pkg/proto/noderecord"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// testFixture pairs a table name with a populated event and column assertions.
type testFixture struct {
	table  string
	event  *xatu.DecoratedEvent
	checks map[string]any
	// rows overrides expected row count (default 1).
	rows int
}

// baseMeta returns a reusable Meta with known values.
func baseMeta() *xatu.Meta {
	return &xatu.Meta{
		Client: &xatu.ClientMeta{
			Name:           "test-client",
			Version:        "0.1.0",
			Id:             "client-id-1",
			Implementation: "xatu",
			Os:             "linux",
			Ethereum: &xatu.ClientMeta_Ethereum{
				Network: &xatu.ClientMeta_Ethereum_Network{
					Name: "mainnet",
					Id:   1,
				},
				Consensus: &xatu.ClientMeta_Ethereum_Consensus{
					Implementation: "lighthouse",
					Version:        "v4.5.6",
				},
				Execution: &xatu.ClientMeta_Ethereum_Execution{
					Implementation: "geth",
					Version:        "v1.13.0",
				},
			},
		},
	}
}

// ts is a helper returning a deterministic timestamp for tests.
func ts() *timestamppb.Timestamp {
	return timestamppb.New(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))
}

// metaWithAdditional returns baseMeta with the given additional_data set.
func metaWithAdditional(ad *xatu.ClientMeta) *xatu.Meta {
	m := baseMeta()
	m.Client = ad
	m.Client.Name = "test-client"
	m.Client.Version = "0.1.0"
	m.Client.Id = "client-id-1"
	m.Client.Implementation = "xatu"
	m.Client.Os = "linux"
	m.Client.Ethereum = &xatu.ClientMeta_Ethereum{
		Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1},
		Consensus: &xatu.ClientMeta_Ethereum_Consensus{
			Implementation: "lighthouse",
			Version:        "v4.5.6",
		},
		Execution: &xatu.ClientMeta_Ethereum_Execution{
			Implementation: "geth",
			Version:        "v1.13.0",
		},
	}

	return m
}

// slotEpochAdditional returns common slot/epoch additional data for beacon tests.
func slotEpochAdditional() *xatu.SlotV2 {
	return &xatu.SlotV2{
		Number:        wrapperspb.UInt64(100),
		StartDateTime: timestamppb.New(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
	}
}

func epochAdditional() *xatu.EpochV2 {
	return &xatu.EpochV2{
		Number:        wrapperspb.UInt64(3),
		StartDateTime: timestamppb.New(time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)),
	}
}

// beaconFixtures returns test fixtures for the beacon domain.
func beaconFixtures() []testFixture {
	return []testFixture{
		{
			table: "beacon_api_eth_v1_events_head",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
					DateTime: ts(),
					Id:       "head-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsHeadV2{
						EthV1EventsHeadV2: &xatu.ClientMeta_AdditionalEthV1EventsHeadV2Data{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsHeadV2{
					EthV1EventsHeadV2: &ethv1.EventHeadV2{
						Slot:            wrapperspb.UInt64(100),
						Block:           "0xblock1",
						State:           "0xstate1",
						EpochTransition: true,
					},
				},
			},
			checks: map[string]any{
				"slot":              uint32(100),
				"block":             "0xblock1",
				"epoch_transition":  true,
				"meta_client_name":  "test-client",
				"meta_network_name": "mainnet",
			},
		},
		{
			table: "beacon_api_eth_v1_events_block",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
					DateTime: ts(),
					Id:       "block-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsBlockV2{
						EthV1EventsBlockV2: &xatu.ClientMeta_AdditionalEthV1EventsBlockV2Data{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsBlockV2{
					EthV1EventsBlockV2: &ethv1.EventBlockV2{
						Slot:  wrapperspb.UInt64(100),
						Block: "0xblockroot",
					},
				},
			},
			checks: map[string]any{
				"slot":  uint32(100),
				"block": "0xblockroot",
			},
		},
		{
			table: "beacon_api_eth_v1_events_attestation",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
					DateTime: ts(),
					Id:       "att-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsAttestationV2{
						EthV1EventsAttestationV2: &xatu.ClientMeta_AdditionalEthV1EventsAttestationV2Data{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsAttestationV2{
					EthV1EventsAttestationV2: &ethv1.AttestationV2{
						AggregationBits: "0xff",
						Signature:       "0xsig",
						Data: &ethv1.AttestationDataV2{
							Slot:            wrapperspb.UInt64(100),
							Index:           wrapperspb.UInt64(5),
							BeaconBlockRoot: "0xbbr",
							Source: &ethv1.CheckpointV2{
								Epoch: wrapperspb.UInt64(2),
								Root:  "0xsource",
							},
							Target: &ethv1.CheckpointV2{
								Epoch: wrapperspb.UInt64(3),
								Root:  "0xtarget",
							},
						},
					},
				},
			},
			checks: map[string]any{
				"slot":              uint32(100),
				"committee_index":   "5",
				"beacon_block_root": "0xbbr",
				"source_epoch":      uint32(2),
				"target_epoch":      uint32(3),
			},
		},
		{
			table: "beacon_api_eth_v1_events_blob_sidecar",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
					DateTime: ts(),
					Id:       "blobsc-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsBlobSidecar{
						EthV1EventsBlobSidecar: &xatu.ClientMeta_AdditionalEthV1EventsBlobSidecarData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsBlobSidecar{
					EthV1EventsBlobSidecar: &ethv1.EventBlobSidecar{
						BlockRoot:     "0xbr",
						Slot:          wrapperspb.UInt64(100),
						Index:         wrapperspb.UInt64(2),
						KzgCommitment: "0xkzg",
						VersionedHash: "0xvh",
					},
				},
			},
			checks: map[string]any{
				"slot":       uint32(100),
				"blob_index": uint64(2),
				"block_root": "0xbr",
			},
		},
		{
			table: "beacon_api_eth_v1_events_chain_reorg",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
					DateTime: ts(),
					Id:       "reorg-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsChainReorgV2{
						EthV1EventsChainReorgV2: &xatu.ClientMeta_AdditionalEthV1EventsChainReorgV2Data{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsChainReorgV2{
					EthV1EventsChainReorgV2: &ethv1.EventChainReorgV2{
						Slot:  wrapperspb.UInt64(100),
						Depth: wrapperspb.UInt64(3),
					},
				},
			},
			checks: map[string]any{
				"slot":  uint32(100),
				"depth": uint16(3),
			},
		},
		{
			table: "beacon_api_eth_v1_events_finalized_checkpoint",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
					DateTime: ts(),
					Id:       "fc-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsFinalizedCheckpointV2{
						EthV1EventsFinalizedCheckpointV2: &xatu.ClientMeta_AdditionalEthV1EventsFinalizedCheckpointV2Data{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsFinalizedCheckpointV2{
					EthV1EventsFinalizedCheckpointV2: &ethv1.EventFinalizedCheckpointV2{
						Block: "0xfcblock",
						State: "0xfcstate",
						Epoch: wrapperspb.UInt64(3),
					},
				},
			},
			checks: map[string]any{
				"block": "0xfcblock",
				"state": "0xfcstate",
				"epoch": uint32(3),
			},
		},
		{
			table: "beacon_api_eth_v1_events_voluntary_exit",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
					DateTime: ts(),
					Id:       "vexit-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsVoluntaryExitV2{
						EthV1EventsVoluntaryExitV2: &xatu.ClientMeta_AdditionalEthV1EventsVoluntaryExitV2Data{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsVoluntaryExitV2{
					EthV1EventsVoluntaryExitV2: &ethv1.EventVoluntaryExitV2{
						Message: &ethv1.EventVoluntaryExitMessageV2{
							Epoch:          wrapperspb.UInt64(3),
							ValidatorIndex: wrapperspb.UInt64(42),
						},
					},
				},
			},
			checks: map[string]any{
				"epoch":           uint32(3),
				"validator_index": uint32(42),
			},
		},
		{
			table: "beacon_api_eth_v1_events_contribution_and_proof",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
					DateTime: ts(),
					Id:       "cp-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsContributionAndProofV2{
						EthV1EventsContributionAndProofV2: &xatu.ClientMeta_AdditionalEthV1EventsContributionAndProofV2Data{},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsContributionAndProofV2{
					EthV1EventsContributionAndProofV2: &ethv1.EventContributionAndProofV2{
						Message: &ethv1.ContributionAndProofV2{
							AggregatorIndex: wrapperspb.UInt64(99),
						},
					},
				},
			},
			checks: map[string]any{
				"aggregator_index": uint32(99),
			},
		},
		{
			table: "beacon_api_eth_v1_events_block_gossip",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP,
					DateTime: ts(),
					Id:       "bg-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsBlockGossip{
						EthV1EventsBlockGossip: &xatu.ClientMeta_AdditionalEthV1EventsBlockGossipData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsBlockGossip{
					EthV1EventsBlockGossip: &ethv1.EventBlockGossip{
						Slot:  wrapperspb.UInt64(100),
						Block: "0xgossipblock",
					},
				},
			},
			checks: map[string]any{
				"slot":  uint32(100),
				"block": "0xgossipblock",
			},
		},
		{
			table: "beacon_api_eth_v1_events_data_column_sidecar",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
					DateTime: ts(),
					Id:       "dcs-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1EventsDataColumnSidecar{
						EthV1EventsDataColumnSidecar: &xatu.ClientMeta_AdditionalEthV1EventsDataColumnSidecarData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1EventsDataColumnSidecar{
					EthV1EventsDataColumnSidecar: &ethv1.EventDataColumnSidecar{
						BlockRoot: "0xdcsblock",
						Slot:      wrapperspb.UInt64(100),
						Index:     wrapperspb.UInt64(7),
					},
				},
			},
			checks: map[string]any{
				"slot":         uint32(100),
				"column_index": uint64(7),
				"block_root":   "0xdcsblock",
			},
		},
		{
			table: "beacon_api_eth_v1_beacon_committee",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
					DateTime: ts(),
					Id:       "comm-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1BeaconCommittee{
						EthV1BeaconCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
							StateId: "head",
							Slot:    slotEpochAdditional(),
							Epoch:   epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1BeaconCommittee{
					EthV1BeaconCommittee: &ethv1.Committee{
						Index: wrapperspb.UInt64(5),
						Slot:  wrapperspb.UInt64(100),
					},
				},
			},
			checks: map[string]any{
				"slot":            uint32(100),
				"committee_index": "5",
			},
		},
		{
			table: "beacon_api_eth_v1_validator_attestation_data",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
					DateTime: ts(),
					Id:       "vad-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1ValidatorAttestationData{
						EthV1ValidatorAttestationData: &xatu.ClientMeta_AdditionalEthV1ValidatorAttestationDataData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1ValidatorAttestationData{
					EthV1ValidatorAttestationData: &ethv1.AttestationDataV2{
						Slot:  wrapperspb.UInt64(100),
						Index: wrapperspb.UInt64(8),
					},
				},
			},
			checks: map[string]any{
				"slot":            uint32(100),
				"committee_index": "8",
			},
		},
		{
			table: "beacon_api_eth_v1_proposer_duty",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
					DateTime: ts(),
					Id:       "pd-head-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1ProposerDuty{
						EthV1ProposerDuty: &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
							StateId: "head",
							Epoch:   epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1ProposerDuty{
					EthV1ProposerDuty: &ethv1.ProposerDuty{
						Slot:           wrapperspb.UInt64(100),
						ValidatorIndex: wrapperspb.UInt64(55),
						Pubkey:         "0xpub",
					},
				},
			},
			checks: map[string]any{
				"slot":                     uint32(100),
				"proposer_validator_index": uint32(55),
			},
		},
		{
			table: "beacon_api_eth_v2_beacon_block",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
					DateTime: ts(),
					Id:       "bb-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{
						EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
							FinalizedWhenRequested: false,
							Slot:                   slotEpochAdditional(),
							Epoch:                  epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
					EthV2BeaconBlockV2: &ethv2.EventBlockV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "beacon_api_eth_v3_validator_block",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
					DateTime: ts(),
					Id:       "vb3-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV3ValidatorBlock{
						EthV3ValidatorBlock: &xatu.ClientMeta_AdditionalEthV3ValidatorBlockData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV3ValidatorBlock{
					EthV3ValidatorBlock: &ethv2.EventBlockV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "beacon_api_eth_v1_beacon_blob",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
					DateTime: ts(),
					Id:       "blob-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1BeaconBlob{
						EthV1BeaconBlob: &xatu.ClientMeta_AdditionalEthV1BeaconBlobData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1BeaconBlob{
					EthV1BeaconBlob: &ethv1.Blob{
						Slot:  wrapperspb.UInt64(100),
						Index: wrapperspb.UInt64(1),
					},
				},
			},
			checks: map[string]any{
				"slot":       uint32(100),
				"blob_index": uint64(1),
			},
		},
	}
}

// canonicalFixtures returns test fixtures for the canonical domain.
func canonicalFixtures() []testFixture {
	return []testFixture{
		{
			table: "canonical_beacon_block",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
					DateTime: ts(),
					Id:       "cbb-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{
						EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
							FinalizedWhenRequested: true,
							Slot:                   slotEpochAdditional(),
							Epoch:                  epochAdditional(),
							Version:                "deneb",
							BlockRoot:              "0xblockroot",
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
					EthV2BeaconBlockV2: &ethv2.EventBlockV2{
						Message: &ethv2.EventBlockV2_DenebBlock{
							DenebBlock: &ethv2.BeaconBlockDeneb{
								Slot:          wrapperspb.UInt64(100),
								ProposerIndex: wrapperspb.UInt64(42),
							},
						},
					},
				},
			},
			checks: map[string]any{
				"block_version": "deneb",
				"block_root":    "0xblockroot",
				"slot":          uint32(100),
			},
		},
		{
			table: "canonical_beacon_committee",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
					DateTime: ts(),
					Id:       "ccomm-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1BeaconCommittee{
						EthV1BeaconCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
							StateId: "finalized",
							Slot:    slotEpochAdditional(),
							Epoch:   epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1BeaconCommittee{
					EthV1BeaconCommittee: &ethv1.Committee{
						Index: wrapperspb.UInt64(7),
						Slot:  wrapperspb.UInt64(100),
					},
				},
			},
			checks: map[string]any{
				"slot":            uint32(100),
				"committee_index": "7",
			},
		},
		{
			table: "canonical_beacon_blob_sidecar",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
					DateTime: ts(),
					Id:       "cbbs-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1BeaconBlobSidecar{
						EthV1BeaconBlobSidecar: &xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1BeaconBlockBlobSidecar{
					EthV1BeaconBlockBlobSidecar: &ethv1.BlobSidecar{
						Slot:  wrapperspb.UInt64(100),
						Index: wrapperspb.UInt64(0),
					},
				},
			},
			checks: map[string]any{
				"slot":       uint32(100),
				"blob_index": uint64(0),
			},
		},
		{
			table: "canonical_beacon_block_attester_slashing",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					DateTime: ts(),
					Id:       "cas-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing{
						EthV2BeaconBlockAttesterSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAttesterSlashingData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
					EthV2BeaconBlockAttesterSlashing: &ethv1.AttesterSlashingV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_block_proposer_slashing",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
					DateTime: ts(),
					Id:       "cps-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockProposerSlashing{
						EthV2BeaconBlockProposerSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockProposerSlashingData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
					EthV2BeaconBlockProposerSlashing: &ethv1.ProposerSlashingV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_block_voluntary_exit",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
					DateTime: ts(),
					Id:       "cbve-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
						EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
					EthV2BeaconBlockVoluntaryExit: &ethv1.SignedVoluntaryExitV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_block_deposit",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
					DateTime: ts(),
					Id:       "cbd-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockDeposit{
						EthV2BeaconBlockDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockDepositData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
					EthV2BeaconBlockDeposit: &ethv1.DepositV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_block_bls_to_execution_change",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
					DateTime: ts(),
					Id:       "cbbls-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockBlsToExecutionChange{
						EthV2BeaconBlockBlsToExecutionChange: &xatu.ClientMeta_AdditionalEthV2BeaconBlockBLSToExecutionChangeData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockBlsToExecutionChange{
					EthV2BeaconBlockBlsToExecutionChange: &ethv2.SignedBLSToExecutionChangeV2{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_block_execution_transaction",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
					DateTime: ts(),
					Id:       "cbtx-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
						EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
					EthV2BeaconBlockExecutionTransaction: &ethv1.Transaction{
						Hash: "0xtxhash",
						From: "0xfrom",
						To:   "0xto",
					},
				},
			},
			checks: map[string]any{
				"hash": "0xtxhash",
				"from": "0xfrom",
			},
		},
		{
			table: "canonical_beacon_block_withdrawal",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
					DateTime: ts(),
					Id:       "cbw-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockWithdrawal{
						EthV2BeaconBlockWithdrawal: &xatu.ClientMeta_AdditionalEthV2BeaconBlockWithdrawalData{
							Block: &xatu.BlockIdentifier{Epoch: epochAdditional()},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockWithdrawal{
					EthV2BeaconBlockWithdrawal: &ethv1.WithdrawalV2{
						Index:          wrapperspb.UInt64(10),
						ValidatorIndex: wrapperspb.UInt64(42),
					},
				},
			},
			checks: map[string]any{
				"withdrawal_index":           uint32(10),
				"withdrawal_validator_index": uint32(42),
			},
		},
		{
			table: "canonical_beacon_block_sync_aggregate",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
					DateTime: ts(),
					Id:       "cbsa-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockSyncAggregate{
					EthV2BeaconBlockSyncAggregate: &xatu.SyncAggregateData{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_elaborated_attestation",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
					DateTime: ts(),
					Id:       "cea-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockElaboratedAttestation{
						EthV2BeaconBlockElaboratedAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockElaboratedAttestationData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV2BeaconBlockElaboratedAttestation{
					EthV2BeaconBlockElaboratedAttestation: &ethv1.ElaboratedAttestation{
						ValidatorIndexes: []*wrapperspb.UInt64Value{
							wrapperspb.UInt64(11), wrapperspb.UInt64(22),
						},
					},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_proposer_duty",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
					DateTime: ts(),
					Id:       "cpd-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1ProposerDuty{
						EthV1ProposerDuty: &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
							StateId: "finalized",
							Slot:    slotEpochAdditional(),
							Epoch:   epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1ProposerDuty{
					EthV1ProposerDuty: &ethv1.ProposerDuty{
						Slot:           wrapperspb.UInt64(100),
						ValidatorIndex: wrapperspb.UInt64(77),
					},
				},
			},
			checks: map[string]any{
				"slot":                     uint32(100),
				"proposer_validator_index": uint32(77),
			},
		},
		{
			table: "canonical_beacon_sync_committee",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
					DateTime: ts(),
					Id:       "csc-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1BeaconSyncCommittee{
						EthV1BeaconSyncCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1BeaconSyncCommittee{
					EthV1BeaconSyncCommittee: &xatu.SyncCommitteeData{
						SyncCommittee: &ethv1.SyncCommittee{
							ValidatorAggregates: []*ethv1.SyncCommitteeValidatorAggregate{
								{Validators: []*wrapperspb.UInt64Value{wrapperspb.UInt64(1)}},
							},
						},
					},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "canonical_beacon_validators",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
					DateTime: ts(),
					Id:       "cv-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1Validators{
						EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1Validators{
					EthV1Validators: &xatu.Validators{
						Validators: []*ethv1.Validator{
							{
								Index:   wrapperspb.UInt64(42),
								Balance: wrapperspb.UInt64(32_000_000_000),
								Status:  wrapperspb.String("active_ongoing"),
								Data: &ethv1.ValidatorData{
									Pubkey:           wrapperspb.String("0xpub"),
									EffectiveBalance: wrapperspb.UInt64(32_000_000_000),
									Slashed:          wrapperspb.Bool(false),
								},
							},
						},
					},
				},
			},
			checks: map[string]any{
				"index":   uint32(42),
				"balance": uint64(32_000_000_000),
				"status":  "active_ongoing",
				"epoch":   uint32(3),
			},
		},
		{
			table: "canonical_beacon_validators_pubkeys",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
					DateTime: ts(),
					Id:       "cvp-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1Validators{
						EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1Validators{
					EthV1Validators: &xatu.Validators{
						Validators: []*ethv1.Validator{
							{
								Index: wrapperspb.UInt64(42),
								Data: &ethv1.ValidatorData{
									Pubkey: wrapperspb.String("0xpub"),
								},
							},
						},
					},
				},
			},
			checks: map[string]any{
				"index": uint32(42),
				"epoch": uint32(3),
			},
		},
		{
			table: "canonical_beacon_validators_withdrawal_credentials",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
					DateTime: ts(),
					Id:       "cvwc-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_EthV1Validators{
						EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_EthV1Validators{
					EthV1Validators: &xatu.Validators{
						Validators: []*ethv1.Validator{
							{
								Index: wrapperspb.UInt64(42),
								Data: &ethv1.ValidatorData{
									WithdrawalCredentials: wrapperspb.String("0xcreds"),
								},
							},
						},
					},
				},
			},
			checks: map[string]any{
				"index": uint32(42),
				"epoch": uint32(3),
			},
		},
	}
}

// libp2pFixtures returns test fixtures for the libp2p domain.
func libp2pFixtures() []testFixture {
	const (
		testPeerID    = "16Uiu2HAmPeer1"
		testNetwork   = "mainnet"
		testTopic     = "/eth2/bba4da96/beacon_block/ssz_snappy"
		testEventID   = "test-event-123"
		testRootEvent = "root-event-456"
	)

	expectedPeerIDKey := flattener.SeaHashInt64(testPeerID + testNetwork)

	return []testFixture{
		{
			table: "libp2p_connected",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_CONNECTED,
					DateTime: ts(),
					Id:       "conn-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
					Libp2PTraceConnected: &libp2p.Connected{
						RemotePeer:   wrapperspb.String(testPeerID),
						RemoteMaddrs: wrapperspb.String("/ip4/1.2.3.4/tcp/9000"),
						AgentVersion: wrapperspb.String("lighthouse/v4.5.6/linux"),
						Direction:    wrapperspb.String("inbound"),
					},
				},
			},
			checks: map[string]any{
				"remote_protocol":             "ip4",
				"remote_ip":                   "1.2.3.4",
				"remote_transport_protocol":   "tcp",
				"remote_port":                 uint16(9000),
				"remote_agent_implementation": "lighthouse",
				"remote_agent_version":        "v4.5.6",
				"remote_agent_version_major":  "4",
				"remote_agent_version_minor":  "5",
				"remote_agent_version_patch":  "6",
				"remote_agent_platform":       "linux",
				"direction":                   "inbound",
				"remote_peer_id_unique_key":   expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_disconnected",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_DISCONNECTED,
					DateTime: ts(),
					Id:       "disc-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceDisconnected{
					Libp2PTraceDisconnected: &libp2p.Disconnected{
						RemotePeer:   wrapperspb.String(testPeerID),
						RemoteMaddrs: wrapperspb.String("/ip4/5.6.7.8/udp/30303"),
						Direction:    wrapperspb.String("outbound"),
					},
				},
			},
			checks: map[string]any{
				"remote_protocol":           "ip4",
				"remote_ip":                 "5.6.7.8",
				"remote_transport_protocol": "udp",
				"remote_port":               uint16(30303),
				"direction":                 "outbound",
				"remote_peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_add_peer",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_ADD_PEER,
					DateTime: ts(),
					Id:       "addp-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceAddPeer{
					Libp2PTraceAddPeer: &libp2p.AddPeer{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_remove_peer",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
					DateTime: ts(),
					Id:       "rmp-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRemovePeer{
					Libp2PTraceRemovePeer: &libp2p.RemovePeer{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_send_rpc",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_SEND_RPC,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceSendRpc{
					Libp2PTraceSendRpc: &libp2p.SendRPC{
						PeerId: wrapperspb.String(testPeerID),
						Meta:   &libp2p.RPCMeta{PeerId: wrapperspb.String(testPeerID)},
					},
				},
			},
			checks: map[string]any{
				"unique_key":         flattener.SeaHashInt64(testEventID),
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_recv_rpc",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RECV_RPC,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRecvRpc{
					Libp2PTraceRecvRpc: &libp2p.RecvRPC{
						PeerId: wrapperspb.String(testPeerID),
						Meta:   &libp2p.RPCMeta{PeerId: wrapperspb.String(testPeerID)},
					},
				},
			},
			checks: map[string]any{
				"unique_key":         flattener.SeaHashInt64(testEventID),
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_drop_rpc",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_DROP_RPC,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceDropRpc{
					Libp2PTraceDropRpc: &libp2p.DropRPC{
						PeerId: wrapperspb.String(testPeerID),
						Meta:   &libp2p.RPCMeta{PeerId: wrapperspb.String(testPeerID)},
					},
				},
			},
			checks: map[string]any{
				"unique_key":         flattener.SeaHashInt64(testEventID),
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_join",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_JOIN,
					DateTime: ts(),
					Id:       "join-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceJoin{
					Libp2PTraceJoin: &libp2p.Join{
						Topic: wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"topic_layer":             "eth2",
				"topic_fork_digest_value": "bba4da96",
				"topic_name":              "beacon_block",
				"topic_encoding":          "ssz_snappy",
			},
		},
		{
			table: "libp2p_leave",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_LEAVE,
					DateTime: ts(),
					Id:       "leave-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceLeave{
					Libp2PTraceLeave: &libp2p.Leave{
						Topic: wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"topic_layer":    "eth2",
				"topic_name":     "beacon_block",
				"topic_encoding": "ssz_snappy",
			},
		},
		{
			table: "libp2p_graft",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_GRAFT,
					DateTime: ts(),
					Id:       "graft-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceGraft{
					Libp2PTraceGraft: &libp2p.Graft{
						PeerId: wrapperspb.String(testPeerID),
						Topic:  wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
				"topic_layer":        "eth2",
				"topic_name":         "beacon_block",
			},
		},
		{
			table: "libp2p_prune",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_PRUNE,
					DateTime: ts(),
					Id:       "prune-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTracePrune{
					Libp2PTracePrune: &libp2p.Prune{
						PeerId: wrapperspb.String(testPeerID),
						Topic:  wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
				"topic_layer":        "eth2",
			},
		},
		{
			table: "libp2p_deliver_message",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE,
					DateTime: ts(),
					Id:       "dm-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceDeliverMessage{
					Libp2PTraceDeliverMessage: &libp2p.DeliverMessage{
						PeerId: wrapperspb.String(testPeerID),
						Topic:  wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
				"topic_layer":        "eth2",
				"topic_name":         "beacon_block",
			},
		},
		{
			table: "libp2p_duplicate_message",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE,
					DateTime: ts(),
					Id:       "dup-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceDuplicateMessage{
					Libp2PTraceDuplicateMessage: &libp2p.DuplicateMessage{
						PeerId: wrapperspb.String(testPeerID),
						Topic:  wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
				"topic_layer":        "eth2",
			},
		},
		{
			table: "libp2p_publish_message",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
					DateTime: ts(),
					Id:       "pub-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTracePublishMessage{
					Libp2PTracePublishMessage: &libp2p.PublishMessage{
						Topic: wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"topic_layer": "eth2",
				"topic_name":  "beacon_block",
			},
		},
		{
			table: "libp2p_reject_message",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE,
					DateTime: ts(),
					Id:       "rej-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRejectMessage{
					Libp2PTraceRejectMessage: &libp2p.RejectMessage{
						PeerId: wrapperspb.String(testPeerID),
						Topic:  wrapperspb.String(testTopic),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
				"topic_layer":        "eth2",
			},
		},
		{
			table: "libp2p_handle_metadata",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_HANDLE_METADATA,
					DateTime: ts(),
					Id:       "hm-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceHandleMetadata{
					Libp2PTraceHandleMetadata: &libp2p.HandleMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_handle_status",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_HANDLE_STATUS,
					DateTime: ts(),
					Id:       "hs-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceHandleStatus{
					Libp2PTraceHandleStatus: &libp2p.HandleStatus{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_peer",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_CONNECTED,
					DateTime: ts(),
					Id:       "peer-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
					Libp2PTraceConnected: &libp2p.Connected{
						RemotePeer: wrapperspb.String(testPeerID),
					},
				},
			},
			checks: map[string]any{
				"peer_id":    testPeerID,
				"unique_key": flattener.SeaHashInt64(testPeerID + testNetwork),
			},
		},
		// RPC meta tables
		{
			table: "libp2p_rpc_meta_control_ihave",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE,
					DateTime: ts(),
					Id:       "ihave-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIhave{
					Libp2PTraceRpcMetaControlIhave: &libp2p.ControlIHaveMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						Topic:        wrapperspb.String(testTopic),
						MessageIndex: wrapperspb.UInt32(0),
						ControlIndex: wrapperspb.UInt32(1),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"peer_id_unique_key":  expectedPeerIDKey,
				"topic_layer":         "eth2",
				"topic_name":          "beacon_block",
			},
		},
		{
			table: "libp2p_rpc_meta_control_iwant",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
					DateTime: ts(),
					Id:       "iwant-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIwant{
					Libp2PTraceRpcMetaControlIwant: &libp2p.ControlIWantMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						MessageIndex: wrapperspb.UInt32(0),
						ControlIndex: wrapperspb.UInt32(1),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"peer_id_unique_key":  expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_rpc_meta_control_idontwant",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
					DateTime: ts(),
					Id:       "idw-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIdontwant{
					Libp2PTraceRpcMetaControlIdontwant: &libp2p.ControlIDontWantMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						MessageIndex: wrapperspb.UInt32(0),
						ControlIndex: wrapperspb.UInt32(1),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"peer_id_unique_key":  expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_rpc_meta_control_graft",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
					DateTime: ts(),
					Id:       "graft-meta-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlGraft{
					Libp2PTraceRpcMetaControlGraft: &libp2p.ControlGraftMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						Topic:        wrapperspb.String(testTopic),
						ControlIndex: wrapperspb.UInt32(0),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"topic_layer":         "eth2",
				"topic_name":          "beacon_block",
			},
		},
		{
			table: "libp2p_rpc_meta_control_prune",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE,
					DateTime: ts(),
					Id:       "prune-meta-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune{
					Libp2PTraceRpcMetaControlPrune: &libp2p.ControlPruneMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						GraftPeerId:  wrapperspb.String("graft-peer-1"),
						Topic:        wrapperspb.String(testTopic),
						ControlIndex: wrapperspb.UInt32(0),
						PeerIndex:    wrapperspb.UInt32(0),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"peer_id_unique_key":  expectedPeerIDKey,
				"topic_layer":         "eth2",
			},
		},
		{
			table: "libp2p_rpc_meta_subscription",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION,
					DateTime: ts(),
					Id:       "sub-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaSubscription{
					Libp2PTraceRpcMetaSubscription: &libp2p.SubMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						TopicId:      wrapperspb.String(testTopic),
						Subscribe:    wrapperspb.Bool(true),
						ControlIndex: wrapperspb.UInt32(0),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"topic_layer":         "eth2",
				"topic_name":          "beacon_block",
			},
		},
		{
			table: "libp2p_rpc_meta_message",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE,
					DateTime: ts(),
					Id:       "msg-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaMessage{
					Libp2PTraceRpcMetaMessage: &libp2p.MessageMetaItem{
						RootEventId:  wrapperspb.String(testRootEvent),
						PeerId:       wrapperspb.String(testPeerID),
						TopicId:      wrapperspb.String(testTopic),
						ControlIndex: wrapperspb.UInt32(0),
					},
				},
			},
			checks: map[string]any{
				"rpc_meta_unique_key": flattener.SeaHashInt64(testRootEvent),
				"peer_id_unique_key":  expectedPeerIDKey,
				"topic_layer":         "eth2",
				"topic_name":          "beacon_block",
			},
		},
		// Gossipsub tables
		{
			table: "libp2p_gossipsub_beacon_block",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubBeaconBlock{
						Libp2PTraceGossipsubBeaconBlock: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconBlockData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
							Topic: wrapperspb.String(testTopic),
							Metadata: &libp2p.TraceEventMetadata{
								PeerId: wrapperspb.String(testPeerID),
							},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconBlock{
					Libp2PTraceGossipsubBeaconBlock: &gossipsub.BeaconBlock{},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
				"slot":               uint32(100),
				"epoch":              uint32(3),
				"topic_layer":        "eth2",
				"topic_name":         "beacon_block",
			},
		},
		{
			table: "libp2p_gossipsub_beacon_attestation",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubBeaconAttestation{
						Libp2PTraceGossipsubBeaconAttestation: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
							Metadata: &libp2p.TraceEventMetadata{
								PeerId: wrapperspb.String(testPeerID),
							},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconAttestation{
					Libp2PTraceGossipsubBeaconAttestation: &ethv1.Attestation{
						Data: &ethv1.AttestationData{
							Slot: 100,
						},
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_gossipsub_blob_sidecar",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubBlobSidecar{
						Libp2PTraceGossipsubBlobSidecar: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
							Topic: wrapperspb.String(testTopic),
							Metadata: &libp2p.TraceEventMetadata{
								PeerId: wrapperspb.String(testPeerID),
							},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBlobSidecar{
					Libp2PTraceGossipsubBlobSidecar: &gossipsub.BlobSidecar{},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_gossipsub_data_column_sidecar",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubDataColumnSidecar{
						Libp2PTraceGossipsubDataColumnSidecar: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
							Topic: wrapperspb.String(testTopic),
							Metadata: &libp2p.TraceEventMetadata{
								PeerId: wrapperspb.String(testPeerID),
							},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubDataColumnSidecar{
					Libp2PTraceGossipsubDataColumnSidecar: &gossipsub.DataColumnSidecar{},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_gossipsub_aggregate_and_proof",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubAggregateAndProof{
						Libp2PTraceGossipsubAggregateAndProof: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubAggregateAndProofData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
							Metadata: &libp2p.TraceEventMetadata{
								PeerId: wrapperspb.String(testPeerID),
							},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubAggregateAndProof{
					Libp2PTraceGossipsubAggregateAndProof: &ethv1.SignedAggregateAttestationAndProofV2{},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_rpc_data_column_custody_probe",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE,
					DateTime: ts(),
					Id:       testEventID,
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_Libp2PTraceRpcDataColumnCustodyProbe{
						Libp2PTraceRpcDataColumnCustodyProbe: &xatu.ClientMeta_AdditionalLibP2PTraceRpcDataColumnCustodyProbeData{},
					},
				}),
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcDataColumnCustodyProbe{
					Libp2PTraceRpcDataColumnCustodyProbe: &libp2p.DataColumnCustodyProbe{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
			checks: map[string]any{
				"peer_id_unique_key": expectedPeerIDKey,
			},
		},
		{
			table: "libp2p_synthetic_heartbeat",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT,
					DateTime: ts(),
					Id:       "hb-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_Libp2PTraceSyntheticHeartbeat{
					Libp2PTraceSyntheticHeartbeat: &libp2p.SyntheticHeartbeat{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
	}
}

// executionFixtures returns test fixtures for the execution domain.
func executionFixtures() []testFixture {
	return []testFixture{
		{
			table: "mempool_transaction",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_MEMPOOL_TRANSACTION_V2,
					DateTime: ts(),
					Id:       "mtx-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_MempoolTransactionV2{
						MempoolTransactionV2: &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
							Hash:  "0xhash",
							From:  "0xfrom",
							Nonce: wrapperspb.UInt64(42),
						},
					},
				}),
			},
			checks: map[string]any{
				"hash": "0xhash",
				"from": "0xfrom",
			},
		},
		{
			table: "execution_block_metrics",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_EXECUTION_BLOCK_METRICS,
					DateTime: ts(),
					Id:       "ebm-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_ExecutionBlockMetrics{
					ExecutionBlockMetrics: &xatu.ExecutionBlockMetrics{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "consensus_engine_api_new_payload",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
					DateTime: ts(),
					Id:       "cenp-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_ConsensusEngineApiNewPayload{
						ConsensusEngineApiNewPayload: &xatu.ClientMeta_AdditionalConsensusEngineAPINewPayloadData{},
					},
				}),
				Data: &xatu.DecoratedEvent_ConsensusEngineApiNewPayload{
					ConsensusEngineApiNewPayload: &xatu.ConsensusEngineAPINewPayload{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "consensus_engine_api_get_blobs",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS,
					DateTime: ts(),
					Id:       "cegb-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_ConsensusEngineApiGetBlobs{
						ConsensusEngineApiGetBlobs: &xatu.ClientMeta_AdditionalConsensusEngineAPIGetBlobsData{},
					},
				}),
				Data: &xatu.DecoratedEvent_ConsensusEngineApiGetBlobs{
					ConsensusEngineApiGetBlobs: &xatu.ConsensusEngineAPIGetBlobs{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "execution_engine_new_payload",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
					DateTime: ts(),
					Id:       "eenp-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_ExecutionEngineNewPayload{
					ExecutionEngineNewPayload: &xatu.ExecutionEngineNewPayload{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "execution_engine_get_blobs",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
					DateTime: ts(),
					Id:       "eegb-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_ExecutionEngineGetBlobs{
					ExecutionEngineGetBlobs: &xatu.ExecutionEngineGetBlobs{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "execution_state_size",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_EXECUTION_STATE_SIZE,
					DateTime: ts(),
					Id:       "ess-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_ExecutionStateSize{
					ExecutionStateSize: &xatu.ExecutionStateSize{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
	}
}

// mevFixtures returns test fixtures for the MEV domain.
func mevFixtures() []testFixture {
	return []testFixture{
		{
			table: "mev_relay_bid_trace",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
					DateTime: ts(),
					Id:       "mev-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_MevRelayBidTraceBuilderBlockSubmission{
						MevRelayBidTraceBuilderBlockSubmission: &xatu.ClientMeta_AdditionalMevRelayBidTraceBuilderBlockSubmissionData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
							Relay: &mevrelay.Relay{
								Name: wrapperspb.String("flashbots"),
							},
						},
					},
				}),
				Data: &xatu.DecoratedEvent_MevRelayBidTraceBuilderBlockSubmission{
					MevRelayBidTraceBuilderBlockSubmission: &mevrelay.BidTrace{
						Slot:                 wrapperspb.UInt64(100),
						BuilderPubkey:        wrapperspb.String("0xbuilder"),
						ProposerPubkey:       wrapperspb.String("0xproposer"),
						ProposerFeeRecipient: wrapperspb.String("0xfee"),
						Value:                wrapperspb.String("1000000000000000000"),
					},
				},
			},
			checks: map[string]any{
				"slot":           uint32(100),
				"builder_pubkey": "0xbuilder",
				"relay_name":     "flashbots",
			},
		},
		{
			table: "mev_relay_proposer_payload_delivered",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED,
					DateTime: ts(),
					Id:       "mevp-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_MevRelayPayloadDelivered{
						MevRelayPayloadDelivered: &xatu.ClientMeta_AdditionalMevRelayPayloadDeliveredData{
							Slot:  slotEpochAdditional(),
							Epoch: epochAdditional(),
						},
					},
				}),
				Data: &xatu.DecoratedEvent_MevRelayPayloadDelivered{
					MevRelayPayloadDelivered: &mevrelay.ProposerPayloadDelivered{
						Slot:          wrapperspb.UInt64(100),
						BuilderPubkey: wrapperspb.String("0xbuilder"),
					},
				},
			},
			checks: map[string]any{
				"slot":           uint32(100),
				"builder_pubkey": "0xbuilder",
			},
		},
		{
			table: "mev_relay_validator_registration",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
					DateTime: ts(),
					Id:       "mevvr-1",
				},
				Meta: metaWithAdditional(&xatu.ClientMeta{
					AdditionalData: &xatu.ClientMeta_MevRelayValidatorRegistration{
						MevRelayValidatorRegistration: &xatu.ClientMeta_AdditionalMevRelayValidatorRegistrationData{},
					},
				}),
				Data: &xatu.DecoratedEvent_MevRelayValidatorRegistration{
					MevRelayValidatorRegistration: &mevrelay.ValidatorRegistration{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
	}
}

// nodeFixtures returns test fixtures for the node domain.
func nodeFixtures() []testFixture {
	return []testFixture{
		{
			table: "node_record_consensus",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_NODE_RECORD_CONSENSUS,
					DateTime: ts(),
					Id:       "nrc-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_NodeRecordConsensus{
					NodeRecordConsensus: &noderecord.Consensus{
						PeerId: wrapperspb.String(""),
					},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
		{
			table: "node_record_execution",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_NODE_RECORD_EXECUTION,
					DateTime: ts(),
					Id:       "nre-1",
				},
				Meta: baseMeta(),
				Data: &xatu.DecoratedEvent_NodeRecordExecution{
					NodeRecordExecution: &noderecord.Execution{},
				},
			},
			checks: map[string]any{
				"meta_client_name": "test-client",
			},
		},
	}
}

// TestSnapshotCorrectness verifies exact column values for every route
// by constructing fully-populated DecoratedEvents with known field values,
// flattening them, and asserting specific column values in the snapshot.
func TestSnapshotCorrectness(t *testing.T) {
	allFixtures := make([]testFixture, 0, 80)
	allFixtures = append(allFixtures, beaconFixtures()...)
	allFixtures = append(allFixtures, canonicalFixtures()...)
	allFixtures = append(allFixtures, libp2pFixtures()...)
	allFixtures = append(allFixtures, executionFixtures()...)
	allFixtures = append(allFixtures, mevFixtures()...)
	allFixtures = append(allFixtures, nodeFixtures()...)

	// Build lookup of covered tables for the completeness check.
	coveredTables := make(map[string]struct{}, len(allFixtures))
	for _, f := range allFixtures {
		coveredTables[f.table] = struct{}{}
	}

	for _, tt := range allFixtures {
		t.Run(tt.table, func(t *testing.T) {
			route := findRouteByTable(t, tt.table)
			batch := route.NewBatch()

			meta := metadata.Extract(tt.event)
			err := batch.FlattenTo(tt.event, meta)
			require.NoError(t, err)

			expectedRows := 1
			if tt.rows > 0 {
				expectedRows = tt.rows
			}

			require.Equal(t, expectedRows, batch.Rows(),
				"unexpected row count for table %s", tt.table)

			snapper, ok := batch.(flattener.Snapshotter)
			require.True(t, ok, "batch for %s must implement Snapshotter", tt.table)

			snap := snapper.Snapshot()
			require.Len(t, snap, expectedRows)

			for col, want := range tt.checks {
				got, exists := snap[0][col]
				if !assert.True(t, exists, "column %q not in snapshot for table %s", col, tt.table) {
					continue
				}

				// Trim null-byte padding from FixedString columns for comparison.
				if gotStr, ok := got.(string); ok {
					if wantStr, ok := want.(string); ok {
						assert.Equal(t, wantStr, strings.TrimRight(gotStr, "\x00"),
							"table %s column %q", tt.table, col)

						continue
					}
				}

				assert.Equal(t, want, got, "table %s column %q", tt.table, col)
			}

			// Verify column alignment as a bonus.
			for _, col := range batch.Input() {
				assert.Equalf(t, expectedRows, col.Data.Rows(),
					"column %q misaligned in table %s", col.Name, tt.table)
			}
		})
	}

	// Completeness check: every registered route should have a fixture with assertions.
	t.Run("completeness", func(t *testing.T) {
		missing := make([]string, 0)

		for _, route := range tabledefs.All() {
			if _, ok := coveredTables[route.TableName()]; !ok {
				missing = append(missing, route.TableName())
			}
		}

		require.Emptyf(t, missing,
			"routes without snapshot test fixtures: %v", missing)

		// Every fixture must have at least one column assertion to be meaningful.
		for _, f := range allFixtures {
			assert.NotEmptyf(t, f.checks,
				"fixture for table %q has empty checks — add column assertions", f.table)
		}
	})
}
