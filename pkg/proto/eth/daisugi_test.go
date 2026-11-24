package eth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/daisugi"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestForkAtLeast(t *testing.T) {
	tests := []struct {
		version, fork spec.DataVersion
		want          bool
	}{
		{spec.DataVersionDaisugi, spec.DataVersionDeneb, true},
		{spec.DataVersionDaisugi, spec.DataVersionElectra, true},
		{spec.DataVersionDaisugi, spec.DataVersionFulu, false},
		{spec.DataVersionDaisugi, spec.DataVersionGloas, false},
		{spec.DataVersionDaisugi, spec.DataVersionHeze, false},
		{spec.DataVersionDaisugi, spec.DataVersionDaisugi, true},
		{spec.DataVersionHeze, spec.DataVersionDaisugi, false},
		{spec.DataVersionElectra, spec.DataVersionGloas, false},
		{spec.DataVersionGloas, spec.DataVersionGloas, true},
		{spec.DataVersionHeze, spec.DataVersionGloas, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s>=%s", tt.version, tt.fork), func(t *testing.T) {
			assert.Equal(t, tt.want, ForkAtLeast(tt.version, tt.fork))
		})
	}
}

func TestNewEventBlockV2FromDaisugiBlock(t *testing.T) {
	// daisugi-devnet-0 slot 45660, with the recursive STARK proof cut to 48 bytes.
	data, err := os.ReadFile(filepath.Join("testdata", "daisugi-devnet-0-45660.json"))
	require.NoError(t, err)

	daisugiBlock := &daisugi.SignedBeaconBlock{}
	require.NoError(t, json.Unmarshal(data, daisugiBlock))
	require.Len(t, daisugiBlock.Message.Body.ExecutionPayload.RecursiveStarkProof, 48)

	// The same block without the proof is an Electra block.
	electraBlock := &electra.SignedBeaconBlock{}
	require.NoError(t, json.Unmarshal(data, electraBlock))

	got, err := NewEventBlockV2FromVersionSignedBeaconBlock(&spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionDaisugi,
		Daisugi: daisugiBlock,
	})
	require.NoError(t, err)

	want, err := NewEventBlockV2FromVersionSignedBeaconBlock(&spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionElectra,
		Electra: electraBlock,
	})
	require.NoError(t, err)

	assert.True(t, proto.Equal(want, got), "daisugi block converts to its electra block\nwant: %v\ngot:  %v", want, got)

	payload := got.GetElectraBlock().GetBody().GetExecutionPayload()
	assert.Equal(t, uint64(45659), payload.GetBlockNumber().GetValue())
	assert.Len(t, payload.GetTransactions(), 5)
	assert.Len(t, payload.GetWithdrawals(), 16)
	assert.Equal(t, daisugiBlock.Signature.String(), got.GetSignature())
}
