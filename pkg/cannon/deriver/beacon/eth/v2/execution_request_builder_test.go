package v2

import (
	"context"
	"errors"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	testPayloadBlockNumber = uint64(4242)
	testPayloadBlockHash   = "0x00000000000000000000000000000000000000000000000000000000000000cd"
)

func testBuilderDepositRequest(amount phase0.Gwei) *gloas.BuilderDepositRequest {
	return &gloas.BuilderDepositRequest{
		Pubkey:                phase0.BLSPubKey{0x0a},
		WithdrawalCredentials: append([]byte{0x03}, make([]byte, 31)...),
		Amount:                amount,
		Signature:             phase0.BLSSignature{0x0b},
	}
}

func testBuilderExitRequest(address byte) *gloas.BuilderExitRequest {
	return &gloas.BuilderExitRequest{
		SourceAddress: bellatrix.ExecutionAddress{address},
		Pubkey:        phase0.BLSPubKey{0x0c},
	}
}

func envelopeWithPayload(requests *gloas.ExecutionRequests) *gloas.SignedExecutionPayloadEnvelope {
	return &gloas.SignedExecutionPayloadEnvelope{
		Message: &gloas.ExecutionPayloadEnvelope{
			Payload: &gloas.ExecutionPayload{
				BlockNumber: testPayloadBlockNumber,
				BlockHash:   phase0.Hash32{31: 0xcd},
			},
			ExecutionRequests: requests,
		},
	}
}

func testBlockIdentifier() *xatu.BlockIdentifier {
	return &xatu.BlockIdentifier{
		Root:    "0x00000000000000000000000000000000000000000000000000000000000000ee",
		Version: "gloas",
		Slot:    &xatu.SlotV2{Number: &wrapperspb.UInt64Value{Value: 123}},
		Epoch:   &xatu.EpochV2{Number: &wrapperspb.UInt64Value{Value: 3}},
	}
}

func hezeBlock(slot phase0.Slot) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionHeze,
		Heze:    &heze.SignedBeaconBlock{Message: &heze.BeaconBlock{Slot: slot}},
	}
}

func TestGetEnvelopeRequests(t *testing.T) {
	requests := &gloas.ExecutionRequests{
		BuilderDeposits: []*gloas.BuilderDepositRequest{testBuilderDepositRequest(1)},
	}

	tests := []struct {
		name         string
		block        *spec.VersionedSignedBeaconBlock
		provider     *fakeEnvelopeProvider
		wantErr      bool
		wantEmpty    bool
		wantBlockIDs []string
	}{
		{
			name:      "before gloas there are no builder requests and no envelope fetch",
			block:     electraBlockWithRequests(spec.DataVersionElectra, nil),
			provider:  &fakeEnvelopeProvider{},
			wantEmpty: true,
		},
		{
			name:         "gloas returns the envelope requests with the payload identity",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{envelope: envelopeWithPayload(requests)},
			wantBlockIDs: []string{"123"},
		},
		{
			name:         "heze returns the envelope requests",
			block:        hezeBlock(456),
			provider:     &fakeEnvelopeProvider{envelope: envelopeWithPayload(requests)},
			wantBlockIDs: []string{"456"},
		},
		{
			name:         "withheld payload has nothing to report",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{},
			wantBlockIDs: []string{"123"},
			wantEmpty:    true,
		},
		{
			name:         "envelope without requests has nothing to report",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{envelope: envelopeWithPayload(nil)},
			wantBlockIDs: []string{"123"},
			wantEmpty:    true,
		},
		{
			name:         "envelope fetch error is returned",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{err: errors.New("beacon node unavailable")},
			wantBlockIDs: []string{"123"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getEnvelopeRequests(context.Background(), tt.provider, tt.block)

			assert.Equal(t, tt.wantBlockIDs, tt.provider.blockIDs)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Nil(t, got.requests)

				return
			}

			assert.Same(t, requests, got.requests)
			assert.Equal(t, testPayloadBlockNumber, got.blockNumber)
			assert.Equal(t, testPayloadBlockHash, got.blockHash)
		})
	}
}

func TestExecutionRequestBuilderDepositDeriver_DeriveEvents(t *testing.T) {
	deriver := &ExecutionRequestBuilderDepositDeriver{clientMeta: &xatu.ClientMeta{Name: "cannon-test"}}
	identifier := testBlockIdentifier()

	t.Run("emits one event per builder deposit with block context", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithPayload(&gloas.ExecutionRequests{
			// Electra-era requests are not builder requests and must be ignored.
			Deposits:        []*electra.DepositRequest{testDepositRequest(1)},
			BuilderDeposits: []*gloas.BuilderDepositRequest{testBuilderDepositRequest(1_000_000_000), testBuilderDepositRequest(2_000_000_000)},
			BuilderExits:    []*gloas.BuilderExitRequest{testBuilderExitRequest(0x01)},
		})}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		require.Len(t, events, 2)

		for index, event := range events {
			assert.Equal(t, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT, event.GetEvent().GetName())
			assert.Equal(t, "cannon-test", event.GetMeta().GetClient().GetName())

			additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderDeposit()
			require.NotNil(t, additional)
			assert.Equal(t, uint64(index), additional.GetPositionInBlock().GetValue())
			assert.Equal(t, testPayloadBlockNumber, additional.GetBlockNumber().GetValue())
			assert.Equal(t, testPayloadBlockHash, additional.GetBlockHash())
			assert.Same(t, identifier, additional.GetBlock())
		}

		first := events[0].GetEthV2BeaconBlockExecutionRequestBuilderDeposit()
		assert.Equal(t, phase0.BLSPubKey{0x0a}.String(), first.GetPubkey().GetValue())
		assert.Equal(t, "0x0300000000000000000000000000000000000000000000000000000000000000", first.GetWithdrawalCredentials().GetValue())
		assert.Equal(t, uint64(1_000_000_000), first.GetAmount().GetValue())
		assert.Equal(t, phase0.BLSSignature{0x0b}.String(), first.GetSignature().GetValue())

		second := events[1].GetEthV2BeaconBlockExecutionRequestBuilderDeposit()
		assert.Equal(t, uint64(2_000_000_000), second.GetAmount().GetValue())
	})

	t.Run("no builder deposits means no events", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithPayload(&gloas.ExecutionRequests{
			BuilderExits: []*gloas.BuilderExitRequest{testBuilderExitRequest(0x01)},
		})}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("withheld payload emits nothing", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Equal(t, []string{"123"}, provider.blockIDs)
	})

	t.Run("pre gloas block emits nothing without fetching an envelope", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{}

		events, err := deriver.deriveEvents(context.Background(), provider, electraBlockWithRequests(spec.DataVersionFulu, nil), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, provider.blockIDs)
	})

	t.Run("envelope fetch error is returned", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{err: errors.New("beacon node unavailable")}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.Error(t, err)
		assert.Nil(t, events)
	})
}

func TestExecutionRequestBuilderExitDeriver_DeriveEvents(t *testing.T) {
	deriver := &ExecutionRequestBuilderExitDeriver{clientMeta: &xatu.ClientMeta{Name: "cannon-test"}}
	identifier := testBlockIdentifier()

	t.Run("emits one event per builder exit with block context", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithPayload(&gloas.ExecutionRequests{
			BuilderDeposits: []*gloas.BuilderDepositRequest{testBuilderDepositRequest(1)},
			BuilderExits:    []*gloas.BuilderExitRequest{testBuilderExitRequest(0x01), testBuilderExitRequest(0x02)},
		})}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		require.Len(t, events, 2)

		for index, event := range events {
			assert.Equal(t, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT, event.GetEvent().GetName())
			assert.Equal(t, "cannon-test", event.GetMeta().GetClient().GetName())

			additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderExit()
			require.NotNil(t, additional)
			assert.Equal(t, uint64(index), additional.GetPositionInBlock().GetValue())
			assert.Equal(t, testPayloadBlockNumber, additional.GetBlockNumber().GetValue())
			assert.Equal(t, testPayloadBlockHash, additional.GetBlockHash())
			assert.Same(t, identifier, additional.GetBlock())
		}

		first := events[0].GetEthV2BeaconBlockExecutionRequestBuilderExit()
		assert.Equal(t, bellatrix.ExecutionAddress{0x01}.String(), first.GetSourceAddress().GetValue())
		assert.Equal(t, phase0.BLSPubKey{0x0c}.String(), first.GetPubkey().GetValue())

		second := events[1].GetEthV2BeaconBlockExecutionRequestBuilderExit()
		assert.Equal(t, bellatrix.ExecutionAddress{0x02}.String(), second.GetSourceAddress().GetValue())
	})

	t.Run("no builder exits means no events", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithPayload(&gloas.ExecutionRequests{
			BuilderDeposits: []*gloas.BuilderDepositRequest{testBuilderDepositRequest(1)},
		})}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("withheld payload emits nothing", func(t *testing.T) {
		events, err := deriver.deriveEvents(context.Background(), &fakeEnvelopeProvider{}, gloasBlock(123), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("pre gloas block emits nothing without fetching an envelope", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{}

		events, err := deriver.deriveEvents(context.Background(), provider, electraBlockWithRequests(spec.DataVersionElectra, nil), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, provider.blockIDs)
	})

	t.Run("envelope fetch error is returned", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{err: errors.New("beacon node unavailable")}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.Error(t, err)
		assert.Nil(t, events)
	})
}

func TestBuilderDerivers_ActivateAtGloas(t *testing.T) {
	assert.Equal(t, spec.DataVersionGloas, (&ExecutionRequestBuilderDepositDeriver{}).ActivationFork())
	assert.Equal(t, spec.DataVersionGloas, (&ExecutionRequestBuilderExitDeriver{}).ActivationFork())
	assert.Equal(t, spec.DataVersionGloas, (&BlockAccessListSummaryDeriver{}).ActivationFork())

	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT, (&ExecutionRequestBuilderDepositDeriver{}).CannonType())
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT, (&ExecutionRequestBuilderExitDeriver{}).CannonType())
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY, (&BlockAccessListSummaryDeriver{}).CannonType())
}
