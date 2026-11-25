package eth

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/pq"
	"github.com/stretchr/testify/require"

	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
)

// The fixture is a pq-devnet-0 block (slot 10617) with its proof shortened and
// a proposer slashing, an attester slashing and PQ execution requests added.
func TestNewEventBlockV2FromVersionSignedBeaconBlockPQ(t *testing.T) {
	raw, err := os.ReadFile("testdata/pq_signed_beacon_block.json")
	require.NoError(t, err)

	var block pq.SignedBeaconBlock
	require.NoError(t, json.Unmarshal(raw, &block))

	//nolint:tagliatelle // JSON tags match the beacon API.
	var want struct {
		Message struct {
			Body struct {
				RandaoReveal      []string `json:"randao_reveal"`
				Proof             string   `json:"proof"`
				ProposerSlashings []struct {
					SignedHeader2 struct {
						Message struct {
							BodyRoot string `json:"body_root"`
						} `json:"message"`
						Signature string `json:"signature"`
					} `json:"signed_header_2"`
				} `json:"proposer_slashings"`
				Attestations []struct {
					AggregationBits string `json:"aggregation_bits"`
				} `json:"attestations"`
				Bid struct {
					Message struct {
						BlockHash string `json:"block_hash"`
					} `json:"message"`
					Signature string `json:"signature"`
				} `json:"signed_execution_payload_bid"`
			} `json:"body"`
		} `json:"message"`
		Signature string `json:"signature"`
	}
	require.NoError(t, json.Unmarshal(raw, &want))

	event, err := NewEventBlockV2FromVersionSignedBeaconBlock(&spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionPQ,
		PQ:      &block,
	})
	require.NoError(t, err)

	require.Equal(t, v2.BlockVersion_PQ, event.GetVersion())
	require.Equal(t, want.Signature, event.GetSignature())
	require.Len(t, event.GetSignature(), 2+2*pq.XMSSSignatureSize)

	msg := event.GetPqBlock()
	require.NotNil(t, msg)
	require.Equal(t, uint64(10617), msg.GetSlot().GetValue())
	require.Equal(t, uint64(6961), msg.GetProposerIndex().GetValue())
	require.Equal(t, "0x38808cfb07e862c9f08120bf52ac8e0eaccfb654ae46bf638d9d006dd072e8b8", msg.GetStateRoot())

	body := msg.GetBody()
	require.Equal(t, "0x6e696d6275732d676574682d3200000000000000000000000000000000000000", body.GetGraffiti())

	reveal := "0x"
	for _, root := range want.Message.Body.RandaoReveal {
		reveal += strings.TrimPrefix(root, "0x")
	}

	require.Equal(t, reveal, body.GetRandaoReveal())
	require.Equal(t, want.Message.Body.Proof, body.GetProof())

	require.Len(t, body.GetAttestations(), 1)
	require.Equal(t, want.Message.Body.Attestations[0].AggregationBits, body.GetAttestations()[0].GetAggregationBits())
	require.Empty(t, body.GetAttestations()[0].GetSignature())
	require.Equal(t, uint64(10615), body.GetAttestations()[0].GetData().GetSlot())

	require.Len(t, body.GetProposerSlashings(), 1)
	header := body.GetProposerSlashings()[0].GetSignedHeader_2()
	require.Equal(t, want.Message.Body.ProposerSlashings[0].SignedHeader2.Message.BodyRoot, header.GetMessage().GetBodyRoot())
	require.Equal(t, want.Message.Body.ProposerSlashings[0].SignedHeader2.Signature, header.GetSignature())

	require.Len(t, body.GetAttesterSlashings(), 1)
	require.Equal(t, []uint64{3, 9}, body.GetAttesterSlashings()[0].GetAttestation_1().GetAttestingIndices())
	require.Equal(t, "0x0102", body.GetAttesterSlashings()[0].GetAttestation_1().GetProof())
	require.Equal(t, "0x0304", body.GetAttesterSlashings()[0].GetAttestation_2().GetProof())

	bid := body.GetSignedExecutionPayloadBid()
	require.Equal(t, want.Message.Body.Bid.Message.BlockHash, bid.GetMessage().GetBlockHash())
	require.Equal(t, uint64(18446744073709551615), bid.GetMessage().GetBuilderIndex().GetValue())
	require.Equal(t, want.Message.Body.Bid.Signature, bid.GetSignature())

	requests := body.GetParentExecutionRequests()
	require.Len(t, requests.GetValidatorPreregistrations(), 1)
	require.Equal(t, "0x"+strings.Repeat("11", 32), requests.GetValidatorPreregistrations()[0].GetPubkey().GetValue())
	require.Equal(t, "0x"+strings.Repeat("33", 32), requests.GetValidatorPreregistrations()[0].GetVrfRoot().GetValue())
	require.Len(t, requests.GetBuilderPreregistrations(), 1)
	require.Equal(t, "0x"+strings.Repeat("55", 32), requests.GetBuilderPreregistrations()[0].GetWithdrawalCredentials().GetValue())
	require.Len(t, requests.GetVoluntaryExits(), 1)
	require.Equal(t, "0x"+strings.Repeat("66", 32), requests.GetVoluntaryExits()[0].GetPubkey().GetValue())
}
