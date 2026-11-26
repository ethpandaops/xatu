package ethereum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/go-eth2-client/spec/pq"
	"github.com/pkg/errors"
)

// go-eth2-client can't decode PQ proposer duties or states: it rejects any public key that is not 48 bytes, and its
// PQ BeaconState JSON rejects Lighthouse's ptc_window. From the PQ fork on, both are read from the beacon API here.

// ProposerDuty is one slot's proposer. PubKey is the hex public key as the beacon node reports it: 48 bytes of BLS
// key, or 32 bytes of XMSS key on PQ.
type ProposerDuty struct {
	Slot           phase0.Slot
	ValidatorIndex phase0.ValidatorIndex
	PubKey         string
}

// FetchProposerDuties returns epoch's proposer duties.
func (b *BeaconNode) FetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*ProposerDuty, error) {
	isPQ, err := b.isPQ(ctx, epoch)
	if err != nil {
		return nil, err
	}

	if !isPQ {
		libDuties, err := b.beacon.FetchProposerDuties(ctx, epoch)
		if err != nil {
			return nil, err
		}

		duties := make([]*ProposerDuty, 0, len(libDuties))
		for _, d := range libDuties {
			duties = append(duties, &ProposerDuty{
				Slot:           d.Slot,
				ValidatorIndex: d.ValidatorIndex,
				PubKey:         fmt.Sprintf("0x%s", hex.EncodeToString(d.PubKey[:])),
			})
		}

		return duties, nil
	}

	var rsp struct {
		Data []struct {
			PubKey         string `json:"pubkey"`
			ValidatorIndex string `json:"validator_index"` //nolint:tagliatelle // beacon API field
			Slot           string `json:"slot"`
		} `json:"data"`
	}

	if err := b.getBeaconJSON(ctx, fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch), &rsp); err != nil {
		return nil, errors.Wrap(err, "failed to fetch proposer duties")
	}

	duties := make([]*ProposerDuty, 0, len(rsp.Data))

	for _, d := range rsp.Data {
		slot, err := strconv.ParseUint(d.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid proposer duty slot %q", d.Slot)
		}

		index, err := strconv.ParseUint(d.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid proposer duty validator index %q", d.ValidatorIndex)
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

// fetchPQValidators returns the validators of a PQ state, their XMSS keys widened the way go-eth2-client widens them
// (pq.XMSSPubKey.DepositPubKey).
func (b *BeaconNode) fetchPQValidators(ctx context.Context, stateID string) (map[phase0.ValidatorIndex]*apiv1.Validator, error) {
	var rsp struct {
		Data []struct {
			Index     string               `json:"index"`
			Balance   string               `json:"balance"`
			Status    apiv1.ValidatorState `json:"status"`
			Validator *pq.Validator        `json:"validator"`
		} `json:"data"`
	}

	if err := b.getBeaconJSON(ctx, fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID), &rsp); err != nil {
		return nil, errors.Wrap(err, "failed to fetch validators")
	}

	validators := make(map[phase0.ValidatorIndex]*apiv1.Validator, len(rsp.Data))

	for _, v := range rsp.Data {
		index, err := strconv.ParseUint(v.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid validator index %q", v.Index)
		}

		balance, err := strconv.ParseUint(v.Balance, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid balance %q of validator %d", v.Balance, index)
		}

		if v.Validator == nil {
			return nil, fmt.Errorf("validator %d has no data", index)
		}

		validators[phase0.ValidatorIndex(index)] = &apiv1.Validator{
			Index:     phase0.ValidatorIndex(index),
			Balance:   phase0.Gwei(balance),
			Status:    v.Status,
			Validator: v.Validator.Legacy(),
		}
	}

	return validators, nil
}

// isPQ reports whether epoch is at or after the network's PQ fork.
func (b *BeaconNode) isPQ(ctx context.Context, epoch phase0.Epoch) (bool, error) {
	provider, ok := b.beacon.Service().(client.SpecProvider)
	if !ok {
		return false, errors.New("spec provider not available")
	}

	specs, err := provider.Spec(ctx, &api.SpecOpts{})
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch spec")
	}

	forkEpoch, ok := specs.Data["PQ_FORK_EPOCH"].(uint64)
	if !ok {
		return false, nil
	}

	return uint64(epoch) >= forkEpoch, nil
}

// getBeaconJSON GETs path from the beacon node and decodes its JSON body into out.
func (b *BeaconNode) getBeaconJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(b.config.Beacon.Address, "/")+path, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	for k, v := range b.config.Beacon.Headers {
		req.Header.Set(k, v)
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))

		return fmt.Errorf("GET %s: status %d: %s", path, rsp.StatusCode, body)
	}

	return json.NewDecoder(rsp.Body).Decode(out)
}
