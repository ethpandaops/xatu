package v1

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
)

func TestTrimmedString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "Empty",
			s:    "",
			want: "",
		},
		{
			name: "Short",
			s:    "abc",
			want: "abc",
		},
		{
			name: "Long",
			s:    "abcdefg",
			want: "abcdefg",
		},
		{
			name: "Longer",
			s:    "abcdefghijk",
			want: "abcdefghijk",
		},
		{
			name: "Longer-trimmed",
			s:    "abcdefghijklmno",
			want: "abcde...klmno",
		},
		{
			name: "hex-trimmed",
			s:    "0xfd3963b996723a6055b3323014c4de94345a7b519b17758b386d6b57a1a16b6d",
			want: "0xfd3...16b6d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimmedString(tt.s); got != tt.want {
				t.Errorf("TrimmedString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBlockAccessListFromGloas_Empty(t *testing.T) {
	result := NewBlockAccessListFromGloas(nil)
	if result == nil {
		t.Fatal("expected non-nil result for nil input")
	}

	if len(result.GetEntries()) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.GetEntries()))
	}
}

func TestNewBlockAccessListFromGloas_InvalidRLP(t *testing.T) {
	result := NewBlockAccessListFromGloas([]byte{0xff, 0xfe, 0xfd})
	if result == nil {
		t.Fatal("expected non-nil result for invalid RLP")
	}

	if len(result.GetEntries()) != 0 {
		t.Errorf("expected 0 entries for invalid RLP, got %d", len(result.GetEntries()))
	}
}

func rlpEncodeBAL(t *testing.T, cbal *bal.ConstructionBlockAccessList) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := cbal.EncodeRLP(&buf); err != nil {
		t.Fatalf("failed to RLP-encode BAL: %v", err)
	}

	return buf.Bytes()
}

func TestNewBlockAccessListFromGloas_BalanceChange(t *testing.T) {
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	cbal := bal.NewConstructionBlockAccessList()
	cbal.AccountRead(addr)
	cbal.Accounts[addr].BalanceChanges[3] = uint256.NewInt(1000)

	encoded := rlpEncodeBAL(t, &cbal)
	result := NewBlockAccessListFromGloas(encoded)

	if len(result.GetEntries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.GetEntries()))
	}

	entry := result.GetEntries()[0]
	if got := entry.GetAddress().GetValue(); got != "0x1111111111111111111111111111111111111111" {
		t.Errorf("unexpected address: %s", got)
	}

	if len(entry.GetBalanceChanges()) != 1 {
		t.Fatalf("expected 1 balance change, got %d", len(entry.GetBalanceChanges()))
	}

	bc := entry.GetBalanceChanges()[0]
	if bc.GetBlockAccessIndex().GetValue() != 3 {
		t.Errorf("expected block_access_index=3, got %d", bc.GetBlockAccessIndex().GetValue())
	}

	if bc.GetPostBalance().GetValue() != "1000" {
		t.Errorf("expected post_balance=1000, got %s", bc.GetPostBalance().GetValue())
	}
}

func TestNewBlockAccessListFromGloas_NonceChange(t *testing.T) {
	addr := common.HexToAddress("0x2222222222222222222222222222222222222222")

	cbal := bal.NewConstructionBlockAccessList()
	cbal.AccountRead(addr)
	cbal.Accounts[addr].NonceChanges[1] = 42

	encoded := rlpEncodeBAL(t, &cbal)
	result := NewBlockAccessListFromGloas(encoded)

	if len(result.GetEntries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.GetEntries()))
	}

	entry := result.GetEntries()[0]

	if len(entry.GetNonceChanges()) != 1 {
		t.Fatalf("expected 1 nonce change, got %d", len(entry.GetNonceChanges()))
	}

	nc := entry.GetNonceChanges()[0]
	if nc.GetBlockAccessIndex().GetValue() != 1 {
		t.Errorf("expected block_access_index=1, got %d", nc.GetBlockAccessIndex().GetValue())
	}

	if nc.GetNewNonce().GetValue() != 42 {
		t.Errorf("expected new_nonce=42, got %d", nc.GetNewNonce().GetValue())
	}
}

func TestNewBlockAccessListFromGloas_StorageChange(t *testing.T) {
	addr := common.HexToAddress("0x3333333333333333333333333333333333333333")
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	value := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000ff")

	cbal := bal.NewConstructionBlockAccessList()
	cbal.AccountRead(addr)
	cbal.Accounts[addr].StorageWrites[slot] = map[uint16]common.Hash{
		2: value,
	}

	encoded := rlpEncodeBAL(t, &cbal)
	result := NewBlockAccessListFromGloas(encoded)

	if len(result.GetEntries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.GetEntries()))
	}

	entry := result.GetEntries()[0]

	if len(entry.GetStorageChanges()) != 1 {
		t.Fatalf("expected 1 storage change, got %d", len(entry.GetStorageChanges()))
	}

	sc := entry.GetStorageChanges()[0]
	if sc.GetBlockAccessIndex().GetValue() != 2 {
		t.Errorf("expected block_access_index=2, got %d", sc.GetBlockAccessIndex().GetValue())
	}
}

func TestNewBlockAccessListFromGloas_CodeChange(t *testing.T) {
	addr := common.HexToAddress("0x4444444444444444444444444444444444444444")

	cbal := bal.NewConstructionBlockAccessList()
	cbal.AccountRead(addr)
	cbal.Accounts[addr].CodeChange = &bal.CodeChange{
		TxIndex: 7,
		Code:    []byte{0x60, 0x80, 0x60, 0x40},
	}

	encoded := rlpEncodeBAL(t, &cbal)
	result := NewBlockAccessListFromGloas(encoded)

	if len(result.GetEntries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.GetEntries()))
	}

	entry := result.GetEntries()[0]

	if len(entry.GetCodeChanges()) != 1 {
		t.Fatalf("expected 1 code change, got %d", len(entry.GetCodeChanges()))
	}

	cc := entry.GetCodeChanges()[0]
	if cc.GetBlockAccessIndex().GetValue() != 7 {
		t.Errorf("expected block_access_index=7, got %d", cc.GetBlockAccessIndex().GetValue())
	}

	if cc.GetNewCode().GetValue() != "0x60806040" {
		t.Errorf("expected new_code=0x60806040, got %s", cc.GetNewCode().GetValue())
	}
}
