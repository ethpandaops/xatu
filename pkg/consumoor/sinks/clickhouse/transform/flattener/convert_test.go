package flattener

import (
	"testing"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIPv6(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isZero bool
	}{
		{"valid_ipv6", "2001:db8::1", false},
		{"valid_ipv4_mapped", "::ffff:192.168.1.1", false},
		{"valid_ipv4", "192.168.1.1", false},
		{"loopback", "::1", false},
		{"empty", "", true},
		{"invalid", "not-an-ip", true},
		{"malformed", "999.999.999.999", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIPv6(tt.input)
			if tt.isZero {
				assert.Equal(t, proto.IPv6{}, result)
			} else {
				assert.NotEqual(t, proto.IPv6{}, result)
			}
		})
	}
}

func TestParseIPv4(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isZero bool
	}{
		{"valid", "192.168.1.1", false},
		{"loopback", "127.0.0.1", false},
		{"broadcast", "255.255.255.255", false},
		{"empty", "", true},
		{"invalid", "not-an-ip", true},
		// IPv6 addresses should return zero for ParseIPv4.
		{"ipv6", "2001:db8::1", true},
		{"ipv6_loopback", "::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIPv4(tt.input)
			if tt.isZero {
				assert.Equal(t, proto.IPv4(0), result)
			} else {
				assert.NotEqual(t, proto.IPv4(0), result)
			}
		})
	}
}

func TestParseIPv4RoundTrip(t *testing.T) {
	// Verify specific byte ordering for a known IP.
	result := ParseIPv4("192.168.1.2")
	expected := proto.IPv4(192<<24 | 168<<16 | 1<<8 | 2)
	assert.Equal(t, expected, result)
}

func TestParseUUID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isZero bool
	}{
		{"valid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid_uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"nil_uuid", "00000000-0000-0000-0000-000000000000", true}, // valid parse but indistinguishable from zero
		{"empty", "", true},
		{"invalid", "not-a-uuid", true},
		{"too_short", "550e8400", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseUUID(tt.input)
			if tt.isZero {
				assert.Equal(t, uuid.UUID{}, result)
			} else {
				assert.NotEqual(t, uuid.UUID{}, result)
			}
		})
	}
}

func TestParseUInt128(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  proto.UInt128
	}{
		{"zero", "0", proto.UInt128{Low: 0, High: 0}},
		{"small", "42", proto.UInt128{Low: 42, High: 0}},
		{"max_uint64", "18446744073709551615", proto.UInt128{Low: 18446744073709551615, High: 0}},
		{"above_uint64", "18446744073709551616", proto.UInt128{Low: 0, High: 1}},
		{"large", "340282366920938463463374607431768211455", proto.UInt128{Low: ^uint64(0), High: ^uint64(0)}},
		// Error cases → zero.
		{"empty", "", proto.UInt128{}},
		{"negative", "-1", proto.UInt128{}},
		{"invalid", "abc", proto.UInt128{}},
		{"whitespace", "  42  ", proto.UInt128{Low: 42, High: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseUInt128(tt.input))
		})
	}
}

func TestUInt128ToString(t *testing.T) {
	tests := []struct {
		name  string
		input proto.UInt128
		want  string
	}{
		{"zero", proto.UInt128{Low: 0, High: 0}, "0"},
		{"small", proto.UInt128{Low: 42, High: 0}, "42"},
		{"max_uint64", proto.UInt128{Low: ^uint64(0), High: 0}, "18446744073709551615"},
		{"above_uint64", proto.UInt128{Low: 0, High: 1}, "18446744073709551616"},
		{"max_uint128", proto.UInt128{Low: ^uint64(0), High: ^uint64(0)}, "340282366920938463463374607431768211455"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, UInt128ToString(tt.input))
		})
	}
}

func TestUInt128RoundTrip(t *testing.T) {
	values := []string{
		"0",
		"1",
		"18446744073709551615",
		"18446744073709551616",
		"340282366920938463463374607431768211455",
		"123456789012345678901234567890",
	}

	for _, v := range values {
		t.Run(v, func(t *testing.T) {
			parsed := ParseUInt128(v)
			assert.Equal(t, v, UInt128ToString(parsed))
		})
	}
}

func TestParseUInt256(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isZero bool
	}{
		{"small", "42", false},
		{"hex", "0xff", false},
		{"hex_upper", "0xFF", false},
		{"large_decimal", "115792089237316195423570985008687907853269984665640564039457584007913129639935", false},
		// Error cases.
		{"empty", "", true},
		{"negative", "-1", true},
		{"invalid", "not_a_number", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseUInt256(tt.input)
			if tt.isZero {
				assert.Equal(t, proto.UInt256{}, result)
			} else {
				assert.NotEqual(t, proto.UInt256{}, result, "expected non-zero result for %q", tt.input)
			}
		})
	}
}

func TestUInt256ToString(t *testing.T) {
	tests := []struct {
		name  string
		input proto.UInt256
		want  string
	}{
		{"zero", proto.UInt256{}, "0"},
		{"small", proto.UInt256{Low: proto.UInt128{Low: 42}}, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, UInt256ToString(tt.input))
		})
	}
}

func TestUInt256RoundTrip(t *testing.T) {
	values := []string{
		"0",
		"42",
		"18446744073709551616",
		"340282366920938463463374607431768211455",
		"115792089237316195423570985008687907853269984665640564039457584007913129639935",
	}

	for _, v := range values {
		t.Run(v, func(t *testing.T) {
			parsed := ParseUInt256(v)
			assert.Equal(t, v, UInt256ToString(parsed))
		})
	}
}

func TestUInt256HexRoundTrip(t *testing.T) {
	// Hex input should parse, and the decimal output should match.
	parsed := ParseUInt256("0xff")
	assert.Equal(t, "255", UInt256ToString(parsed))

	parsed = ParseUInt256("0x100")
	assert.Equal(t, "256", UInt256ToString(parsed))
}

func TestFormatDecimal(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		scale int
		want  string
	}{
		{"zero_scale", 12345, 0, "12345"},
		{"negative_scale", 12345, -1, "12345"},
		{"scale_3", 272800, 3, "272.800"},
		{"scale_3_small", 1618, 3, "1.618"},
		{"scale_3_zero", 0, 3, "0.000"},
		{"scale_6", 1000000, 6, "1.000000"},
		{"negative_value", -1618, 3, "-1.618"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatDecimal(tt.value, tt.scale))
		})
	}
}

func TestScaleDecimal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		scale int
		want  int64
	}{
		{"simple", "1.618", 3, 1618},
		{"integer", "42", 3, 42000},
		{"zero", "0", 3, 0},
		{"negative", "-1.5", 3, -1500},
		{"whitespace", "  1.5  ", 3, 1500},
		// Error/edge cases.
		{"invalid", "abc", 3, 0},
		{"empty", "", 3, 0},
		{"zero_scale", "1.5", 0, 0},
		{"negative_scale", "1.5", -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ScaleDecimal(tt.input, tt.scale))
		})
	}
}

func TestFormatScaleDecimalRoundTrip(t *testing.T) {
	// Scale then format should return the original string (with fixed decimal places).
	scaled := ScaleDecimal("272.800", 3)
	assert.Equal(t, "272.800", FormatDecimal(scaled, 3))

	scaled = ScaleDecimal("1.618", 3)
	assert.Equal(t, "1.618", FormatDecimal(scaled, 3))
}

func TestPadToFixed(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		size  int
		want  []byte
	}{
		{"exact_size", []byte("hello"), 5, []byte("hello")},
		{"pad_short", []byte("hi"), 5, []byte("hi\x00\x00\x00")},
		{"truncate_long", []byte("hello world"), 5, []byte("hello")},
		{"nil_input", nil, 3, []byte("\x00\x00\x00")},
		{"empty_input", []byte{}, 3, []byte("\x00\x00\x00")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadToFixed(tt.input, tt.size)
			require.Len(t, result, tt.size)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSafeColFixedStr(t *testing.T) {
	var col SafeColFixedStr
	col.SetSize(4)

	// Short value — should be padded.
	col.Append([]byte("ab"))
	assert.Equal(t, 1, col.Rows())

	// Exact value.
	col.Append([]byte("abcd"))
	assert.Equal(t, 2, col.Rows())

	// Long value — should be truncated.
	col.Append([]byte("abcdef"))
	assert.Equal(t, 3, col.Rows())

	// Nil value — should be padded to zeros.
	col.Append(nil)
	assert.Equal(t, 4, col.Rows())

	// Verify reads.
	assert.Equal(t, []byte("ab\x00\x00"), col.Row(0))
	assert.Equal(t, []byte("abcd"), col.Row(1))
	assert.Equal(t, []byte("abcd"), col.Row(2)) // truncated to 4
	assert.Equal(t, []byte("\x00\x00\x00\x00"), col.Row(3))
}

func TestUint8SliceToIntSlice(t *testing.T) {
	assert.Nil(t, Uint8SliceToIntSlice(nil))
	assert.Equal(t, []int{}, Uint8SliceToIntSlice([]uint8{}))
	assert.Equal(t, []int{0, 1, 255}, Uint8SliceToIntSlice([]uint8{0, 1, 255}))
}

func TestByteSlicesToStrings(t *testing.T) {
	assert.Nil(t, ByteSlicesToStrings(nil))
	assert.Equal(t, []string{}, ByteSlicesToStrings([][]byte{}))
	assert.Equal(t,
		[]string{"hello", "world"},
		ByteSlicesToStrings([][]byte{[]byte("hello"), []byte("world")}),
	)
}

func TestNewNullableFixedStr(t *testing.T) {
	col := NewNullableFixedStr(4)
	require.NotNil(t, col)

	// Append a non-null value.
	col.Append(proto.NewNullable([]byte("ab")))
	assert.Equal(t, 1, col.Rows())

	// Append a null value.
	col.Append(proto.Nullable[[]byte]{})
	assert.Equal(t, 2, col.Rows())

	// Read back.
	row0 := col.Row(0)
	assert.True(t, row0.Set)
	assert.Equal(t, []byte("ab\x00\x00"), row0.Value) // padded

	row1 := col.Row(1)
	assert.False(t, row1.Set)
}

func TestStringsToBytes(t *testing.T) {
	result := StringsToBytes([]string{"hello", "world"})
	assert.Equal(t, [][]byte{[]byte("hello"), []byte("world")}, result)

	result = StringsToBytes(nil)
	assert.Len(t, result, 0)
}
