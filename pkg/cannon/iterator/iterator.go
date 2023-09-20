package iterator

import (
	"context"
	"errors"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Iterator interface {
	UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error
	Next(ctx context.Context) (next *xatu.CannonLocation, lookAhead []*xatu.CannonLocation, err error)
}

// Ensure that derivers implements the EventDeriver interface
var _ Iterator = &CheckpointIterator{}
var _ Iterator = &SlotIterator{}

var (
	ErrLocationUpToDate = errors.New("location up to date")

	SlotZero = phase0.Slot(0)
)

func GetDefaultSlotLocationForNetworkAndType(network string, cannonType xatu.CannonType) phase0.Slot {
	switch network {
	case "goerli", "prater":
		return getSlotFromNetworkAndType(cannonType, GoerliDefaultSlotStartingPosition)
	case "mainnet":
		return getSlotFromNetworkAndType(cannonType, MainnetDefaultSlotStartingPosition)
	case "sepolia":
		return getSlotFromNetworkAndType(cannonType, SepoliaDefaultSlotStartingPosition)
	case "holesky":
		return getSlotFromNetworkAndType(cannonType, HoleskyDefaultSlotStartingPosition)
	default:
		return SlotZero
	}
}

func getSlotFromNetworkAndType(typ xatu.CannonType, networkDefaults DefaultSlotStartingPositions) phase0.Slot {
	if networkDefaults == nil {
		return SlotZero
	}

	if slot, exists := networkDefaults[typ]; exists {
		return slot
	}

	return SlotZero
}
