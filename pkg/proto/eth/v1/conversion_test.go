package v1

import (
	"encoding/hex"
	"fmt"
	"testing"
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

// TestNewBlockAccessListFromGloas_RealDevnetData tests decoding real BAL data
// captured from a Gloas devnet beacon node (slot 200). Contains 4 entries:
// - Withdrawal Request Contract (reads only)
// - Consolidation Request Contract (reads only)
// - History Storage Contract (1 storage write)
// - Beacon Roots Contract (2 storage writes)
func TestNewBlockAccessListFromGloas_RealDevnetData(t *testing.T) {
	rawHex := "f8d1de9400000961ef480eb55e80d19ad83579a64c007002c0c480010203c0c0c0" +
		"de940000bbddc7ce488642fb579f8b00f3a590007251c0c480010203c0c0c0" +
		"f841940000f90827f1c53a10cb7a02335b175320002935e7e681c7e3e280a02da04ac41c52c5ed22ac6e519822f2a3b5e852d835cab242548ba7cd531b8519c0c0c0c0" +
		"f84e94000f3df6d732807ef1319fb7b8bb8522d0beac02f4cb821316c7c6808469c1e4ede7823315e3e280a03879f24e13e88a5db134e7092a18829d88875fc5c4252d3f27b21b7a96fd2e8bc0c0c0c0"

	rawBytes, err := hex.DecodeString(rawHex)
	if err != nil {
		t.Fatalf("failed to decode hex: %v", err)
	}

	result := NewBlockAccessListFromGloas(rawBytes)

	if len(result.GetEntries()) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result.GetEntries()))
	}

	// Entry 0: Withdrawal Request Contract - reads only, no writes
	entry0 := result.GetEntries()[0]
	if got := entry0.GetAddress().GetValue(); got != "0x00000961ef480eb55e80d19ad83579a64c007002" {
		t.Errorf("entry 0 address: got %s", got)
	}

	if len(entry0.GetStorageChanges()) != 0 {
		t.Errorf("entry 0: expected 0 storage changes, got %d", len(entry0.GetStorageChanges()))
	}

	if len(entry0.GetStorageReads()) != 4 {
		t.Fatalf("entry 0: expected 4 storage reads, got %d", len(entry0.GetStorageReads()))
	}

	// Verify read slots are sequential (0x00, 0x01, 0x02, 0x03)
	for i, r := range entry0.GetStorageReads() {
		expected := fmt.Sprintf("0x00000000000000000000000000000000000000000000000000000000000000%02x", i)
		if r.GetValue() != expected {
			t.Errorf("entry 0 read[%d]: expected %s, got %s", i, expected, r.GetValue())
		}
	}

	// Entry 2: History Storage Contract - 1 storage write
	entry2 := result.GetEntries()[2]
	if len(entry2.GetStorageChanges()) != 1 {
		t.Fatalf("entry 2: expected 1 storage change, got %d", len(entry2.GetStorageChanges()))
	}

	sc := entry2.GetStorageChanges()[0]
	if sc.GetBlockAccessIndex().GetValue() != 0 {
		t.Errorf("expected block_access_index=0, got %d", sc.GetBlockAccessIndex().GetValue())
	}

	// Entry 3: Beacon Roots Contract - 2 storage writes
	entry3 := result.GetEntries()[3]
	if got := entry3.GetAddress().GetValue(); got != "0x000f3df6d732807ef1319fb7b8bb8522d0beac02" {
		t.Errorf("entry 3 address: got %s", got)
	}

	if len(entry3.GetStorageChanges()) != 2 {
		t.Fatalf("entry 3: expected 2 storage changes, got %d", len(entry3.GetStorageChanges()))
	}
}
