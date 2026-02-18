package clmimicry_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ethpandaops/xatu/pkg/clmimicry"
	"github.com/stretchr/testify/assert"
)

// TestSipHash verifies that our SipHash implementation produces the expected output values
// for a set of known inputs. This ensures the hash function works correctly.
//
// SipHash is a hash function that takes a message of any size and produces a fixed-size
// 64-bit output that appears random but is deterministic for the same input.
// It's used for sharding messages across multiple buckets.
//
// Tests borrowed from the reference implementation:
// - https://github.com/veorq/SipHash/blob/master/main.c
func TestSipHash(t *testing.T) {
	// Use a consistent key for all test cases.
	// In SipHash, the key is the secret sauce that influences the output.
	key := [16]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}

	// Each test case provides an input byte array and its expected output hash.
	// If our SipHash implementation is correct, it should produce exactly these values.
	testCases := []struct {
		input    []byte
		expected uint64
	}{
		{[]byte{}, 0x726fdb47dd0e0e31},                                                     // Empty input
		{[]byte{0x00}, 0x74f839c593dc67fd},                                                 // Single zero byte
		{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}, 0xab0200f58b01d137},             // 7 bytes
		{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, 0x93f5f5799a932462},       // 8 bytes (full block)
		{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, 0x9e0082df0ba9e4b0}, // 9 bytes (full block plus 1)
	}

	// Run each test case and compare the actual output with the expected output.
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Vector-%d", i), func(t *testing.T) {
			result := clmimicry.SipHash(key, tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetShard verifies that our sharding function correctly distributes messages
// across the available shards (buckets) in a balanced and consistent way.
//
// 1. Always map the same message to the same shard.
// 2. Distribute messages evenly across all shards.
// 3. Work with different numbers of shards (in our case, configurable per msg topic).
// 4. Handle edge cases like very long or empty messages.
func TestGetShard(t *testing.T) {
	const totalShards uint64 = 64

	// First, check that messages are distributed evenly across all shards.
	t.Run("Distribution", func(t *testing.T) {
		// Keep track of how many messages end up in each shard.
		shardCounts := make(map[uint64]int)

		// Generate 6400 different message IDs (100 per shard).
		// For each one, check which shard it's assigned to and count them.
		for i := range 6400 {
			// Create a unique hash-like ID for each test case.
			msgID := generateHashMsgID("test-distribution", i)
			shard := clmimicry.GetShard(msgID, totalShards)

			// Make sure we never get an invalid shard number.
			assert.Less(t, shard, totalShards, "Shard number should be less than total shards")

			// Count how many messages go to each shard.
			shardCounts[shard]++
		}

		// Verify that all 64 shards are being used.
		// If any shards have zero messages, something is wrong with the distribution.
		assert.Equal(t, int(totalShards), len(shardCounts), "All shards should receive some messages")

		// Check that each shard gets roughly the same number of messages.
		// With 6400 messages and 64 shards, each should get about 100 messages.
		// We allow a 25% variance (75-126 messages per shard) to account for
		// natural variation in distribution.
		for shard, count := range shardCounts {
			assert.Greater(t, count, 75, "Shard %d has too few items: %d", shard, count)
			assert.Less(t, count, 126, "Shard %d has too many items: %d", shard, count)
		}
	})

	// Check that the same message always maps to the same shard.
	// This is critical for consistent lookups - we need to know
	// exactly which shard contains a particular message.
	t.Run("Consistency", func(t *testing.T) {
		// Test a variety of realistic hash-like message IDs.
		testIDs := []string{
			// Standard ethereum-style hash IDs of different values.
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"0xe7f6c011776e8db7cd330b54174fd76f7d0216b612387a5ffcfb81e6f0919683",
			"0x7d87c5ea75f7b21e7fd118cba9ff5ae17aba7260470782e4d6df1eec7a8c4695",
			// Edge case: all zeros.
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			// Edge case: all ones.
			"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		}

		// For each message ID, calculate its shard twice.
		// Both calculations should always give the same result.
		for _, msgID := range testIDs {
			shard1 := clmimicry.GetShard(msgID, totalShards)
			shard2 := clmimicry.GetShard(msgID, totalShards)
			assert.Equal(t, shard1, shard2, "Same message ID should always map to same shard")
		}
	})

	// Test that our sharding works with different numbers of shards.
	t.Run("DifferentTotalShards", func(t *testing.T) {
		// Use a consistent message ID for all tests.
		var (
			msgID       = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			shardCounts = []uint64{1, 2, 4, 8, 16, 32, 64, 128, 256}
		)

		// For each shard count, verify we get a valid shard number.
		for _, count := range shardCounts {
			shard := clmimicry.GetShard(msgID, count)

			// The shard number should always be between 0 and (count-1).
			assert.Less(t, shard, count, "Shard %d is not less than total shards %d", shard, count)
		}
	})

	// Test that our sharding handles unusual message formats correctly.
	t.Run("EdgeCases", func(t *testing.T) {
		// Empty string edge case.
		var (
			emptyMsg   = ""
			emptyShard = clmimicry.GetShard(emptyMsg, totalShards)
		)

		assert.Less(t, emptyShard, totalShards, "Empty message should map to a valid shard")

		// Very long message ID edge case (simulates a huge transaction ID or similar).
		var longMsg strings.Builder
		longMsg.WriteString("0x")

		for range 1000 {
			longMsg.WriteString("1234567890abcdef")
		}

		longShard := clmimicry.GetShard(longMsg.String(), totalShards)

		assert.Less(t, longShard, totalShards, "Long message should map to a valid shard")
	})

	// Test specifically with 64 buckets to validate our production scenario.
	t.Run("64BucketsSpecific", func(t *testing.T) {
		// Create a test message for each possible shard.
		for i := range 64 {
			// Generate a unique message ID for each test.
			msgID := generateHashMsgID("bucket-test", i)
			shard := clmimicry.GetShard(msgID, 64)

			// Verify the shard number is valid (0-63).
			assert.Less(t, shard, uint64(64), "Shard should be between 0 and 63")
		}
	})

	// Test with real-world eth hash formats.
	t.Run("RealWorldHashFormats", func(t *testing.T) {
		// Block hashes.
		ethBlockHashes := []string{
			"0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3",
			"0x88e96d4537bea4d9c05d12549907b32561d3bf31f45aae734cdc119f13406cb6",
			"0xb495a1d7e6663152ae92708da4843337b958146015a2802f4193a410044698c9",
		}

		// Transaction hashes.
		txHashes := []string{
			"0x5c504ed432cb51138bcf09aa5e8a410dd4a1e204ef84bfed1be16dfba1b22060",
			"0xc55e2b90168af6972193c1f86fa4d7d7b31a29c156665d15b9cd48618b5177ef",
		}

		allHashes := append(ethBlockHashes, txHashes...)

		// Verify each maps to a valid shard.
		for _, hash := range allHashes {
			shard := clmimicry.GetShard(hash, totalShards)
			assert.Less(t, shard, totalShards, "Hash %s should map to a valid shard", hash)
		}

		// Now generate a large number of eth-style hashes and check
		// their distribution across shards.
		shardDistribution := make(map[uint64]int)

		for i := range 1000 {
			shard := clmimicry.GetShard(fmt.Sprintf("0x%064x", i), totalShards)
			shardDistribution[shard]++
		}

		// We should see usage of at least 90% of available shards
		assert.GreaterOrEqual(
			t,
			len(shardDistribution),
			int(totalShards*9/10),
			"At least 90% of shards should be used with eth-style hashes",
		)
	})
}

// TestIsShardActive tests the IsShardActive function
func TestIsShardActive(t *testing.T) {
	tests := []struct {
		name         string
		shard        uint64
		activeShards []uint64
		expected     bool
	}{
		{
			name:         "shard is active",
			shard:        2,
			activeShards: []uint64{1, 2, 3},
			expected:     true,
		},
		{
			name:         "shard is not active",
			shard:        4,
			activeShards: []uint64{1, 2, 3},
			expected:     false,
		},
		{
			name:         "empty active shards",
			shard:        1,
			activeShards: []uint64{},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clmimicry.IsShardActive(tt.shard, tt.activeShards)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// generateHashMsgID creates a realistic hash-like message ID for testing.
//
// This helper function creates deterministic but realistic message IDs
// by hashing a seed string combined with an index. This simulates how
// real-world hashes would be distributed.
//
// Parameters:
//   - seed: A base string to start with
//   - index: A number to make each ID unique
//
// Returns:
//   - A hex-encoded SHA-256 hash that looks like a typical blockchain hash
func generateHashMsgID(seed string, index int) string {
	// Create a deterministic hash-like string by hashing the seed and index.
	h := sha256.New()
	h.Write([]byte(seed))
	fmt.Fprintf(h, "%d", index)

	return hex.EncodeToString(h.Sum(nil))
}
