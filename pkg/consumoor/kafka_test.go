package consumoor

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock Writer ---

type mockWriter struct {
	mu       sync.Mutex
	flushErr error
	flushes  int
}

func (w *mockWriter) Start(_ context.Context) error      { return nil }
func (w *mockWriter) Stop(_ context.Context) error       { return nil }
func (w *mockWriter) Write(_ string, _ []map[string]any) {}

func (w *mockWriter) FlushAll(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flushes++

	return w.flushErr
}

func (w *mockWriter) setFlushErr(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flushErr = err
}

func (w *mockWriter) flushCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.flushes
}

// --- mock ConsumerGroupSession ---

type markCall struct {
	topic     string
	partition int32
	offset    int64
}

type mockSession struct {
	mu      sync.Mutex
	marks   []markCall
	commits int
}

func (s *mockSession) Claims() map[string][]int32                       { return nil }
func (s *mockSession) MemberID() string                                 { return "test" }
func (s *mockSession) GenerationID() int32                              { return 1 }
func (s *mockSession) Context() context.Context                         { return context.Background() }
func (s *mockSession) ResetOffset(_ string, _ int32, _ int64, _ string) {} //nolint:revive // mock
func (s *mockSession) MarkMessage(_ *sarama.ConsumerMessage, _ string)  {}

func (s *mockSession) MarkOffset(topic string, partition int32, offset int64, _ string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.marks = append(s.marks, markCall{topic, partition, offset})
}

func (s *mockSession) Commit() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.commits++
}

func (s *mockSession) getMarks() []markCall {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]markCall, len(s.marks))
	copy(out, s.marks)

	return out
}

func (s *mockSession) commitCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.commits
}

// --- helpers ---

var testMetrics = NewMetrics("test")

func newTestCoordinator(
	writer *mockWriter,
	session *mockSession,
	interval time.Duration,
) *commitCoordinator {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	return newCommitCoordinator(
		log,
		writer,
		testMetrics,
		session,
		interval,
	)
}

// --- tests ---

func TestCommitCoordinator_HappyPath(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour) // long interval — we trigger manually

	// Track some messages
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 1, Offset: 5}, DeliveryStatusDelivered)

	// Trigger commit manually
	cc.commit()

	// Verify: FlushAll called once
	assert.Equal(t, 1, writer.flushCount())

	// Verify: only highwater offsets marked (offset+1)
	marks := session.getMarks()
	assert.Len(t, marks, 2)

	markMap := make(map[string]int64, len(marks))
	for _, m := range marks {
		key := fmt.Sprintf("%s/%d", m.topic, m.partition)
		markMap[key] = m.offset
	}

	assert.Equal(t, int64(12), markMap["t1/0"]) // 11+1
	assert.Equal(t, int64(6), markMap["t1/1"])  // 5+1

	// Verify: session.Commit() called
	assert.Equal(t, 1, session.commitCount())
}

func TestCommitCoordinator_EmptyPending(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.commit()

	// No messages tracked — should be a no-op
	assert.Equal(t, 0, writer.flushCount())
	assert.Empty(t, session.getMarks())
	assert.Equal(t, 0, session.commitCount())
}

func TestCommitCoordinator_FlushAllFailure(t *testing.T) {
	writer := &mockWriter{flushErr: fmt.Errorf("CH down")}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11}, DeliveryStatusDelivered)

	// First commit fails — messages should be put back
	cc.commit()

	assert.Equal(t, 1, writer.flushCount())
	assert.Empty(t, session.getMarks())
	assert.Equal(t, 0, session.commitCount())

	// Fix the writer
	writer.setFlushErr(nil)

	// Retry succeeds — messages from first batch should now commit
	cc.commit()

	assert.Equal(t, 2, writer.flushCount())
	assert.Len(t, session.getMarks(), 1) // only partition 0
	assert.Equal(t, 1, session.commitCount())

	marks := session.getMarks()
	assert.Equal(t, int64(12), marks[0].offset) // highwater 11+1
}

func TestCommitCoordinator_FlushFailurePreservesOrder(t *testing.T) {
	writer := &mockWriter{flushErr: fmt.Errorf("CH down")}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	// Track batch 1
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10}, DeliveryStatusDelivered)

	// Flush fails — batch 1 goes back to pending
	cc.commit()

	// Track batch 2 (arrived while batch 1 was being retried)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 20}, DeliveryStatusDelivered)

	// Fix writer, commit again — both batches should commit
	writer.setFlushErr(nil)
	cc.commit()

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, int64(21), marks[0].offset) // highwater is 20+1
}

func TestCommitCoordinator_PermanentFlushErrorRejectsDeliveredAndCommits(t *testing.T) {
	writer := &mockWriter{
		flushErr: &rowConversionError{
			table:   "canonical_beacon_block",
			skipped: 1,
			cause:   fmt.Errorf("cannot convert column %q", "event_date_time"),
		},
	}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11}, DeliveryStatusDelivered)

	cc.commit()

	assert.Equal(t, 1, writer.flushCount())

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, "t1", marks[0].topic)
	assert.Equal(t, int32(0), marks[0].partition)
	assert.Equal(t, int64(12), marks[0].offset)
	assert.Equal(t, 1, session.commitCount())

	cc.mu.Lock()
	defer cc.mu.Unlock()

	assert.Empty(t, cc.pending)
}

func TestCommitCoordinator_MultiTopicPartition(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "topicA", Partition: 0, Offset: 100}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "topicA", Partition: 1, Offset: 200}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "topicB", Partition: 0, Offset: 50}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "topicA", Partition: 0, Offset: 105}, DeliveryStatusDelivered)

	cc.commit()

	marks := session.getMarks()
	assert.Len(t, marks, 3)

	markMap := make(map[string]int64, len(marks))
	for _, m := range marks {
		key := fmt.Sprintf("%s/%d", m.topic, m.partition)
		markMap[key] = m.offset
	}

	assert.Equal(t, int64(106), markMap["topicA/0"]) // 105+1
	assert.Equal(t, int64(201), markMap["topicA/1"]) // 200+1
	assert.Equal(t, int64(51), markMap["topicB/0"])  // 50+1
}

func TestCommitCoordinator_OffsetZeroIsCommitted(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 2, Offset: 0}, DeliveryStatusDelivered)
	cc.commit()

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, "t1", marks[0].topic)
	assert.Equal(t, int32(2), marks[0].partition)
	assert.Equal(t, int64(1), marks[0].offset) // 0+1
	assert.Equal(t, 1, session.commitCount())
}

func TestCommitCoordinator_RejectedIsCommitEligible(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11}, DeliveryStatusRejected)
	cc.commit()

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, int64(12), marks[0].offset)
	assert.Equal(t, 1, session.commitCount())
}

func TestCommitCoordinator_ErroredBlocksContiguousCommit(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10}, DeliveryStatusDelivered)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11}, DeliveryStatusErrored)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 12}, DeliveryStatusDelivered)

	cc.commit()

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, int64(11), marks[0].offset) // committed only through offset 10
	assert.Equal(t, 1, session.commitCount())

	cc.mu.Lock()
	pending := append([]trackedMsg(nil), cc.pending...)
	cc.mu.Unlock()

	require.Len(t, pending, 2)
	assert.Equal(t, int64(11), pending[0].offset)
	assert.Equal(t, DeliveryStatusErrored, pending[0].status)
	assert.Equal(t, int64(12), pending[1].offset)

	cc.commit()
	assert.Equal(t, 2, writer.flushCount())
	assert.Equal(t, 1, session.commitCount())
}

func TestCommitCoordinator_StopFinalFlush(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Start()

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 42}, DeliveryStatusDelivered)

	// Stop should trigger a final flush+commit
	cc.Stop()

	assert.Equal(t, 1, writer.flushCount())
	assert.Equal(t, 1, session.commitCount())

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, int64(43), marks[0].offset)
}

func TestCommitCoordinator_IntervalTriggers(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, 50*time.Millisecond)

	cc.Start()

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 1}, DeliveryStatusDelivered)

	// Wait for at least one interval tick to fire
	require.Eventually(t, func() bool {
		return writer.flushCount() > 0
	}, 2*time.Second, 10*time.Millisecond)

	cc.Stop()

	assert.GreaterOrEqual(t, writer.flushCount(), 1)
	assert.GreaterOrEqual(t, session.commitCount(), 1)
}

func TestBuildSaramaConfig_SCRAMMechanismsSetClientGenerator(t *testing.T) {
	tests := []struct {
		name      string
		mechanism string
		expected  sarama.SASLMechanism
	}{
		{
			name:      "scram sha256",
			mechanism: "SCRAM-SHA-256",
			expected:  sarama.SASLTypeSCRAMSHA256,
		},
		{
			name:      "scram sha512",
			mechanism: "SCRAM-SHA-512",
			expected:  sarama.SASLTypeSCRAMSHA512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := buildSaramaConfig(&KafkaConfig{
				OffsetDefault:          "oldest",
				FetchMinBytes:          1,
				FetchWaitMaxMs:         500,
				MaxPartitionFetchBytes: 1_048_576,
				SessionTimeoutMs:       30_000,
				HeartbeatIntervalMs:    3_000,
				SASLConfig:             &SASLConfig{Mechanism: tt.mechanism, User: "alice", Password: "secret"},
				CommitInterval:         5 * time.Second,
			})
			require.NoError(t, err)

			assert.True(t, cfg.Net.SASL.Enable)
			assert.Equal(t, tt.expected, cfg.Net.SASL.Mechanism)
			assert.NotNil(t, cfg.Net.SASL.SCRAMClientGeneratorFunc)
			assert.Nil(t, cfg.Net.SASL.TokenProvider)
			assert.NoError(t, cfg.Validate())

			client := cfg.Net.SASL.SCRAMClientGeneratorFunc()
			require.NotNil(t, client)
			assert.NoError(t, client.Begin("alice", "secret", ""))
		})
	}
}

func TestBuildSaramaConfig_OAuthMechanismSetsTokenProvider(t *testing.T) {
	cfg, err := buildSaramaConfig(&KafkaConfig{
		OffsetDefault:          "oldest",
		FetchMinBytes:          1,
		FetchWaitMaxMs:         500,
		MaxPartitionFetchBytes: 1_048_576,
		SessionTimeoutMs:       30_000,
		HeartbeatIntervalMs:    3_000,
		SASLConfig:             &SASLConfig{Mechanism: "OAUTHBEARER", User: "alice", Password: "token-value"},
		CommitInterval:         5 * time.Second,
	})
	require.NoError(t, err)

	assert.True(t, cfg.Net.SASL.Enable)
	assert.Equal(t, string(sarama.SASLTypeOAuth), string(cfg.Net.SASL.Mechanism))
	assert.NotNil(t, cfg.Net.SASL.TokenProvider)
	assert.Nil(t, cfg.Net.SASL.SCRAMClientGeneratorFunc)
	assert.NoError(t, cfg.Validate())

	token, err := cfg.Net.SASL.TokenProvider.Token()
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "token-value", token.Token)
}
