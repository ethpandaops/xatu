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
)

type fakeEnvelopeProvider struct {
	envelope *gloas.SignedExecutionPayloadEnvelope
	err      error
	blockIDs []string
}

func (f *fakeEnvelopeProvider) GetExecutionPayloadEnvelope(_ context.Context, blockID string) (*gloas.SignedExecutionPayloadEnvelope, error) {
	f.blockIDs = append(f.blockIDs, blockID)

	return f.envelope, f.err
}

func testDepositRequest(index uint64) *electra.DepositRequest {
	return &electra.DepositRequest{
		Pubkey:                phase0.BLSPubKey{0x01},
		WithdrawalCredentials: make([]byte, 32),
		Amount:                32_000_000_000,
		Signature:             phase0.BLSSignature{0x02},
		Index:                 index,
	}
}

func testWithdrawalRequest(amount phase0.Gwei) *electra.WithdrawalRequest {
	return &electra.WithdrawalRequest{
		SourceAddress:   bellatrix.ExecutionAddress{0x03},
		ValidatorPubkey: phase0.BLSPubKey{0x04},
		Amount:          amount,
	}
}

func testConsolidationRequest() *electra.ConsolidationRequest {
	return &electra.ConsolidationRequest{
		SourceAddress: bellatrix.ExecutionAddress{0x05},
		SourcePubkey:  phase0.BLSPubKey{0x06},
		TargetPubkey:  phase0.BLSPubKey{0x07},
	}
}

func electraBlockWithRequests(version spec.DataVersion, requests *electra.ExecutionRequests) *spec.VersionedSignedBeaconBlock {
	block := &electra.SignedBeaconBlock{
		Message: &electra.BeaconBlock{
			Slot: 100,
			Body: &electra.BeaconBlockBody{ExecutionRequests: requests},
		},
	}

	versioned := &spec.VersionedSignedBeaconBlock{Version: version}
	if version == spec.DataVersionFulu {
		versioned.Fulu = block
	} else {
		versioned.Electra = block
	}

	return versioned
}

func gloasBlock(slot phase0.Slot) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionGloas,
		Gloas: &gloas.SignedBeaconBlock{
			Message: &gloas.BeaconBlock{Slot: slot, Body: &gloas.BeaconBlockBody{}},
		},
	}
}

func envelopeWithRequests(requests *gloas.ExecutionRequests) *gloas.SignedExecutionPayloadEnvelope {
	return &gloas.SignedExecutionPayloadEnvelope{
		Message: &gloas.ExecutionPayloadEnvelope{
			Payload:           &gloas.ExecutionPayload{},
			ExecutionRequests: requests,
		},
	}
}

func TestGetExecutionRequests(t *testing.T) {
	envelopeRequests := &gloas.ExecutionRequests{
		Deposits:       []*electra.DepositRequest{testDepositRequest(7), testDepositRequest(8)},
		Withdrawals:    []*electra.WithdrawalRequest{testWithdrawalRequest(1)},
		Consolidations: []*electra.ConsolidationRequest{testConsolidationRequest()},
	}

	tests := []struct {
		name               string
		block              *spec.VersionedSignedBeaconBlock
		provider           *fakeEnvelopeProvider
		wantErr            bool
		wantEmpty          bool
		wantBlockIDs       []string
		wantDeposits       int
		wantWithdrawals    int
		wantConsolidations int
	}{
		{
			name:      "deneb has no execution requests",
			block:     &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionDeneb},
			provider:  &fakeEnvelopeProvider{},
			wantEmpty: true,
		},
		{
			name: "electra reads the block body",
			block: electraBlockWithRequests(spec.DataVersionElectra, &electra.ExecutionRequests{
				Deposits: []*electra.DepositRequest{testDepositRequest(1)},
			}),
			provider:     &fakeEnvelopeProvider{},
			wantDeposits: 1,
		},
		{
			name: "fulu reads the block body",
			block: electraBlockWithRequests(spec.DataVersionFulu, &electra.ExecutionRequests{
				Withdrawals: []*electra.WithdrawalRequest{testWithdrawalRequest(1), testWithdrawalRequest(2)},
			}),
			provider:        &fakeEnvelopeProvider{},
			wantWithdrawals: 2,
		},
		{
			name:               "gloas reads the payload envelope",
			block:              gloasBlock(123),
			provider:           &fakeEnvelopeProvider{envelope: envelopeWithRequests(envelopeRequests)},
			wantBlockIDs:       []string{"123"},
			wantDeposits:       2,
			wantWithdrawals:    1,
			wantConsolidations: 1,
		},
		{
			name:         "gloas withheld payload has no requests",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{},
			wantBlockIDs: []string{"123"},
			wantEmpty:    true,
		},
		{
			name:         "gloas envelope without requests",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{envelope: envelopeWithRequests(nil)},
			wantBlockIDs: []string{"123"},
			wantEmpty:    true,
		},
		{
			name:         "gloas envelope fetch error is returned",
			block:        gloasBlock(123),
			provider:     &fakeEnvelopeProvider{err: errors.New("beacon node unavailable")},
			wantBlockIDs: []string{"123"},
			wantErr:      true,
		},
		{
			name: "heze reads the payload envelope",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionHeze,
				Heze:    &heze.SignedBeaconBlock{Message: &heze.BeaconBlock{Slot: 456}},
			},
			provider:           &fakeEnvelopeProvider{envelope: envelopeWithRequests(envelopeRequests)},
			wantBlockIDs:       []string{"456"},
			wantDeposits:       2,
			wantWithdrawals:    1,
			wantConsolidations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests, err := getExecutionRequests(context.Background(), tt.provider, tt.block)

			assert.Equal(t, tt.wantBlockIDs, tt.provider.blockIDs)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, requests)
			assert.Equal(t, tt.block.Version, requests.Version)
			assert.Equal(t, tt.wantEmpty, requests.IsEmpty())

			if tt.wantEmpty {
				return
			}

			deposits, err := requests.Deposits()
			require.NoError(t, err)
			assert.Len(t, deposits, tt.wantDeposits)

			withdrawals, err := requests.Withdrawals()
			require.NoError(t, err)
			assert.Len(t, withdrawals, tt.wantWithdrawals)

			consolidations, err := requests.Consolidations()
			require.NoError(t, err)
			assert.Len(t, consolidations, tt.wantConsolidations)
		})
	}
}

func TestExecutionRequestDerivers_PreGloasBody(t *testing.T) {
	ctx := context.Background()
	block := electraBlockWithRequests(spec.DataVersionElectra, &electra.ExecutionRequests{
		Deposits:       []*electra.DepositRequest{testDepositRequest(9)},
		Withdrawals:    []*electra.WithdrawalRequest{testWithdrawalRequest(5)},
		Consolidations: []*electra.ConsolidationRequest{testConsolidationRequest()},
	})

	deposits, err := (&ExecutionRequestDepositDeriver{}).getDeposits(ctx, block)
	require.NoError(t, err)
	require.Len(t, deposits, 1)
	assert.Equal(t, uint64(9), deposits[0].GetIndex().GetValue())
	assert.Equal(t, uint64(32_000_000_000), deposits[0].GetAmount().GetValue())

	withdrawals, err := (&ExecutionRequestWithdrawalDeriver{}).getExecutionRequestWithdrawals(ctx, block)
	require.NoError(t, err)
	require.Len(t, withdrawals, 1)
	assert.Equal(t, uint64(5), withdrawals[0].GetAmount().GetValue())

	consolidations, err := (&ExecutionRequestConsolidationDeriver{}).getConsolidations(ctx, block)
	require.NoError(t, err)
	require.Len(t, consolidations, 1)
	assert.Equal(t, testConsolidationRequest().TargetPubkey.String(), consolidations[0].GetTargetPubkey().GetValue())
}
