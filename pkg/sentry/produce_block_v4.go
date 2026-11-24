package sentry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// produceBlockV4Body requests a local build: no builders to ask, and a zero
// boost factor so a p2p bid only wins when the local payload is unviable.
var produceBlockV4Body = []byte(`{"min_bid":"0","builder_boost_factor":"0","builders":[]}`)

var produceBlockV4Client = &http.Client{Timeout: 30 * time.Second}

//nolint:tagliatelle // JSON tags match the beacon API.
type produceBlockV4Response struct {
	Version                  string          `json:"version"`
	ConsensusBlockValue      string          `json:"consensus_block_value"`
	ExecutionPayloadValue    string          `json:"execution_payload_value"`
	ExecutionPayloadIncluded bool            `json:"execution_payload_included"`
	Data                     json.RawMessage `json:"data"`
}

//nolint:tagliatelle // JSON tags match the beacon API.
type produceBlockV4Contents struct {
	Block                    *gloas.BeaconBlock              `json:"block"`
	ExecutionPayloadEnvelope *gloas.ExecutionPayloadEnvelope `json:"execution_payload_envelope"`
}

// produceBlockV4 requests a block over POST /eth/v4/validator/blocks, which
// replaces the v3 GET from Gloas on. The envelope is set when the beacon node
// built the payload itself.
func (s *Sentry) produceBlockV4(
	ctx context.Context,
	slot phase0.Slot,
	randaoReveal *phase0.BLSSignature,
) (*api.VersionedProposal, *gloas.ExecutionPayloadEnvelope, error) {
	u, err := url.Parse(s.Config.Ethereum.BeaconNodeAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid beacon node address: %w", err)
	}

	u = u.JoinPath("eth/v4/validator/blocks", strconv.FormatUint(uint64(slot), 10))

	query := u.Query()
	query.Set("randao_reveal", fmt.Sprintf("%#x", randaoReveal[:]))
	query.Set("skip_randao_verification", "")
	query.Set("include_payload", "true")
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(produceBlockV4Body))
	if err != nil {
		return nil, nil, err
	}

	for k, v := range s.Config.Ethereum.BeaconNodeHeaders {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Eth-Consensus-Version", spec.DataVersionGloas.String())

	rsp, err := produceBlockV4Client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to request beacon block proposal v4: %w", err)
	}
	defer rsp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(rsp.Body, 64<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read beacon block proposal v4: %w", err)
	}

	if rsp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("beacon block proposal v4 failed with status %d: %s", rsp.StatusCode, bytes.TrimSpace(body))
	}

	return decodeProduceBlockV4(body)
}

func decodeProduceBlockV4(body []byte) (*api.VersionedProposal, *gloas.ExecutionPayloadEnvelope, error) {
	var rsp produceBlockV4Response
	if err := json.Unmarshal(body, &rsp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode beacon block proposal v4: %w", err)
	}

	if rsp.Version != spec.DataVersionGloas.String() {
		return nil, nil, fmt.Errorf("unsupported beacon block proposal v4 version %q", rsp.Version)
	}

	consensusValue, ok := new(big.Int).SetString(rsp.ConsensusBlockValue, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid consensus block value %q", rsp.ConsensusBlockValue)
	}

	executionValue, ok := new(big.Int).SetString(rsp.ExecutionPayloadValue, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid execution payload value %q", rsp.ExecutionPayloadValue)
	}

	proposal := &api.VersionedProposal{
		Version:        spec.DataVersionGloas,
		ConsensusValue: consensusValue,
		ExecutionValue: executionValue,
	}

	var envelope *gloas.ExecutionPayloadEnvelope

	if rsp.ExecutionPayloadIncluded {
		var contents produceBlockV4Contents
		if err := json.Unmarshal(rsp.Data, &contents); err != nil {
			return nil, nil, fmt.Errorf("failed to decode beacon block contents: %w", err)
		}

		proposal.Gloas = contents.Block
		envelope = contents.ExecutionPayloadEnvelope
	} else {
		proposal.Gloas = &gloas.BeaconBlock{}
		if err := json.Unmarshal(rsp.Data, proposal.Gloas); err != nil {
			return nil, nil, fmt.Errorf("failed to decode beacon block: %w", err)
		}
	}

	if proposal.Gloas == nil {
		return nil, nil, fmt.Errorf("beacon block proposal v4 has no block")
	}

	return proposal, envelope, nil
}
