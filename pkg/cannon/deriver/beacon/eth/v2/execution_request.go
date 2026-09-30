package v2

import (
	"context"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/pkg/errors"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
)

type executionPayloadEnvelopeProvider interface {
	GetExecutionPayloadEnvelope(ctx context.Context, blockID string) (*gloas.SignedExecutionPayloadEnvelope, error)
}

// getExecutionRequests returns the execution requests produced by the block's
// execution payload. From Gloas (EIP-7732) they live in the separately fetched
// payload envelope, and a withheld payload produced none.
func getExecutionRequests(
	ctx context.Context,
	envelopes executionPayloadEnvelopeProvider,
	block *spec.VersionedSignedBeaconBlock,
) (*spec.VersionedExecutionRequests, error) {
	requests := &spec.VersionedExecutionRequests{Version: block.Version}

	if block.Version < spec.DataVersionElectra {
		return requests, nil
	}

	if block.Version < spec.DataVersionGloas {
		bodyRequests, err := block.ExecutionRequests()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain execution requests")
		}

		return bodyRequests, nil
	}

	slot, err := block.Slot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain block slot")
	}

	envelope, err := envelopes.GetExecutionPayloadEnvelope(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get execution payload envelope for slot %d", slot)
	}

	if envelope == nil || envelope.Message == nil || envelope.Message.ExecutionRequests == nil {
		return requests, nil
	}

	switch block.Version {
	case spec.DataVersionGloas:
		requests.Gloas = envelope.Message.ExecutionRequests
	case spec.DataVersionHeze:
		requests.Heze = envelope.Message.ExecutionRequests
	default:
		return nil, errors.Errorf("unsupported block version %s for execution requests", block.Version)
	}

	return requests, nil
}
