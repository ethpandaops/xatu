package iterator

import (
	"context"
	"errors"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Iterator interface {
	UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error
	Next(ctx context.Context) (xatu.CannonLocation, error)
}

var (
	ErrLocationUpToDate = errors.New("location up to date")

	SlotZero = phase0.Slot(0)
)

func GetDefaultSlotLocationForNetworkAndType(networkID string, cannonType xatu.CannonType) phase0.Slot {
	switch networkID {
	case "5":
		return getSlotFromNetworkAndType(cannonType, GoerliDefaultSlotStartingPosition)
	case "1":
		return getSlotFromNetworkAndType(cannonType, MainnetDefaultSlotStartingPosition)
	case "11155111":
		return getSlotFromNetworkAndType(cannonType, SepoliaDefaultSlotStartingPosition)
	case "17000":
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
