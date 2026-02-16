package flattener

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	libp2p "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestConditionalRoutingPredicates(t *testing.T) {
	canonicalCommittee := findFlattenerByTable(t, "canonical_beacon_committee")
	headCommittee := findFlattenerByTable(t, "beacon_api_eth_v1_beacon_committee")
	canonicalBlock := findFlattenerByTable(t, "canonical_beacon_block")

	finalizedCommitteeEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE, DateTime: timestamppb.Now(), Id: "1"},
		Meta:  &xatu.Meta{Client: &xatu.ClientMeta{AdditionalData: &xatu.ClientMeta_EthV1BeaconCommittee{EthV1BeaconCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{StateId: "finalized"}}}},
		Data:  &xatu.DecoratedEvent_EthV1BeaconCommittee{EthV1BeaconCommittee: &ethv1.Committee{}},
	}
	headCommitteeEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE, DateTime: timestamppb.Now(), Id: "2"},
		Meta:  &xatu.Meta{Client: &xatu.ClientMeta{AdditionalData: &xatu.ClientMeta_EthV1BeaconCommittee{EthV1BeaconCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{StateId: "head"}}}},
		Data:  &xatu.DecoratedEvent_EthV1BeaconCommittee{EthV1BeaconCommittee: &ethv1.Committee{}},
	}

	assert.True(t, canonicalCommittee.ShouldProcess(finalizedCommitteeEvent))
	assert.False(t, headCommittee.ShouldProcess(finalizedCommitteeEvent))
	assert.False(t, canonicalCommittee.ShouldProcess(headCommitteeEvent))
	assert.True(t, headCommittee.ShouldProcess(headCommitteeEvent))

	finalizedBlockEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2, DateTime: timestamppb.Now(), Id: "3"},
		Meta:  &xatu.Meta{Client: &xatu.ClientMeta{AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{FinalizedWhenRequested: true}}}},
		Data:  &xatu.DecoratedEvent_EthV2BeaconBlockV2{EthV2BeaconBlockV2: &ethv2.EventBlockV2{}},
	}
	nonFinalizedBlockEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2, DateTime: timestamppb.Now(), Id: "4"},
		Meta:  &xatu.Meta{Client: &xatu.ClientMeta{AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{FinalizedWhenRequested: false}}}},
		Data:  &xatu.DecoratedEvent_EthV2BeaconBlockV2{EthV2BeaconBlockV2: &ethv2.EventBlockV2{}},
	}

	assert.True(t, canonicalBlock.ShouldProcess(finalizedBlockEvent))
	assert.False(t, canonicalBlock.ShouldProcess(nonFinalizedBlockEvent))
}

func TestValidatorsFanout(t *testing.T) {
	pubkeysFlattener := findFlattenerByTable(t, "canonical_beacon_validators_pubkeys")

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS, DateTime: timestamppb.Now(), Id: "validators-1"},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "validator-client",
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1},
				},
				AdditionalData: &xatu.ClientMeta_EthV1Validators{
					EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
						Epoch: &xatu.EpochV2{
							Number:        wrapperspb.UInt64(123),
							StartDateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
						},
					},
				},
			},
		},
		Data: &xatu.DecoratedEvent_EthV1Validators{EthV1Validators: &xatu.Validators{Validators: []*ethv1.Validator{
			{
				Index:   wrapperspb.UInt64(42),
				Status:  wrapperspb.String("active_ongoing"),
				Balance: wrapperspb.UInt64(32_000_000_000),
				Data: &ethv1.ValidatorData{
					Pubkey:                wrapperspb.String("0xabc"),
					WithdrawalCredentials: wrapperspb.String("0xdef"),
					Slashed:               wrapperspb.Bool(false),
					EffectiveBalance:      wrapperspb.UInt64(32_000_000_000),
					ActivationEpoch:       wrapperspb.UInt64(10),
				},
			},
		}}},
	}

	rows, err := pubkeysFlattener.Flatten(event, metadata.Extract(event))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, uint64(42), rows[0]["index"])
	assert.Equal(t, "0xabc", rows[0]["pubkey"])
	assert.Equal(t, uint64(123), rows[0]["epoch"])
	assert.Equal(t, int64(1_700_000_000), rows[0]["epoch_start_date_time"])
}

func TestLibP2PEnrichment(t *testing.T) {
	connectedFlattener := findFlattenerByTable(t, "libp2p_connected")

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_LIBP2P_TRACE_CONNECTED, DateTime: timestamppb.Now(), Id: "libp2p-1"},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "peer-client",
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1},
				},
			},
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceConnected{Libp2PTraceConnected: &libp2p.Connected{
			RemotePeer:   wrapperspb.String("16Uiu2peer"),
			RemoteMaddrs: wrapperspb.String("/ip4/1.2.3.4/tcp/9000"),
		}},
	}

	rows, err := connectedFlattener.Flatten(event, metadata.Extract(event))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "ip4", rows[0]["remote_protocol"])
	assert.Equal(t, "1.2.3.4", rows[0]["remote_ip"])
	assert.Equal(t, "tcp", rows[0]["remote_transport_protocol"])
	assert.Equal(t, uint64(9000), rows[0]["remote_port"])
	assert.NotZero(t, rows[0]["remote_peer_id_unique_key"])
}

func TestSyncCommitteeMutator(t *testing.T) {
	syncCommitteeFlattener := findFlattenerByTable(t, "canonical_beacon_sync_committee")

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE, DateTime: timestamppb.Now(), Id: "sync-1"},
		Meta: &xatu.Meta{Client: &xatu.ClientMeta{AdditionalData: &xatu.ClientMeta_EthV1BeaconSyncCommittee{
			EthV1BeaconSyncCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeData{
				Epoch: &xatu.EpochV2{Number: wrapperspb.UInt64(321)},
			},
		}}},
		Data: &xatu.DecoratedEvent_EthV1BeaconSyncCommittee{EthV1BeaconSyncCommittee: &xatu.SyncCommitteeData{SyncCommittee: &ethv1.SyncCommittee{
			ValidatorAggregates: []*ethv1.SyncCommitteeValidatorAggregate{
				{Validators: []*wrapperspb.UInt64Value{wrapperspb.UInt64(1), wrapperspb.UInt64(2)}},
				{Validators: []*wrapperspb.UInt64Value{wrapperspb.UInt64(3)}},
			},
		}}},
	}

	rows, err := syncCommitteeFlattener.Flatten(event, metadata.Extract(event))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	aggs, ok := rows[0]["validator_aggregates"].([][]uint64)
	require.True(t, ok)
	require.Len(t, aggs, 2)
	assert.Equal(t, []uint64{1, 2}, aggs[0])
	assert.Equal(t, []uint64{3}, aggs[1])
}

func findFlattenerByTable(t *testing.T, table string) Flattener {
	t.Helper()

	for _, f := range All() {
		if f.TableName() == table {
			return f
		}
	}

	t.Fatalf("flattener for table %s not found", table)

	return nil
}
