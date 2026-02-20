package flattener

import (
	"math"
	"math/big"
	"net/netip"
	"strconv"
	"strings"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/google/uuid"
)

// ParseIPv6 converts a string IP address to proto.IPv6.
// Returns zero value on parse failure.
func ParseIPv6(s string) proto.IPv6 {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return proto.IPv6{}
	}

	return proto.ToIPv6(addr)
}

// ParseIPv4 converts a string IPv4 address to proto.IPv4.
// Returns zero value on parse failure.
func ParseIPv4(s string) proto.IPv4 {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return proto.IPv4(0)
	}

	if !addr.Is4() {
		return proto.IPv4(0)
	}

	b := addr.As4()

	return proto.IPv4(uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]))
}

// ParseUUID converts a string UUID to uuid.UUID.
// Returns zero value on parse failure.
func ParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}
	}

	return id
}

// ParseUInt128 converts a decimal string to proto.UInt128.
// Returns zero value on parse failure.
func ParseUInt128(s string) proto.UInt128 {
	s = strings.TrimSpace(s)
	if s == "" {
		return proto.UInt128{}
	}

	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return proto.UInt128{}
	}

	if n.Sign() < 0 {
		return proto.UInt128{}
	}

	var buf [16]byte

	n.FillBytes(buf[:])

	hi := uint64(buf[0])<<56 | uint64(buf[1])<<48 | uint64(buf[2])<<40 | uint64(buf[3])<<32 |
		uint64(buf[4])<<24 | uint64(buf[5])<<16 | uint64(buf[6])<<8 | uint64(buf[7])
	lo := uint64(buf[8])<<56 | uint64(buf[9])<<48 | uint64(buf[10])<<40 | uint64(buf[11])<<32 |
		uint64(buf[12])<<24 | uint64(buf[13])<<16 | uint64(buf[14])<<8 | uint64(buf[15])

	return proto.UInt128{Low: lo, High: hi}
}

// ParseUInt256 converts a decimal or hex string to proto.UInt256.
// Returns zero value on parse failure.
func ParseUInt256(s string) proto.UInt256 {
	s = strings.TrimSpace(s)
	if s == "" {
		return proto.UInt256{}
	}

	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		hexStr := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
		if _, ok := n.SetString(hexStr, 16); !ok {
			return proto.UInt256{}
		}
	}

	if n.Sign() < 0 {
		return proto.UInt256{}
	}

	var buf [32]byte

	n.FillBytes(buf[:])

	return proto.UInt256{
		Low: proto.UInt128{
			Low: uint64(buf[24])<<56 | uint64(buf[25])<<48 | uint64(buf[26])<<40 | uint64(buf[27])<<32 |
				uint64(buf[28])<<24 | uint64(buf[29])<<16 | uint64(buf[30])<<8 | uint64(buf[31]),
			High: uint64(buf[16])<<56 | uint64(buf[17])<<48 | uint64(buf[18])<<40 | uint64(buf[19])<<32 |
				uint64(buf[20])<<24 | uint64(buf[21])<<16 | uint64(buf[22])<<8 | uint64(buf[23]),
		},
		High: proto.UInt128{
			Low: uint64(buf[8])<<56 | uint64(buf[9])<<48 | uint64(buf[10])<<40 | uint64(buf[11])<<32 |
				uint64(buf[12])<<24 | uint64(buf[13])<<16 | uint64(buf[14])<<8 | uint64(buf[15]),
			High: uint64(buf[0])<<56 | uint64(buf[1])<<48 | uint64(buf[2])<<40 | uint64(buf[3])<<32 |
				uint64(buf[4])<<24 | uint64(buf[5])<<16 | uint64(buf[6])<<8 | uint64(buf[7]),
		},
	}
}

// UInt128ToString converts a proto.UInt128 to its decimal string representation.
func UInt128ToString(v proto.UInt128) string {
	if v.High == 0 {
		return strconv.FormatUint(v.Low, 10)
	}

	n := new(big.Int).SetUint64(v.High)
	n.Lsh(n, 64)
	n.Or(n, new(big.Int).SetUint64(v.Low))

	return n.String()
}

// UInt256ToString converts a proto.UInt256 to its decimal string representation.
func UInt256ToString(v proto.UInt256) string {
	if v.High == (proto.UInt128{}) {
		return UInt128ToString(v.Low)
	}

	hi := new(big.Int).SetUint64(v.High.High)
	hi.Lsh(hi, 64)
	hi.Or(hi, new(big.Int).SetUint64(v.High.Low))
	hi.Lsh(hi, 128)

	lo := new(big.Int).SetUint64(v.Low.High)
	lo.Lsh(lo, 64)
	lo.Or(lo, new(big.Int).SetUint64(v.Low.Low))

	hi.Or(hi, lo)

	return hi.String()
}

// Uint8SliceToIntSlice converts []uint8 (which JSON-marshals as base64)
// to []int for correct JSON array serialization.
func Uint8SliceToIntSlice(b []uint8) []int {
	if b == nil {
		return nil
	}

	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}

	return out
}

// ByteSlicesToStrings converts [][]byte to []string via simple type cast.
// Used by generated Snapshot code for Array(FixedString(N)) columns where
// the stored bytes are already UTF-8 strings (e.g. hex hashes with "0x" prefix).
func ByteSlicesToStrings(bs [][]byte) []string {
	if bs == nil {
		return nil
	}

	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = string(b)
	}

	return out
}

// FormatDecimal formats a scaled Decimal64 value back to its string
// representation with the given number of decimal places. For example,
// FormatDecimal(272800, 3) returns "272.800".
func FormatDecimal(v int64, scale int) string {
	if scale <= 0 {
		return strconv.FormatInt(v, 10)
	}

	return strconv.FormatFloat(float64(v)/math.Pow10(scale), 'f', scale, 64)
}

// ScaleDecimal converts a string decimal value to a scaled int64 for
// ch-go Decimal columns. For example, "1.618" with scale 3 becomes 1618.
// Returns 0 on parse failure.
func ScaleDecimal(s string, scale int) int64 {
	if scale <= 0 {
		return 0
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}

	return int64(math.Round(f * math.Pow10(scale)))
}

// PadToFixed pads or truncates b to exactly size bytes.
func PadToFixed(b []byte, size int) []byte {
	if len(b) == size {
		return b
	}

	padded := make([]byte, size)
	copy(padded, b)

	return padded
}

// SafeColFixedStr wraps proto.ColFixedStr to pad or truncate values
// on Append, avoiding panics from ch-go on short byte slices.
type SafeColFixedStr struct {
	proto.ColFixedStr
}

// Append pads or truncates b to the column's fixed size before appending.
func (c *SafeColFixedStr) Append(b []byte) {
	c.ColFixedStr.Append(PadToFixed(b, c.Size))
}

// StringsToBytes converts a string slice to a byte-slice slice.
// Used by generated columnar batch code for Array(FixedString(N)) columns.
func StringsToBytes(ss []string) [][]byte {
	bs := make([][]byte, len(ss))
	for i, s := range ss {
		bs[i] = []byte(s)
	}

	return bs
}

// NewNullableFixedStr creates a ColNullable[[]byte] backed by a
// SafeColFixedStr of the given size. Used by generated columnar batch
// code for Nullable(FixedString(N)) columns.
func NewNullableFixedStr(size int) *proto.ColNullable[[]byte] {
	fs := &SafeColFixedStr{}
	fs.SetSize(size)

	return &proto.ColNullable[[]byte]{Values: fs}
}

// TypedColInput wraps a proto.ColInput to override Type() with a full
// type string that includes parameters (e.g. "Decimal(10, 3)" instead
// of bare "Decimal64").
type TypedColInput struct {
	proto.ColInput
	CHType proto.ColumnType
}

// Type returns the full CH type string.
func (c *TypedColInput) Type() proto.ColumnType { return c.CHType }

// Reset delegates to the inner column.
func (c *TypedColInput) Reset() {
	if r, ok := c.ColInput.(interface{ Reset() }); ok {
		r.Reset()
	}
}
