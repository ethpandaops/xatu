package clmimicry

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// sentryShardingYAML mirrors the topic patterns a production sentry runs with.
const sentryShardingYAML = `
topics:
  "LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_RPC_META_CONTROL_IHAVE*:.*block.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_HANDLE_STATUS": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_HANDLE_METADATA": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_RPC_META_CONTROL_GRAFT*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_RPC_META_CONTROL_PRUNE*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_JOIN*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_PRUNE*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_LEAVE*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_GRAFT*:.*": {totalShards: 1, activeShards: ["0"]}
  "LIBP2P_TRACE_RECV_RPC": {totalShards: 512, activeShards: ["0-511"]}
  "LIBP2P_TRACE_SEND_RPC": {totalShards: 512, activeShards: ["0-511"]}
  ".*": {totalShards: 512, activeShards: ["0-9"]}
`

func newSentrySharder(tb testing.TB) *UnifiedSharder {
	tb.Helper()

	config := &ShardingConfig{}
	require.NoError(tb, yaml.Unmarshal([]byte(sentryShardingYAML), config))

	sharder, err := NewUnifiedSharder(config, true)
	require.NoError(tb, err)

	return sharder
}

// effectiveShards reduces a config to the fields that drive a decision; equal-rate pattern ties resolve in map order.
func effectiveShards(config *TopicShardingConfig) *TopicShardingConfig {
	if config == nil {
		return nil
	}

	return &TopicShardingConfig{TotalShards: config.TotalShards, ActiveShards: config.ActiveShards}
}

func sentryTopics() []string {
	topics := make([]string, 0, 4+64+128)
	topics = append(topics,
		"",
		"/eth2/12345678/beacon_block/ssz_snappy",
		"/eth2/12345678/beacon_aggregate_and_proof/ssz_snappy",
		"/eth2/12345678/sync_committee_contribution_and_proof/ssz_snappy",
	)

	for i := range 64 {
		topics = append(topics, fmt.Sprintf("/eth2/12345678/beacon_attestation_%d/ssz_snappy", i))
	}

	for i := range 128 {
		topics = append(topics, fmt.Sprintf("/eth2/12345678/data_column_sidecar_%d/ssz_snappy", i))
	}

	return topics
}

func TestFindTopicConfigCachedMatchesUncached(t *testing.T) {
	sharder := newSentrySharder(t)

	eventTypes := make([]xatu.Event_Name, 0, len(xatu.Event_Name_name))
	for value := range xatu.Event_Name_name {
		eventTypes = append(eventTypes, xatu.Event_Name(value))
	}

	for _, topic := range sentryTopics() {
		for _, eventType := range eventTypes {
			want := effectiveShards(sharder.matchTopicConfig(topic, eventType))

			// First call populates the cache, second call is served from it.
			assert.Equal(t, want, effectiveShards(sharder.findTopicConfig(topic, eventType)), "%s %s", eventType, topic)
			assert.Equal(t, want, effectiveShards(sharder.findTopicConfig(topic, eventType)), "%s %s", eventType, topic)
		}
	}
}

func TestFindTopicConfigCachesMisses(t *testing.T) {
	config := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			"LIBP2P_TRACE_RPC_META_CONTROL_IHAVE:.*beacon_block.*": {TotalShards: 1, ActiveShards: []uint64{0}},
		},
	}

	sharder, err := NewUnifiedSharder(config, true)
	require.NoError(t, err)

	topic := "/eth2/12345678/beacon_attestation_1/ssz_snappy"

	assert.Nil(t, sharder.findTopicConfig(topic, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE))
	assert.Nil(t, sharder.findTopicConfig(topic, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE))
	assert.Equal(t, int64(1), sharder.topicConfigCount.Load())
}

func TestFindTopicConfigCacheIsBounded(t *testing.T) {
	sharder := newSentrySharder(t)
	eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE

	for i := range maxTopicConfigCacheEntries + 100 {
		topic := fmt.Sprintf("/eth2/peer-supplied-%d/beacon_block/ssz_snappy", i)

		got := effectiveShards(sharder.findTopicConfig(topic, eventType))
		require.Equal(t, effectiveShards(sharder.matchTopicConfig(topic, eventType)), got)
	}

	assert.Equal(t, int64(maxTopicConfigCacheEntries), sharder.topicConfigCount.Load())

	// Lookups past the bound still resolve correctly.
	topic := "/eth2/12345678/beacon_attestation_7/ssz_snappy"
	assert.Equal(t, effectiveShards(sharder.matchTopicConfig(topic, eventType)), effectiveShards(sharder.findTopicConfig(topic, eventType)))
}

func TestFindTopicConfigConcurrent(t *testing.T) {
	sharder := newSentrySharder(t)
	topics := sentryTopics()
	eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for i := range 2000 {
				topic := topics[i%len(topics)]
				want := effectiveShards(sharder.matchTopicConfig(topic, eventType))

				if !assert.Equal(t, want, effectiveShards(sharder.findTopicConfig(topic, eventType)), topic) {
					return
				}
			}
		})
	}

	wg.Wait()

	assert.Equal(t, int64(len(topics)), sharder.topicConfigCount.Load())
}

func BenchmarkShouldProcessIHave(b *testing.B) {
	sharder := newSentrySharder(b)
	topics := sentryTopics()
	msgIDs := make([]string, 1024)

	for i := range msgIDs {
		msgIDs[i] = fmt.Sprintf("%064x", i*2654435761)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		sharder.ShouldProcess(
			xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE,
			msgIDs[i%len(msgIDs)],
			topics[i%len(topics)],
		)
	}
}
