package sentry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
)

func TestDecodeProduceBlockV4_BlockContents(t *testing.T) {
	body, err := os.ReadFile("testdata/produce_block_v4_contents.json")
	require.NoError(t, err)

	proposal, envelope, err := decodeProduceBlockV4(body)
	require.NoError(t, err)

	assert.Equal(t, spec.DataVersionGloas, proposal.Version)
	require.NotNil(t, proposal.Gloas)
	assert.Equal(t, phase0.Slot(11300425), proposal.Gloas.Slot)
	assert.Equal(t, "12345", proposal.ExecutionValue.String())
	assert.Positive(t, proposal.ConsensusValue.Sign())

	slot, err := proposal.Slot()
	require.NoError(t, err)
	assert.Equal(t, phase0.Slot(11300425), slot)

	require.NotNil(t, envelope)
	require.NotNil(t, envelope.Payload)
	assert.Len(t, envelope.Payload.Transactions, 2)
}

func TestDecodeProduceBlockV4_BlockOnly(t *testing.T) {
	body, err := os.ReadFile("testdata/produce_block_v4_contents.json")
	require.NoError(t, err)

	var rsp map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &rsp))

	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rsp["data"], &data))

	rsp["data"] = data["block"]
	rsp["execution_payload_included"] = json.RawMessage(`false`)

	blockOnly, err := json.Marshal(rsp)
	require.NoError(t, err)

	proposal, envelope, err := decodeProduceBlockV4(blockOnly)
	require.NoError(t, err)

	require.NotNil(t, proposal.Gloas)
	assert.Equal(t, phase0.Slot(11300425), proposal.Gloas.Slot)
	assert.Nil(t, envelope)
}

func TestDecodeProduceBlockV4_Errors(t *testing.T) {
	tests := map[string]string{
		"not json":         `nope`,
		"other version":    `{"version":"heze","consensus_block_value":"1","execution_payload_value":"1","data":{}}`,
		"bad value":        `{"version":"gloas","consensus_block_value":"x","execution_payload_value":"1","data":{}}`,
		"missing contents": `{"version":"gloas","consensus_block_value":"1","execution_payload_value":"1","execution_payload_included":true,"data":{}}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := decodeProduceBlockV4([]byte(body))
			assert.Error(t, err)
		})
	}
}

func TestProduceBlockV4_Request(t *testing.T) {
	fixture, err := os.ReadFile("testdata/produce_block_v4_contents.json")
	require.NoError(t, err)

	var (
		gotPath, gotQuery, gotBody string
		gotHeaders                 http.Header
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPath, gotQuery, gotBody, gotHeaders = r.Method+" "+r.URL.Path, r.URL.RawQuery, string(body), r.Header

		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	s := &Sentry{Config: &Config{Ethereum: ethereum.Config{
		BeaconNodeAddress: srv.URL + "/",
		BeaconNodeHeaders: map[string]string{"Authorization": "Bearer token"},
	}}}

	proposal, envelope, err := s.produceBlockV4(context.Background(), 11300425, &infinityRandaoReveal)
	require.NoError(t, err)
	require.NotNil(t, proposal.Gloas)
	require.NotNil(t, envelope)

	assert.Equal(t, "POST /eth/v4/validator/blocks/11300425", gotPath)
	assert.Contains(t, gotQuery, "include_payload=true")
	assert.Contains(t, gotQuery, "skip_randao_verification=")
	assert.Contains(t, gotQuery, "randao_reveal=0xc000")
	assert.JSONEq(t, `{"min_bid":"0","builder_boost_factor":"0","builders":[]}`, gotBody)
	assert.Equal(t, "gloas", gotHeaders.Get("Eth-Consensus-Version"))
	assert.Equal(t, "Bearer token", gotHeaders.Get("Authorization"))
}

func TestProduceBlockV4_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":500,"message":"Engine payload is not available"}`))
	}))
	defer srv.Close()

	s := &Sentry{Config: &Config{Ethereum: ethereum.Config{BeaconNodeAddress: srv.URL}}}

	_, _, err := s.produceBlockV4(context.Background(), 1, &infinityRandaoReveal)
	require.ErrorContains(t, err, "Engine payload is not available")
}
