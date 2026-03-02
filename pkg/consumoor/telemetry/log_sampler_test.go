package telemetry

import (
	"testing"
	"time"
)

func TestLogSampler_FirstCallAlwaysAllowed(t *testing.T) {
	s := NewLogSampler(time.Hour) // large interval so nothing expires

	ok, suppressed := s.Allow("key1")
	if !ok {
		t.Fatal("first call should be allowed")
	}

	if suppressed != 0 {
		t.Fatalf("first call should have 0 suppressed, got %d", suppressed)
	}
}

func TestLogSampler_SuppressesWithinInterval(t *testing.T) {
	s := NewLogSampler(time.Hour)

	s.Allow("key1") // first — allowed

	ok, _ := s.Allow("key1")
	if ok {
		t.Fatal("second call within interval should be suppressed")
	}

	ok, _ = s.Allow("key1")
	if ok {
		t.Fatal("third call within interval should be suppressed")
	}
}

func TestLogSampler_DifferentKeysIndependent(t *testing.T) {
	s := NewLogSampler(time.Hour)

	s.Allow("key1")

	ok, suppressed := s.Allow("key2")
	if !ok {
		t.Fatal("different key should be allowed")
	}

	if suppressed != 0 {
		t.Fatalf("different key should have 0 suppressed, got %d", suppressed)
	}
}

func TestLogSampler_ReportsSuppressedCount(t *testing.T) {
	s := NewLogSampler(50 * time.Millisecond)

	s.Allow("key1")

	// Suppress 3 occurrences.
	s.Allow("key1")
	s.Allow("key1")
	s.Allow("key1")

	// Wait for interval to expire.
	time.Sleep(60 * time.Millisecond)

	ok, suppressed := s.Allow("key1")
	if !ok {
		t.Fatal("call after interval should be allowed")
	}

	if suppressed != 3 {
		t.Fatalf("expected 3 suppressed, got %d", suppressed)
	}

	// Next call within new interval should suppress again.
	ok, _ = s.Allow("key1")
	if ok {
		t.Fatal("call within new interval should be suppressed")
	}
}

func TestLogSampler_SuppressedCountResets(t *testing.T) {
	s := NewLogSampler(50 * time.Millisecond)

	s.Allow("key1")
	s.Allow("key1") // +1 suppressed

	time.Sleep(60 * time.Millisecond)

	// Drains the suppressed count.
	ok, suppressed := s.Allow("key1")
	if !ok {
		t.Fatal("should be allowed")
	}

	if suppressed != 1 {
		t.Fatalf("expected 1 suppressed, got %d", suppressed)
	}

	// Immediately suppress again — count should be fresh.
	s.Allow("key1")

	time.Sleep(60 * time.Millisecond)

	ok, suppressed = s.Allow("key1")
	if !ok {
		t.Fatal("should be allowed")
	}

	if suppressed != 1 {
		t.Fatalf("expected 1 suppressed after reset, got %d", suppressed)
	}
}
