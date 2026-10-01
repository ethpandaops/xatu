package v2

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// devnetBAL is a block access list captured from a Gloas devnet beacon node:
// four accounts, three storage writes over three slots and eight storage reads.
const devnetBAL = "f8d1de9400000961ef480eb55e80d19ad83579a64c007002c0c480010203c0c0c0" +
	"de940000bbddc7ce488642fb579f8b00f3a590007251c0c480010203c0c0c0" +
	"f841940000f90827f1c53a10cb7a02335b175320002935e7e681c7e3e280a02da04ac41c52c5ed22ac6e519822f2a3b5e852d835cab242548ba7cd531b8519c0c0c0c0" +
	"f84e94000f3df6d732807ef1319fb7b8bb8522d0beac02f4cb821316c7c6808469c1e4ede7823315e3e280a03879f24e13e88a5db134e7092a18829d88875fc5c4252d3f27b21b7a96fd2e8bc0c0c0c0"

func envelopeWithBAL(raw []byte) *gloas.SignedExecutionPayloadEnvelope {
	return &gloas.SignedExecutionPayloadEnvelope{
		Message: &gloas.ExecutionPayloadEnvelope{
			Payload: &gloas.ExecutionPayload{
				BlockNumber:     testPayloadBlockNumber,
				BlockHash:       phase0.Hash32{31: 0xcd},
				BlockAccessList: raw,
			},
		},
	}
}

func TestBlockAccessListSummaryDeriver_DeriveEvents(t *testing.T) {
	deriver := &BlockAccessListSummaryDeriver{clientMeta: &xatu.ClientMeta{Name: "cannon-test"}}
	identifier := testBlockIdentifier()

	raw, err := hex.DecodeString(devnetBAL)
	require.NoError(t, err)

	t.Run("emits one summary event per block", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithBAL(raw)}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, []string{"123"}, provider.blockIDs)

		event := events[0]
		assert.Equal(t, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY, event.GetEvent().GetName())
		assert.Equal(t, "cannon-test", event.GetMeta().GetClient().GetName())

		summary := event.GetEthV2BeaconBlockAccessListSummary()
		require.NotNil(t, summary)
		assert.Equal(t, uint32(4), summary.GetAccountsTouched().GetValue())
		assert.Equal(t, uint32(3), summary.GetStorageSlotsChanged().GetValue())
		assert.Equal(t, uint32(3), summary.GetStorageChanges().GetValue())
		assert.Equal(t, uint32(8), summary.GetStorageReads().GetValue())
		assert.Equal(t, uint32(0), summary.GetBalanceChanges().GetValue())
		assert.Equal(t, uint32(0), summary.GetNonceChanges().GetValue())
		assert.Equal(t, uint32(0), summary.GetCodeChanges().GetValue())
		assert.Equal(t, uint32(3), summary.GetTotalChanges().GetValue())
		assert.Equal(t, uint32(len(raw)), summary.GetBalSizeBytes().GetValue())
		assert.Equal(t, "0x77d73a8c93424f86fec3b88a664fe5a13134e1a017a2d5d96eac421773f4baa0", summary.GetBalHash().GetValue())

		additional := event.GetMeta().GetClient().GetEthV2BeaconBlockAccessListSummary()
		require.NotNil(t, additional)
		assert.Equal(t, testPayloadBlockNumber, additional.GetBlockNumber().GetValue())
		assert.Equal(t, testPayloadBlockHash, additional.GetBlockHash())
		assert.Same(t, identifier, additional.GetBlock())
	})

	t.Run("heze block is summarised too", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithBAL(raw)}

		events, err := deriver.deriveEvents(context.Background(), provider, hezeBlock(456), identifier)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, []string{"456"}, provider.blockIDs)
	})

	t.Run("withheld payload emits nothing", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Equal(t, []string{"123"}, provider.blockIDs)
	})

	t.Run("payload without an access list emits nothing", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithBAL(nil)}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("an empty access list still has a summary row", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithBAL([]byte{0xc0})}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, uint32(0), events[0].GetEthV2BeaconBlockAccessListSummary().GetAccountsTouched().GetValue())
		assert.Equal(t, uint32(1), events[0].GetEthV2BeaconBlockAccessListSummary().GetBalSizeBytes().GetValue())
	})

	t.Run("an undecodable access list is an error rather than a zeroed row", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{envelope: envelopeWithBAL([]byte{0xff, 0xfe, 0xfd})}

		events, err := deriver.deriveEvents(context.Background(), provider, gloasBlock(123), identifier)
		require.Error(t, err)
		assert.Nil(t, events)
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
