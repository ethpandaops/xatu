package iterator

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type DefaultSlotStartingPositions map[xatu.CannonType]phase0.Slot

var (
	GoerliDefaultSlotStartingPosition = DefaultSlotStartingPositions{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:          phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:                 phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: phase0.Slot(162304 * 32), // Capella fork
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   phase0.Slot(112260 * 32), // Bellatrix fork
	}

	MainnetDefaultSlotStartingPosition = DefaultSlotStartingPositions{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:          phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:                 phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: phase0.Slot(194048 * 32), // Capella fork
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   phase0.Slot(144896 * 32), // Bellatrix fork
	}

	SepoliaDefaultSlotStartingPosition = DefaultSlotStartingPositions{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:          phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:                 phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: phase0.Slot(56832 * 32), // Capella fork
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   phase0.Slot(100 * 32),   // Bellatrix fork
	}

	HoleskyDefaultSlotStartingPosition = DefaultSlotStartingPositions{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:       phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:          phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:                 phase0.Slot(0),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: phase0.Slot(256 * 32), // Capella fork
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   phase0.Slot(0),
	}
)
