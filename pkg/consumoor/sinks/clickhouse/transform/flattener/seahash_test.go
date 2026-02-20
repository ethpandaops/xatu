package flattener

import "testing"

func TestSeaHash64ReferenceVector(t *testing.T) {
	t.Parallel()

	got := SeaHash64("to be or not to be")
	want := uint64(1988685042348123509)

	if got != want {
		t.Fatalf("SeaHash64 mismatch: got=%d want=%d", got, want)
	}
}
