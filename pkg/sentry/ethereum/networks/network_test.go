package networks

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestDeriveNetwork(t *testing.T) {
	tests := []struct {
		slot uint64
		root string
		want NetworkName
	}{
		{
			slot: 50,
			root: "0x68937f266e8f339e3d605b00424446f8db835a4f2548636a906095babc5fb308",
			want: NetworkNameMainnet,
		},
		{
			slot: 50,
			root: "0x79bed901a63e22bb4dbed9330372bc399ea4d5faec87de916a80410135f3475c",
			want: NetworkNameSepolia,
		},
		{
			slot: 50,
			root: "0x93379731ee4c8e438a2c74c850e80fa7b2ccab4d9e25e6dc5567d3a33059b4d4",
			want: NetworkNameGoerli,
		},
	}

	for _, test := range tests {
		if got := DeriveNetworkName(phase0.Slot(test.slot), test.root); got != test.want {
			t.Errorf("DeriveNetworkName(%d, %s) = %s, want %s", test.slot, test.root, got, test.want)
		}
	}
}
