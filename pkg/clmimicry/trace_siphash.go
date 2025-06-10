package clmimicry

import (
	"encoding/binary"
)

// SipHash implements the SipHash-2-4 algorithm, a fast and efficient hash function
// designed for message authentication and hash-table lookups.
//
// Key features of SipHash:
// - Deterministic output for identical inputs
// - Even distribution of outputs across the range of uint64
//
// When used for sharding (via GetShard), SipHash provides:
// - Consistent distribution of messages across shards
// - Deterministic routing where the same message always maps to the same shard
//
// Parameters:
//   - key: A 16-byte secret key (can be fixed for consistent sharding)
//   - data: The message bytes to hash
//
// Returns:
//   - A 64-bit unsigned integer hash value
//
// References:
//   - Reference implementation: https://github.com/veorq/SipHash
//   - ClickHouse documentation: https://clickhouse.com/docs/sql-reference/functions/hash-functions#siphash64
func SipHash(key [16]byte, data []byte) uint64 {
	// SipHash constants, init the internal state of SipHash.
	// Together, they form 'somepseudorandomlygeneratedbytes'.
	//
	// If you were to change these constants, you'd essentially be creating a variant of SipHash that's no
	// longer compatible with the standard implementation.
	//
	// For example:
	// python: https://github.com/zacharyvoase/python-csiphash/blob/master/siphash24.c#L77
	// c: https://github.com/veorq/SipHash/blob/master/siphash.c#L89
	// go: https://github.com/dchest/siphash/blob/722c5ea451891c5b35d1bd5c1a4628ed4ff96920/hash.go#L15
	v0 := uint64(0x736f6d6570736575) // somepseud
	v1 := uint64(0x646f72616e646f6d) // dorandom
	v2 := uint64(0x6c7967656e657261) // lygenera
	v3 := uint64(0x7465646279746573) // tedbytes

	// Load key.
	k0 := binary.LittleEndian.Uint64(key[0:8])
	k1 := binary.LittleEndian.Uint64(key[8:16])

	// Initialize state.
	v0 ^= k0
	v1 ^= k1
	v2 ^= k0
	v3 ^= k1

	// Process data in 8-byte blocks.
	dataLen := len(data)
	b := uint64(dataLen) << 56

	fullBlocks := dataLen / 8

	// Process full blocks.
	for i := 0; i < fullBlocks; i++ {
		m := binary.LittleEndian.Uint64(data[i*8:])
		v3 ^= m

		// Compression function (2 rounds).
		sipRound(&v0, &v1, &v2, &v3)
		sipRound(&v0, &v1, &v2, &v3)

		v0 ^= m
	}

	// Process remaining bytes.
	offset := fullBlocks * 8
	for i := offset; i < dataLen; i++ {
		b |= uint64(data[i]) << uint((i-offset)*8) //nolint:gosec // This is safe.
	}

	// Finalization.
	v3 ^= b

	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)

	v0 ^= b

	// Final mixing.
	v2 ^= 0xff

	// 4 finalization rounds.
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)
	sipRound(&v0, &v1, &v2, &v3)

	return v0 ^ v1 ^ v2 ^ v3
}

// GetShard calculates which shard a message belongs to based on its ID.
//
// This function uses SipHash to consistently map message IDs (typically hashes themselves)
// to specific shards, ensuring even distribution across the available shards.
//
// Key benefits:
// - Deterministic: The same message ID always maps to the same shard
// - Balanced: Messages are evenly distributed across all shards
//
// Parameters:
//   - shardingKey: The identifier to use for sharding (often a hash like "0x1234...abcd")
//   - totalShards: The total number of available shards (e.g., 64)
//
// Returns:
//   - The shard number (0 to totalShards-1) where this message should be processed
func GetShard(shardingKey string, totalShards uint64) uint64 {
	// Use a fixed key for consistent hashing.
	key := [16]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}

	// Calculate hash.
	hash := SipHash(key, []byte(shardingKey))

	// Return shard number.
	return hash % totalShards
}

// sipRound performs one round of SipHash's compression function.
//
// Each round of sipRound transforms the internal state through a series
// of bitwise operations and rotations to increase diffusion (spreading
// of input bits across the state).
func sipRound(v0, v1, v2, v3 *uint64) {
	*v0 += *v1
	*v1 = (*v1 << 13) | (*v1 >> (64 - 13))
	*v1 ^= *v0
	*v0 = (*v0 << 32) | (*v0 >> (64 - 32))

	*v2 += *v3
	*v3 = (*v3 << 16) | (*v3 >> (64 - 16))
	*v3 ^= *v2

	*v0 += *v3
	*v3 = (*v3 << 21) | (*v3 >> (64 - 21))
	*v3 ^= *v0

	*v2 += *v1
	*v1 = (*v1 << 17) | (*v1 >> (64 - 17))
	*v1 ^= *v2
	*v2 = (*v2 << 32) | (*v2 >> (64 - 32))
}
