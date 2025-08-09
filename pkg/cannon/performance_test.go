package cannon

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/cannon/mocks"
	"github.com/stretchr/testify/assert"
)

// BenchmarkMockCreation benchmarks the performance of creating mock objects
func BenchmarkMockCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timeProvider := &mocks.MockTimeProvider{}
		ntpClient := &mocks.MockNTPClient{}
		scheduler := &mocks.MockScheduler{}
		sink := mocks.NewMockSink("test", "stdout")

		// Use the mocks to prevent optimization
		_ = timeProvider
		_ = ntpClient
		_ = scheduler
		_ = sink
	}
}

// BenchmarkTimeProviderOperations benchmarks time provider operations
func BenchmarkTimeProviderOperations(b *testing.B) {
	timeProvider := &mocks.MockTimeProvider{}
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider.SetCurrentTime(fixedTime)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := timeProvider.Now()
		_ = now
	}
}

// BenchmarkNTPClientOperations benchmarks NTP client operations
func BenchmarkNTPClientOperations(b *testing.B) {
	ntpClient := &mocks.MockNTPClient{}
	ntpResponse := &mocks.MockNTPResponse{}
	ntpResponse.SetClockOffset(10 * time.Millisecond)

	ntpClient.On("Query", "time.google.com").Return(ntpResponse, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response, err := ntpClient.Query("time.google.com")
		if err != nil {
			b.Fatal(err)
		}
		_ = response.ClockOffset()
	}
}

// BenchmarkSchedulerOperations benchmarks scheduler operations
func BenchmarkSchedulerOperations(b *testing.B) {
	scheduler := &mocks.MockScheduler{}
	scheduler.On("Start").Return()
	scheduler.On("IsStarted").Return(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.Start()
		started := scheduler.IsStarted()
		_ = started
	}
}

// BenchmarkSinkOperations benchmarks sink operations
func BenchmarkSinkOperations(b *testing.B) {
	sink := mocks.NewMockSink("test", "stdout")
	ctx := context.Background()

	sink.On("Start", ctx).Return(nil)
	sink.On("IsStarted").Return(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sink.Start(ctx)
		if err != nil {
			b.Fatal(err)
		}
		started := sink.IsStarted()
		_ = started
	}
}

// BenchmarkConcurrentMockAccess benchmarks concurrent access to mocks
func BenchmarkConcurrentMockAccess(b *testing.B) {
	timeProvider := &mocks.MockTimeProvider{}
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider.SetCurrentTime(fixedTime)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			now := timeProvider.Now()
			_ = now
		}
	})
}

// TestMemoryUsage tests memory usage patterns
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Measure baseline memory
	runtime.GC()
	var baselineStats runtime.MemStats
	runtime.ReadMemStats(&baselineStats)

	// Create many mock objects
	const numObjects = 1000
	timeProviders := make([]*mocks.MockTimeProvider, numObjects)
	ntpClients := make([]*mocks.MockNTPClient, numObjects)
	schedulers := make([]*mocks.MockScheduler, numObjects)
	sinks := make([]*mocks.MockSink, numObjects)

	for i := 0; i < numObjects; i++ {
		timeProviders[i] = &mocks.MockTimeProvider{}
		ntpClients[i] = &mocks.MockNTPClient{}
		schedulers[i] = &mocks.MockScheduler{}
		sinks[i] = mocks.NewMockSink("test", "stdout")
	}

	// Measure memory after creating objects
	runtime.GC()
	var afterStats runtime.MemStats
	runtime.ReadMemStats(&afterStats)

	// Calculate memory usage (handle potential underflow)
	var memoryUsed, memoryPerObject uint64
	if afterStats.Alloc > baselineStats.Alloc {
		memoryUsed = afterStats.Alloc - baselineStats.Alloc
		memoryPerObject = memoryUsed / numObjects
	} else {
		// Memory was possibly freed by GC, use current allocation
		memoryUsed = afterStats.Alloc
		memoryPerObject = memoryUsed / numObjects
	}

	t.Logf("Created %d mock objects", numObjects)
	t.Logf("Baseline memory: %d bytes", baselineStats.Alloc)
	t.Logf("After memory: %d bytes", afterStats.Alloc)
	t.Logf("Total memory used: %d bytes", memoryUsed)
	t.Logf("Memory per object: %d bytes", memoryPerObject)

	// Verify reasonable memory usage (less than 100KB per object, allowing for GC variance)
	assert.Less(t, memoryPerObject, uint64(100*1024), "Memory usage per object should be reasonable")

	// Use the objects to prevent optimization
	for i := 0; i < numObjects; i++ {
		_ = timeProviders[i]
		_ = ntpClients[i]
		_ = schedulers[i]
		_ = sinks[i]
	}
}

// TestGoroutineLeaks tests for goroutine leaks
func TestGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	// Measure baseline goroutines
	baselineGoroutines := runtime.NumGoroutine()

	// Create and use mock objects
	const numIterations = 100
	for i := 0; i < numIterations; i++ {
		timeProvider := &mocks.MockTimeProvider{}
		ntpClient := &mocks.MockNTPClient{}
		scheduler := &mocks.MockScheduler{}
		sink := mocks.NewMockSink("test", "stdout")

		// Use the mocks
		fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		timeProvider.SetCurrentTime(fixedTime)
		_ = timeProvider.Now()

		ntpResponse := &mocks.MockNTPResponse{}
		ntpResponse.SetClockOffset(10 * time.Millisecond)
		ntpClient.On("Query", "test").Return(ntpResponse, nil)
		_, _ = ntpClient.Query("test")

		scheduler.On("Start").Return()
		scheduler.Start()

		ctx := context.Background()
		sink.On("Start", ctx).Return(nil)
		_ = sink.Start(ctx)
	}

	// Give time for goroutines to finish
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Measure final goroutines
	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Baseline goroutines: %d", baselineGoroutines)
	t.Logf("Final goroutines: %d", finalGoroutines)
	t.Logf("Goroutine difference: %d", finalGoroutines-baselineGoroutines)

	// Verify no significant goroutine leaks (allow for small variance)
	assert.Less(t, finalGoroutines-baselineGoroutines, 10, "Should not have significant goroutine leaks")
}

// TestStressOperations performs stress testing of mock operations
func TestStressOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	timeProvider := &mocks.MockTimeProvider{}
	ntpClient := &mocks.MockNTPClient{}
	scheduler := &mocks.MockScheduler{}

	// Set up mocks for many operations
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider.SetCurrentTime(fixedTime)

	ntpResponse := &mocks.MockNTPResponse{}
	ntpResponse.SetClockOffset(10 * time.Millisecond)
	ntpClient.On("Query", "time.google.com").Return(ntpResponse, nil).Times(10000)

	scheduler.On("Start").Return().Times(10000)

	// Perform stress operations
	const numOperations = 10000
	for i := 0; i < numOperations; i++ {
		// Time operations
		_ = timeProvider.Now()

		// NTP operations
		response, err := ntpClient.Query("time.google.com")
		assert.NoError(t, err)
		_ = response.ClockOffset()

		// Scheduler operations
		scheduler.Start()
	}

	// Verify all expectations were met
	timeProvider.AssertExpectations(t)
	ntpClient.AssertExpectations(t)
	scheduler.AssertExpectations(t)
}

// TestConcurrentStressOperations tests concurrent stress operations
func TestConcurrentStressOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	timeProvider := &mocks.MockTimeProvider{}
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider.SetCurrentTime(fixedTime)

	const numGoroutines = 10
	const numOperationsPerGoroutine = 1000

	done := make(chan bool, numGoroutines)

	// Start multiple goroutines performing operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				now := timeProvider.Now()
				assert.Equal(t, fixedTime, now)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	timeProvider.AssertExpectations(t)
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		timeProvider := &mocks.MockTimeProvider{}
		fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		timeProvider.SetCurrentTime(fixedTime)
		now := timeProvider.Now()
		_ = now
	}
}
