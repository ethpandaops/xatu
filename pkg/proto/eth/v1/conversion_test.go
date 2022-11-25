package v1

import "testing"

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
