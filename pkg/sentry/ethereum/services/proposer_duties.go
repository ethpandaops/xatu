package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// go-eth2-client rejects proposer duties whose public key is not 48 bytes, and PQ networks report 32-byte XMSS keys,
// so duties are read from the beacon API directly.

const proposerDutiesTimeout = 30 * time.Second

// ProposerDuty is one slot's proposer. PubKey is the hex public key as the beacon node reports it: 48 bytes of BLS
// key, or 32 bytes of XMSS key on PQ networks.
type ProposerDuty struct {
	Slot           phase0.Slot
	ValidatorIndex phase0.ValidatorIndex
	PubKey         string
}

type beaconAPI struct {
	address string
	headers map[string]string
}

func (b beaconAPI) fetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*ProposerDuty, error) {
	ctx, cancel := context.WithTimeout(ctx, proposerDutiesTimeout)
	defer cancel()

	path := fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(b.address, "/")+path, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	for k, v := range b.headers {
		req.Header.Set(k, v)
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))

		return nil, fmt.Errorf("GET %s: status %d: %s", path, rsp.StatusCode, body)
	}

	var body struct {
		Data []struct {
			PubKey         string `json:"pubkey"`
			ValidatorIndex string `json:"validator_index"` //nolint:tagliatelle // beacon API field
			Slot           string `json:"slot"`
		} `json:"data"`
	}

	if err := json.NewDecoder(rsp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode proposer duties: %w", err)
	}

	duties := make([]*ProposerDuty, 0, len(body.Data))

	for _, d := range body.Data {
		slot, err := strconv.ParseUint(d.Slot, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid proposer duty slot %q: %w", d.Slot, err)
		}

		index, err := strconv.ParseUint(d.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid proposer duty validator index %q: %w", d.ValidatorIndex, err)
		}

		if d.PubKey == "" {
			return nil, fmt.Errorf("proposer duty for slot %d has no public key", slot)
		}

		duties = append(duties, &ProposerDuty{
			Slot:           phase0.Slot(slot),
			ValidatorIndex: phase0.ValidatorIndex(index),
			PubKey:         strings.ToLower(d.PubKey),
		})
	}

	return duties, nil
}
