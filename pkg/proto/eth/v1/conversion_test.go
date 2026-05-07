package v1

import (
	"encoding/hex"
	"fmt"
	"testing"

	bitfield "github.com/OffchainLabs/go-bitfield"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
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
		if r.GetKey().GetValue() != expected {
			t.Errorf("entry 0 read[%d]: expected %s, got %s", i, expected, r.GetKey().GetValue())
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

func TestNewSignedExecutionPayloadBidFromGloas_Nil(t *testing.T) {
	if got := NewSignedExecutionPayloadBidFromGloas(nil); got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}

	if got := NewSignedExecutionPayloadBidFromGloas(&gloas.SignedExecutionPayloadBid{}); got != nil {
		t.Errorf("expected nil for bid with nil Message, got %v", got)
	}
}

func TestNewSignedExecutionPayloadBidFromGloas_Populated(t *testing.T) {
	parentHash := phase0.Hash32{0x11, 0x22}
	parentRoot := phase0.Root{0x33, 0x44}
	blockHash := phase0.Hash32{0x55, 0x66}
	prevRandao := phase0.Root{0x77, 0x88}
	feeRecipient := bellatrix.ExecutionAddress{0x99, 0xaa}
	commitment := deneb.KZGCommitment{0xbb, 0xcc}

	bid := &gloas.SignedExecutionPayloadBid{
		Message: &gloas.ExecutionPayloadBid{
			ParentBlockHash:    parentHash,
			ParentBlockRoot:    parentRoot,
			BlockHash:          blockHash,
			PrevRandao:         prevRandao,
			FeeRecipient:       feeRecipient,
			GasLimit:           30_000_000,
			BuilderIndex:       gloas.BuilderIndex(7),
			Slot:               phase0.Slot(99),
			Value:              phase0.Gwei(123_456),
			ExecutionPayment:   phase0.Gwei(7_890),
			BlobKZGCommitments: []deneb.KZGCommitment{commitment},
		},
		Signature: phase0.BLSSignature{0xde, 0xad, 0xbe, 0xef},
	}

	got := NewSignedExecutionPayloadBidFromGloas(bid)
	if got == nil {
		t.Fatal("expected non-nil result")
	}

	msg := got.GetMessage()
	if msg == nil {
		t.Fatal("expected non-nil message")
	}

	if msg.GetParentBlockHash() != parentHash.String() {
		t.Errorf("parent_block_hash mismatch: got %q want %q", msg.GetParentBlockHash(), parentHash.String())
	}

	if msg.GetParentBlockRoot() != parentRoot.String() {
		t.Errorf("parent_block_root mismatch: got %q want %q", msg.GetParentBlockRoot(), parentRoot.String())
	}

	if msg.GetBlockHash() != blockHash.String() {
		t.Errorf("block_hash mismatch: got %q want %q", msg.GetBlockHash(), blockHash.String())
	}

	if msg.GetFeeRecipient() != feeRecipient.String() {
		t.Errorf("fee_recipient mismatch: got %q want %q", msg.GetFeeRecipient(), feeRecipient.String())
	}

	if v := msg.GetGasLimit().GetValue(); v != 30_000_000 {
		t.Errorf("gas_limit: got %d want 30000000", v)
	}

	if v := msg.GetBuilderIndex().GetValue(); v != 7 {
		t.Errorf("builder_index: got %d want 7", v)
	}

	if v := msg.GetSlot().GetValue(); v != 99 {
		t.Errorf("slot: got %d want 99", v)
	}

	if v := msg.GetValue().GetValue(); v != 123_456 {
		t.Errorf("value: got %d want 123456", v)
	}

	if v := msg.GetExecutionPayment().GetValue(); v != 7_890 {
		t.Errorf("execution_payment: got %d want 7890", v)
	}

	if commitments := msg.GetBlobKzgCommitments(); len(commitments) != 1 {
		t.Errorf("blob_kzg_commitments len: got %d want 1", len(commitments))
	} else if commitments[0] != KzgCommitmentToString(commitment) {
		t.Errorf("blob_kzg_commitments[0]: got %q want %q", commitments[0], KzgCommitmentToString(commitment))
	}

	if got.GetSignature() != bid.Signature.String() {
		t.Errorf("signature mismatch: got %q want %q", got.GetSignature(), bid.Signature.String())
	}
}

func TestNewPayloadAttestationsFromGloas_Empty(t *testing.T) {
	if got := NewPayloadAttestationsFromGloas(nil); len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got len %d", len(got))
	}

	if got := NewPayloadAttestationsFromGloas([]*gloas.PayloadAttestation{}); len(got) != 0 {
		t.Errorf("expected empty slice for empty input, got len %d", len(got))
	}

	// Nil entries in the slice are skipped, not panicked on.
	if got := NewPayloadAttestationsFromGloas([]*gloas.PayloadAttestation{nil}); len(got) != 0 {
		t.Errorf("expected nil entries to be skipped, got len %d", len(got))
	}
}

func TestNewPayloadAttestationsFromGloas_Populated(t *testing.T) {
	bits := bitfield.NewBitvector512()
	bits.SetBitAt(3, true)
	bits.SetBitAt(11, true)

	beaconRoot := phase0.Root{0xa0, 0xb1}

	atts := []*gloas.PayloadAttestation{
		{
			AggregationBits: bits,
			Data: &gloas.PayloadAttestationData{
				BeaconBlockRoot:   beaconRoot,
				Slot:              phase0.Slot(50),
				PayloadPresent:    true,
				BlobDataAvailable: false,
			},
			Signature: phase0.BLSSignature{0x01, 0x02},
		},
	}

	got := NewPayloadAttestationsFromGloas(atts)
	if len(got) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(got))
	}

	a := got[0]

	wantBits := fmt.Sprintf("0x%x", bits)
	if a.GetAggregationBits() != wantBits {
		t.Errorf("aggregation_bits: got %q want %q", a.GetAggregationBits(), wantBits)
	}

	data := a.GetData()
	if data == nil {
		t.Fatal("expected non-nil data")
	}

	if data.GetBeaconBlockRoot() != beaconRoot.String() {
		t.Errorf("beacon_block_root: got %q want %q", data.GetBeaconBlockRoot(), beaconRoot.String())
	}

	if data.GetSlot().GetValue() != 50 {
		t.Errorf("slot: got %d want 50", data.GetSlot().GetValue())
	}

	if !data.GetPayloadPresent() {
		t.Error("expected payload_present=true")
	}

	if data.GetBlobDataAvailable() {
		t.Error("expected blob_data_available=false")
	}

	if a.GetSignature() != atts[0].Signature.String() {
		t.Errorf("signature mismatch: got %q want %q", a.GetSignature(), atts[0].Signature.String())
	}
}

// TestNewPayloadAttestationsFromGloas_NilData ensures the conversion is robust
// against malformed inputs where Data is missing.
func TestNewPayloadAttestationsFromGloas_NilData(t *testing.T) {
	atts := []*gloas.PayloadAttestation{
		{
			AggregationBits: bitfield.NewBitvector512(),
			Data:            nil,
			Signature:       phase0.BLSSignature{},
		},
	}

	got := NewPayloadAttestationsFromGloas(atts)
	if len(got) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(got))
	}

	if got[0].GetData() != nil {
		t.Errorf("expected nil data for input with nil Data, got %v", got[0].GetData())
	}
}
