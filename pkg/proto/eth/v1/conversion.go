package v1

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func RootAsString(root phase0.Root) string {
	return fmt.Sprintf("%#x", root)
}

func SlotAsString(slot phase0.Slot) string {
	return fmt.Sprintf("%d", slot)
}

func EpochAsString(epoch phase0.Epoch) string {
	return fmt.Sprintf("%d", epoch)
}

func BLSSignatureToString(s phase0.BLSSignature) string {
	return fmt.Sprintf("%#x", s)
}

func BytesToString(b []byte) string {
	return fmt.Sprintf("%#x", b)
}
