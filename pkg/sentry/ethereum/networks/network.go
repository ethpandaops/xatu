package networks

import "github.com/attestantio/go-eth2-client/spec/phase0"

type NetworkName string

var (
	NetworkNameNone    NetworkName = "none"
	NetworkNameUnknown NetworkName = "unknown"
	NetworkNameMainnet NetworkName = "mainnet"
	NetworkNameSepolia NetworkName = "sepolia"
	NetworkNameGoerli  NetworkName = "goerli"
)

type NetworkIdentifierSlot struct {
	NetworkName NetworkName
	Slot        phase0.Slot
	BlockRoot   string
}

var NetworkSlotBlockRoots = []NetworkIdentifierSlot{
	{
		NetworkName: NetworkNameMainnet,
		Slot:        phase0.Slot(50),
		BlockRoot:   "0x68937f266e8f339e3d605b00424446f8db835a4f2548636a906095babc5fb308",
	},
	{
		NetworkName: NetworkNameSepolia,
		Slot:        phase0.Slot(50),
		BlockRoot:   "0x79bed901a63e22bb4dbed9330372bc399ea4d5faec87de916a80410135f3475c",
	},
	{
		NetworkName: NetworkNameGoerli,
		Slot:        phase0.Slot(50),
		BlockRoot:   "0x93379731ee4c8e438a2c74c850e80fa7b2ccab4d9e25e6dc5567d3a33059b4d4",
	},
}

func AllNetworkIdentifierSlots() []phase0.Slot {
	// Deduplicate the slots.
	slots := map[phase0.Slot]struct{}{}

	for _, networkSlotBlockRoot := range NetworkSlotBlockRoots {
		slots[networkSlotBlockRoot.Slot] = struct{}{}
	}

	slotsSlice := make([]phase0.Slot, 0, len(slots))
	for slot := range slots {
		slotsSlice = append(slotsSlice, slot)
	}

	return slotsSlice
}

func DeriveNetworkName(slot phase0.Slot, blockRoot string) NetworkName {
	for _, networkSlotBlockRoot := range NetworkSlotBlockRoots {
		if networkSlotBlockRoot.Slot == slot && networkSlotBlockRoot.BlockRoot == blockRoot {
			return networkSlotBlockRoot.NetworkName
		}
	}

	return NetworkNameUnknown
}
