package v2

import (
	"context"
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/pkg/errors"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
)

// envelopeRequests are the execution requests carried by a Gloas execution
// payload envelope, together with the identity of the execution block that
// produced them. The zero value means there are no requests to report.
type envelopeRequests struct {
	requests    *gloas.ExecutionRequests
	blockNumber uint64
	blockHash   string
}

// getEnvelopeRequests returns the execution requests of the block's payload
// envelope along with the execution block number and hash. The zero value is
// returned when there is nothing to report: builder requests (EIP-8282) only
// exist from Gloas, and a withheld payload (no envelope) produced no requests.
func getEnvelopeRequests(
	ctx context.Context,
	envelopes executionPayloadEnvelopeProvider,
	block *spec.VersionedSignedBeaconBlock,
) (envelopeRequests, error) {
	if block.Version < spec.DataVersionGloas {
		return envelopeRequests{}, nil
	}

	slot, err := block.Slot()
	if err != nil {
		return envelopeRequests{}, errors.Wrap(err, "failed to obtain block slot")
	}

	envelope, err := envelopes.GetExecutionPayloadEnvelope(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return envelopeRequests{}, errors.Wrapf(err, "failed to get execution payload envelope for slot %d", slot)
	}

	if envelope == nil || envelope.Message == nil || envelope.Message.Payload == nil || envelope.Message.ExecutionRequests == nil {
		return envelopeRequests{}, nil
	}

	return envelopeRequests{
		requests:    envelope.Message.ExecutionRequests,
		blockNumber: envelope.Message.Payload.BlockNumber,
		blockHash:   fmt.Sprintf("%#x", envelope.Message.Payload.BlockHash),
	}, nil
}
