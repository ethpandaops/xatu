package iterator

import (
	"context"
	"errors"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Iterator interface {
	Start(ctx context.Context) error
	UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error
	Next(ctx context.Context) (next *xatu.CannonLocation, lookAhead []*xatu.CannonLocation, err error)
}

// Ensure that derivers implements the EventDeriver interface
var _ Iterator = &CheckpointIterator{}
var _ Iterator = &BackfillingCheckpoint{}

var (
	ErrLocationUpToDate = errors.New("location up to date")

	SlotZero = phase0.Slot(0)
)

func GetDefaultSlotLocation(forkEpochs state.ForkEpochs, cannonType xatu.CannonType) phase0.Slot {
	defaults := NewSlotDefaultsFromForkEpochs(forkEpochs)
	if slot, exists := defaults[cannonType]; exists {
		return slot
	}

	return SlotZero
}
