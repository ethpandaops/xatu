package route

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeGraffiti(t *testing.T) {
	tests := map[string]struct {
		graffiti []byte
		want     string
	}{
		"empty":            {graffiti: make([]byte, 32), want: ""},
		"full width":       {graffiti: []byte("0123456789abcdef0123456789abcdef"), want: "0123456789abcdef0123456789abcdef"},
		"interior null":    {graffiti: append([]byte("a\x00b"), make([]byte, 29)...), want: "a\x00b"},
		"emoji":            {graffiti: append([]byte("stake \xf0\x9f\x90\xa0"), make([]byte, 22)...), want: "stake \U0001F420"},
		"invalid sequence": {graffiti: append([]byte{'o', 'k', 0xff, 0xfe}, make([]byte, 28)...), want: "ok�"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			raw, text, ok := DecodeGraffiti("0x" + hex.EncodeToString(tt.graffiti))
			assert.True(t, ok)
			assert.Equal(t, tt.graffiti, raw)
			assert.Equal(t, tt.want, text)
		})
	}

	for name, graffiti := range map[string]string{"empty": "", "short": "0x4745", "non hex": "0xzz"} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, _, ok := DecodeGraffiti(graffiti)
			assert.False(t, ok)
		})
	}
}
