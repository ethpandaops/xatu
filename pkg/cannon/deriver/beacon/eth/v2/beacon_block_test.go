package v2

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/golang/snappy"
	"github.com/holiman/uint256"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/eth"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// loadEraBlock decodes testdata/blocks/<name>.ssz.snappy, a block entry copied from a Nimbus era file.
func loadEraBlock(t *testing.T, name string, version spec.DataVersion) *spec.VersionedSignedBeaconBlock {
	t.Helper()

	framed, err := os.ReadFile(filepath.Join("testdata", "blocks", name+".ssz.snappy"))
	require.NoError(t, err)

	data, err := io.ReadAll(snappy.NewReader(bytes.NewReader(framed)))
	require.NoError(t, err)

	block := &spec.VersionedSignedBeaconBlock{Version: version}

	var target any

	switch version {
	case spec.DataVersionPhase0:
		block.Phase0 = &phase0.SignedBeaconBlock{}
		target = block.Phase0
	case spec.DataVersionAltair:
		block.Altair = &altair.SignedBeaconBlock{}
		target = block.Altair
	case spec.DataVersionBellatrix:
		block.Bellatrix = &bellatrix.SignedBeaconBlock{}
		target = block.Bellatrix
	case spec.DataVersionCapella:
		block.Capella = &capella.SignedBeaconBlock{}
		target = block.Capella
	case spec.DataVersionFulu:
		block.Fulu = &electra.SignedBeaconBlock{}
		target = block.Fulu
	case spec.DataVersionGloas:
		block.Gloas = &gloas.SignedBeaconBlock{}
		target = block.Gloas
	default:
		t.Fatalf("no fixture decoder for %s", version)
	}

	require.NoError(t, dynssz.GetGlobalDynSsz().UnmarshalSSZ(target, data))

	return block
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func TestSetBlockSizes(t *testing.T) {
	// Expected values are the canonical_beacon_block rows the cannon derived for these slots from the beacon node.
	tests := []struct {
		fixture                       string
		version                       spec.DataVersion
		root                          string
		total, totalCompressed        uint64
		txCount, txBytes, txCompBytes *uint64
	}{
		{
			fixture: "mainnet-1", version: spec.DataVersionPhase0,
			root:  "0xbacd20f09da907734434f052bd4c9503aa16bab1960e89ea20610d08d064481c",
			total: 304, totalCompressed: 279,
		},
		{
			fixture: "mainnet-4636671", version: spec.DataVersionAltair,
			root:  "0xe7d4f3b7924c30ae047fceabb853b8afdae32b85e0a87ab6c4c37421b353a1da",
			total: 33488, totalCompressed: 16720,
		},
		{
			fixture: "mainnet-4636672", version: spec.DataVersionBellatrix,
			root:  "0x9429ce339da8944dd2e1565be8cac5bf634cae2120b6937c081e39148a7f4b1a",
			total: 34000, totalCompressed: 16389,
			txCount: uint64Ptr(0), txBytes: uint64Ptr(0), txCompBytes: uint64Ptr(1),
		},
		{
			fixture: "mainnet-4700013", version: spec.DataVersionBellatrix,
			root:  "0x810a00400a80cdffc11ffdcf17ac404ac4dba215b95221955a9dfddf163d0b0d",
			total: 52332, totalCompressed: 30787,
			txCount: uint64Ptr(80), txBytes: uint64Ptr(17884), txCompBytes: uint64Ptr(12808),
		},
		{
			// Gloas transaction stats come from the payload envelope, not the body.
			fixture: "sepolia-11374000", version: spec.DataVersionGloas,
			root:  "0x1a4161ae68d3aa561f4463754b17b53523972882f5ac745d50ecb57e1e01e7b6",
			total: 1802, totalCompressed: 1482,
		},
	}

	optional := func(t *testing.T, field string, want *uint64, got interface{ GetValue() uint64 }, isNil bool) {
		t.Helper()

		if want == nil {
			assert.True(t, isNil, "%s is unset without an execution payload", field)

			return
		}

		require.False(t, isNil, "%s is set", field)
		assert.Equal(t, *want, got.GetValue(), field)
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s %s", tc.version, tc.fixture), func(t *testing.T) {
			block := loadEraBlock(t, tc.fixture, tc.version)

			extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}
			require.NoError(t, setBlockSizes(extra, block))

			assert.Equal(t, tc.root, extra.GetBlockRoot())
			assert.Equal(t, tc.total, extra.GetTotalBytes().GetValue())
			assert.Equal(t, tc.totalCompressed, extra.GetTotalBytesCompressed().GetValue())
			optional(t, "transactions count", tc.txCount, extra.GetTransactionsCount(), extra.GetTransactionsCount() == nil)
			optional(t, "transactions bytes", tc.txBytes, extra.GetTransactionsTotalBytes(), extra.GetTransactionsTotalBytes() == nil)
			optional(t, "transactions compressed bytes", tc.txCompBytes, extra.GetTransactionsTotalBytesCompressed(),
				extra.GetTransactionsTotalBytesCompressed() == nil)

			event, err := eth.NewEventBlockV2FromVersionSignedBeaconBlock(block)
			require.NoError(t, err)
			assert.NotNil(t, event.GetMessage())
		})
	}
}

func TestSetExecutionPayloadEnvelope(t *testing.T) {
	raw, err := hex.DecodeString(devnetBAL)
	require.NoError(t, err)

	t.Run("revealed payload sets its header, access list hash and transaction stats", func(t *testing.T) {
		envelope := envelopeWithBAL(raw)
		envelope.Message.Payload.GasUsed = 21_000
		envelope.Message.Payload.BaseFeePerGas = uint256.NewInt(7)
		envelope.Message.Payload.SlotNumber = 123
		envelope.Message.Payload.Transactions = []bellatrix.Transaction{{0x01, 0x02}, {0x03}}

		provider := &fakeEnvelopeProvider{envelope: envelope}
		extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

		require.NoError(t, setExecutionPayloadEnvelope(context.Background(), extra, provider, gloasBlock(123)))
		assert.Equal(t, []string{"123"}, provider.blockIDs)

		require.NotNil(t, extra.GetPayloadPresent())
		assert.True(t, extra.GetPayloadPresent().GetValue())

		payload := extra.GetExecutionPayloadEnvelope().GetMessage().GetPayload()
		require.NotNil(t, payload)
		assert.Equal(t, testPayloadBlockNumber, payload.GetBlockNumber().GetValue())
		assert.Equal(t, testPayloadBlockHash, payload.GetBlockHash())
		assert.Equal(t, uint64(21_000), payload.GetGasUsed().GetValue())
		assert.Equal(t, "7", payload.GetBaseFeePerGas())
		assert.Equal(t, uint64(123), payload.GetSlotNumber().GetValue())
		assert.Empty(t, payload.GetTransactions())

		assert.Equal(t, "0x77d73a8c93424f86fec3b88a664fe5a13134e1a017a2d5d96eac421773f4baa0",
			extra.GetExecutionPayloadBlockAccessListHash().GetValue())
		assert.Equal(t, uint64(2), extra.GetTransactionsCount().GetValue())
		assert.Equal(t, uint64(3), extra.GetTransactionsTotalBytes().GetValue())
	})

	t.Run("withheld payload is not present and leaves the payload unset", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{}
		extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

		require.NoError(t, setExecutionPayloadEnvelope(context.Background(), extra, provider, gloasBlock(123)))

		require.NotNil(t, extra.GetPayloadPresent())
		assert.False(t, extra.GetPayloadPresent().GetValue())
		assert.Nil(t, extra.GetExecutionPayloadEnvelope())
		assert.Nil(t, extra.GetExecutionPayloadBlockAccessListHash())
		assert.Nil(t, extra.GetTransactionsCount())
	})

	t.Run("pre gloas block is left alone without fetching an envelope", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{}
		extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

		require.NoError(t, setExecutionPayloadEnvelope(context.Background(), extra, provider,
			electraBlockWithRequests(spec.DataVersionFulu, nil)))
		assert.Empty(t, provider.blockIDs)
		assert.Nil(t, extra.GetPayloadPresent())
	})

	t.Run("envelope fetch error is returned", func(t *testing.T) {
		provider := &fakeEnvelopeProvider{err: errors.New("beacon node unavailable")}
		extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

		require.Error(t, setExecutionPayloadEnvelope(context.Background(), extra, provider, gloasBlock(123)))
	})
}
