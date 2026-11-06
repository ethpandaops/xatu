package route

import (
	"encoding/hex"
	"strings"
)

// DecodeGraffiti decodes a block body graffiti, hex encoded with an optional 0x prefix. It returns the 32 raw bytes
// and the graffiti as text: UTF-8 with trailing null bytes removed and invalid sequences replaced. ok is false when
// the graffiti isn't 32 bytes of hex.
func DecodeGraffiti(graffiti string) (raw []byte, text string, ok bool) {
	raw, err := hex.DecodeString(strings.TrimPrefix(graffiti, "0x"))
	if err != nil || len(raw) != 32 {
		return nil, "", false
	}

	return raw, strings.ToValidUTF8(strings.TrimRight(string(raw), "\x00"), "\uFFFD"), true
}
