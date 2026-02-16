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
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10})
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11})
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 1, Offset: 5})

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

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10})
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 11})

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
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 10})

	// Flush fails — batch 1 goes back to pending
	cc.commit()

	// Track batch 2 (arrived while batch 1 was being retried)
	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 20})

	// Fix writer, commit again — both batches should commit
	writer.setFlushErr(nil)
	cc.commit()

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, int64(21), marks[0].offset) // highwater is 20+1
}

func TestCommitCoordinator_MultiTopicPartition(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Track(&sarama.ConsumerMessage{Topic: "topicA", Partition: 0, Offset: 100})
	cc.Track(&sarama.ConsumerMessage{Topic: "topicA", Partition: 1, Offset: 200})
	cc.Track(&sarama.ConsumerMessage{Topic: "topicB", Partition: 0, Offset: 50})
	cc.Track(&sarama.ConsumerMessage{Topic: "topicA", Partition: 0, Offset: 105})

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

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 2, Offset: 0})
	cc.commit()

	marks := session.getMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, "t1", marks[0].topic)
	assert.Equal(t, int32(2), marks[0].partition)
	assert.Equal(t, int64(1), marks[0].offset) // 0+1
	assert.Equal(t, 1, session.commitCount())
}

func TestCommitCoordinator_StopFinalFlush(t *testing.T) {
	writer := &mockWriter{}
	session := &mockSession{}
	cc := newTestCoordinator(writer, session, time.Hour)

	cc.Start()

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 42})

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

	cc.Track(&sarama.ConsumerMessage{Topic: "t1", Partition: 0, Offset: 1})

	// Wait for at least one interval tick to fire
	require.Eventually(t, func() bool {
		return writer.flushCount() > 0
	}, 2*time.Second, 10*time.Millisecond)

	cc.Stop()

	assert.GreaterOrEqual(t, writer.flushCount(), 1)
	assert.GreaterOrEqual(t, session.commitCount(), 1)
}
