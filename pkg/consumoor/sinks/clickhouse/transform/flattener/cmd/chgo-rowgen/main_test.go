package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProtoGen(t *testing.T) {
	tests := []struct {
		chType    string
		fieldType string
		isPtr     bool
		zeroVal   string
		inputRef  string
	}{
		// Scalar integers.
		{"Bool", "proto.ColBool", false, "false", "ref"},
		{"UInt8", "proto.ColUInt8", false, "0", "ref"},
		{"UInt16", "proto.ColUInt16", false, "0", "ref"},
		{"UInt32", "proto.ColUInt32", false, "0", "ref"},
		{"UInt64", "proto.ColUInt64", false, "0", "ref"},
		{"Int8", "proto.ColInt8", false, "0", "ref"},
		{"Int16", "proto.ColInt16", false, "0", "ref"},
		{"Int32", "proto.ColInt32", false, "0", "ref"},
		{"Int64", "proto.ColInt64", false, "0", "ref"},
		{"Float32", "proto.ColFloat32", false, "0", "ref"},
		{"Float64", "proto.ColFloat64", false, "0", "ref"},

		// String.
		{"String", "proto.ColStr", false, `""`, "ref"},

		// DateTime variants.
		{"DateTime", "proto.ColDateTime", false, "time.Time{}", "ref"},
		{"DateTime64(3)", "proto.ColDateTime64", false, "time.Time{}", "ref"},
		{"DateTime64(6, 'UTC')", "proto.ColDateTime64", false, "time.Time{}", "ref"},

		// Network types.
		{"IPv4", "proto.ColIPv4", false, "proto.IPv4(0)", "ref"},
		{"IPv6", "proto.ColIPv6", false, "proto.IPv6{}", "ref"},

		// UUID.
		{"UUID", "proto.ColUUID", false, "uuid.UUID{}", "ref"},

		// Big integers.
		{"UInt128", "proto.ColUInt128", false, "proto.UInt128{}", "ref"},
		{"UInt256", "proto.ColUInt256", false, "proto.UInt256{}", "ref"},

		// FixedString.
		{"FixedString(66)", "flattener.SafeColFixedStr", false, "nil", "ref"},
		{"FixedString(1)", "flattener.SafeColFixedStr", false, "nil", "ref"},

		// Decimal.
		{"Decimal64(3)", "proto.ColDecimal64", false, "0", "ref"},
		{"Decimal(10, 5)", "proto.ColDecimal64", false, "0", "ref"},

		// Enum and Date fallback to String.
		{"Enum8('a' = 1, 'b' = 2)", "proto.ColStr", false, `""`, "ref"},
		{"Enum16('x' = 100)", "proto.ColStr", false, `""`, "ref"},
		{"Date", "proto.ColStr", false, `""`, "ref"},
		{"Date32", "proto.ColStr", false, `""`, "ref"},

		// LowCardinality wrappers.
		{"LowCardinality(String)", "proto.ColStr", false, `""`, "ref"},
		{"LowCardinality(UInt32)", "proto.ColUInt32", false, "0", "ref"},

		// Nullable scalars.
		{"Nullable(UInt32)", "*proto.ColNullable[uint32]", true, "proto.Nullable[uint32]{}", "ptr"},
		{"Nullable(String)", "*proto.ColNullable[string]", true, "proto.Nullable[string]{}", "ptr"},
		{"Nullable(Float64)", "*proto.ColNullable[float64]", true, "proto.Nullable[float64]{}", "ptr"},
		{"Nullable(IPv6)", "*proto.ColNullable[proto.IPv6]", true, "proto.Nullable[proto.IPv6]{}", "ptr"},
		{"Nullable(IPv4)", "*proto.ColNullable[proto.IPv4]", true, "proto.Nullable[proto.IPv4]{}", "ptr"},
		{"Nullable(DateTime)", "*proto.ColNullable[time.Time]", true, "proto.Nullable[time.Time]{}", "ptr"},

		// Nullable FixedString uses special constructor.
		{"Nullable(FixedString(66))", "*proto.ColNullable[[]byte]", true, "proto.Nullable[[]byte]{}", "ptr"},

		// LowCardinality(Nullable(T)) → treats as Nullable(T).
		{"LowCardinality(Nullable(String))", "*proto.ColNullable[string]", true, "proto.Nullable[string]{}", "ptr"},
		{"LowCardinality(Nullable(UInt32))", "*proto.ColNullable[uint32]", true, "proto.Nullable[uint32]{}", "ptr"},

		// Array scalars.
		{"Array(UInt32)", "*proto.ColArr[uint32]", true, "nil", "ptr"},
		{"Array(String)", "*proto.ColArr[string]", true, "nil", "ptr"},
		{"Array(Int64)", "*proto.ColArr[int64]", true, "nil", "ptr"},
		{"Array(UInt8)", "*proto.ColArr[uint8]", true, "nil", "ptr"},

		// Array(FixedString(N)) — special case.
		{"Array(FixedString(66))", "*proto.ColArr[[]byte]", true, "nil", "ptr"},

		// Array(Array(UInt32)) — nested array.
		{"Array(Array(UInt32))", "*proto.ColArr[[]uint32]", true, "nil", "ptr"},

		// Map.
		{"Map(String, String)", "*proto.ColMap[string, string]", true, "nil", "ptr"},

		// Unknown type falls back to String.
		{"SomeFutureType", "proto.ColStr", false, `""`, "ref"},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			gen := resolveProtoGen(tt.chType)
			assert.Equal(t, tt.fieldType, gen.fieldType, "fieldType")
			assert.Equal(t, tt.isPtr, gen.isPtr, "isPtr")
			assert.Equal(t, tt.zeroVal, gen.zeroVal, "zeroVal")
			assert.Equal(t, tt.inputRef, gen.inputRef, "inputRef")
		})
	}
}

func TestResolveProtoGenSnapshotFmt(t *testing.T) {
	tests := []struct {
		chType      string
		snapshotFmt string
	}{
		{"DateTime", "%s.Unix()"},
		{"DateTime64(3)", "%s.UnixMilli()"},
		{"DateTime64(6, 'UTC')", "%s.UnixMicro()"},
		{"UUID", "%s.String()"},
		{"UInt128", "flattener.UInt128ToString(%s)"},
		{"UInt256", "flattener.UInt256ToString(%s)"},
		{"FixedString(66)", "string(%s)"},
		{"Decimal64(3)", "flattener.FormatDecimal(int64(%s), 3)"},
		{"Decimal(18, 9)", "flattener.FormatDecimal(int64(%s), 9)"},
		// Array(UInt8) needs base64 workaround.
		{"Array(UInt8)", "flattener.Uint8SliceToIntSlice(%s)"},
		// Array(FixedString) needs byte-to-string conversion.
		{"Array(FixedString(66))", "flattener.ByteSlicesToStrings(%s)"},
		// Scalars without custom snapshot format.
		{"UInt32", ""},
		{"String", ""},
		{"Bool", ""},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			gen := resolveProtoGen(tt.chType)
			assert.Equal(t, tt.snapshotFmt, gen.snapshotFmt)
		})
	}
}

func TestResolveProtoGenInitExpr(t *testing.T) {
	tests := []struct {
		chType   string
		hasInit  bool
		contains string // substring the initExpr should contain
	}{
		{"DateTime64(3)", true, "Precision(3)"},
		{"DateTime64(6, 'UTC')", true, "Precision(6)"},
		{"FixedString(66)", true, "SetSize(66)"},
		{"FixedString(42)", true, "SetSize(42)"},
		{"Nullable(UInt32)", true, "Nullable()"},
		{"Nullable(FixedString(66))", true, "NewNullableFixedStr(66)"},
		{"Array(UInt32)", true, "NewArray"},
		{"Array(FixedString(10))", true, "SetSize(10)"},
		{"Map(String, String)", true, "NewMap"},
		// Plain scalars don't need init.
		{"UInt32", false, ""},
		{"String", false, ""},
		{"DateTime", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			gen := resolveProtoGen(tt.chType)
			if tt.hasInit {
				require.NotEmpty(t, gen.initExpr, "expected initExpr for %s", tt.chType)
				assert.Contains(t, gen.initExpr, tt.contains)
			} else {
				assert.Empty(t, gen.initExpr)
			}
		})
	}
}

func TestResolveProtoGenImportFlags(t *testing.T) {
	// DateTime-based types need time import for Nullable wrappers.
	gen := resolveProtoGen("Nullable(DateTime)")
	assert.True(t, gen.needsTime, "Nullable(DateTime) should need time import")

	gen = resolveProtoGen("Nullable(DateTime64(3))")
	assert.True(t, gen.needsTime, "Nullable(DateTime64) should need time import")

	// UUID needs uuid import.
	gen = resolveProtoGen("UUID")
	assert.True(t, gen.needsUUID)

	gen = resolveProtoGen("Nullable(UUID)")
	assert.True(t, gen.needsUUID)

	// Regular types don't need extra imports.
	gen = resolveProtoGen("UInt32")
	assert.False(t, gen.needsTime)
	assert.False(t, gen.needsUUID)
}

func TestProtoGoType(t *testing.T) {
	tests := []struct {
		chType     string
		wantGoType string
	}{
		{"Bool", "bool"},
		{"UInt8", "uint8"},
		{"UInt16", "uint16"},
		{"UInt32", "uint32"},
		{"UInt64", "uint64"},
		{"Int8", "int8"},
		{"Int16", "int16"},
		{"Int32", "int32"},
		{"Int64", "int64"},
		{"Float32", "float32"},
		{"Float64", "float64"},
		{"String", "string"},
		{"DateTime", "time.Time"},
		{"DateTime64(3)", "time.Time"},
		{"IPv4", "proto.IPv4"},
		{"IPv6", "proto.IPv6"},
		{"UUID", "uuid.UUID"},
		{"UInt128", "proto.UInt128"},
		{"UInt256", "proto.UInt256"},
		{"Decimal64(3)", "int64"},
		{"FixedString(66)", "[]byte"},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			gen := resolveProtoGen(tt.chType)
			assert.Equal(t, tt.wantGoType, gen.protoGoType())
		})
	}
}

func TestToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"slot", "Slot"},
		{"slot_start_date_time", "SlotStartDateTime"},
		{"meta_client_id", "MetaClientID"},
		{"meta_client_ip", "MetaClientIP"},
		{"meta_client_os", "MetaClientOS"},
		{"peer_id_unique_key", "PeerIDUniqueKey"},
		{"rpc_meta_message", "RPCMetaMessage"},
		{"api_method", "APIMethod"},
		{"cpu_count", "CPUCount"},
		{"asn_number", "ASNNumber"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toPascal(tt.input))
		})
	}
}

func TestSplitTopLevel(t *testing.T) {
	tests := []struct {
		input string
		sep   rune
		want  []string
	}{
		// Simple split.
		{"String, UInt32", ',', []string{"String", "UInt32"}},
		// Nested parentheses preserved.
		{"Map(String, String), UInt32", ',', []string{"Map(String, String)", "UInt32"}},
		// Deeply nested.
		{"Array(Nullable(UInt32)), String", ',', []string{"Array(Nullable(UInt32))", "String"}},
		// Single element.
		{"String", ',', []string{"String"}},
		// Empty.
		{"", ',', nil},
		// Multiple commas inside parens.
		{"Decimal(18, 9)", ',', []string{"Decimal(18, 9)"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitTopLevel(tt.input, tt.sep)
			if tt.want == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUnwrapTypeOK(t *testing.T) {
	tests := []struct {
		typ     string
		wrapper string
		inner   string
		ok      bool
	}{
		{"Nullable(UInt32)", "Nullable", "UInt32", true},
		{"Array(String)", "Array", "String", true},
		{"LowCardinality(Nullable(String))", "LowCardinality", "Nullable(String)", true},
		{"Map(String, UInt32)", "Map", "String, UInt32", true},
		// Not wrapped.
		{"UInt32", "Nullable", "", false},
		{"String", "Array", "", false},
		// Wrong wrapper.
		{"Nullable(UInt32)", "Array", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.typ+"_"+tt.wrapper, func(t *testing.T) {
			inner, ok := unwrapTypeOK(tt.typ, tt.wrapper)
			assert.Equal(t, tt.ok, ok)

			if tt.ok {
				assert.Equal(t, tt.inner, inner)
			}
		})
	}
}

func TestParseDateTime64Precision(t *testing.T) {
	tests := []struct {
		chType string
		want   int
	}{
		{"DateTime64(3)", 3},
		{"DateTime64(6)", 6},
		{"DateTime64(6, 'UTC')", 6},
		{"DateTime64(3, 'Europe/Berlin')", 3},
		{"DateTime64(0)", 0},
		// Fallbacks.
		{"DateTime64", 3},
		{"DateTime", 3},
		{"NotDateTime", 3},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			assert.Equal(t, tt.want, parseDateTime64Precision(tt.chType))
		})
	}
}

func TestParseFixedStrSize(t *testing.T) {
	tests := []struct {
		chType string
		want   int
	}{
		{"FixedString(66)", 66},
		{"FixedString(1)", 1},
		{"FixedString(128)", 128},
		// Fallbacks.
		{"FixedString(-1)", 1},
		{"FixedString(abc)", 1},
		{"FixedString", 1},
		{"String", 1},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFixedStrSize(tt.chType))
		})
	}
}

func TestParseDecScale(t *testing.T) {
	tests := []struct {
		chType string
		want   int
	}{
		{"Decimal64(3)", 3},
		{"Decimal64(9)", 9},
		{"Decimal(18, 5)", 5},
		{"Decimal(38, 18)", 18},
		{"Decimal32(2)", 2},
		{"Decimal128(10)", 10},
		{"Decimal256(20)", 20},
		// Fallbacks.
		{"Decimal64", 0},
		{"String", 0},
	}

	for _, tt := range tests {
		t.Run(tt.chType, func(t *testing.T) {
			assert.Equal(t, tt.want, parseDecScale(tt.chType))
		})
	}
}

func TestTableConstName(t *testing.T) {
	assert.Equal(t, "libp2pGossipsubBeaconBlockTableName", tableConstName("libp2pGossipsubBeaconBlockRow"))
	assert.Equal(t, "fooTableName", tableConstName("fooRow"))
	assert.Equal(t, "fooBarTableName", tableConstName("fooBar"))
}

func TestBatchTypeName(t *testing.T) {
	assert.Equal(t, "libp2pGossipsubBeaconBlockBatch", batchTypeName("libp2pGossipsubBeaconBlockRow"))
	assert.Equal(t, "fooBatch", batchTypeName("fooRow"))
	assert.Equal(t, "fooBarBatch", batchTypeName("fooBar"))
}

func TestNullableInnerGoType(t *testing.T) {
	tests := []struct {
		fieldType string
		want      string
	}{
		{"*proto.ColNullable[uint32]", "uint32"},
		{"*proto.ColNullable[string]", "string"},
		{"*proto.ColNullable[float64]", "float64"},
		{"*proto.ColNullable[time.Time]", "time.Time"},
		{"*proto.ColNullable[proto.IPv6]", "proto.IPv6"},
		{"*proto.ColNullable[[]byte]", "[]byte"},
		// Not a nullable type — falls back to string.
		{"proto.ColStr", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldType, func(t *testing.T) {
			assert.Equal(t, tt.want, nullableInnerGoType(tt.fieldType))
		})
	}
}

func TestBaseTypeName(t *testing.T) {
	assert.Equal(t, "FixedString", baseTypeName("FixedString(66)"))
	assert.Equal(t, "DateTime64", baseTypeName("DateTime64(3, 'UTC')"))
	assert.Equal(t, "Nullable", baseTypeName("Nullable(UInt32)"))
	assert.Equal(t, "String", baseTypeName("String"))
	assert.Equal(t, "UInt32", baseTypeName("UInt32"))
}

func TestIsNumericGoType(t *testing.T) {
	for _, typ := range []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64"} {
		assert.True(t, isNumericGoType(typ), typ)
	}

	for _, typ := range []string{"string", "bool", "time.Time", "proto.IPv6"} {
		assert.False(t, isNumericGoType(typ), typ)
	}
}
