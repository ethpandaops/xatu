package relaymonitor

import (
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// Set and Has used to access the underlying map without any synchronization.
// Multiple relays each get their first Set call from their own goroutine in
// production (one per configured slot-time offset, plus the consistency
// coordinator), so concurrent first-inserts into the shared map were a real
// race. Run with -race to verify there is no unsynchronized access.
func TestDuplicateBidCache_ConcurrentSetAndHas(t *testing.T) {
	cache := NewDuplicateBidCache(time.Minute)

	relays := []string{"flashbots", "bloxroute-max-profit", "bloxroute-regulated", "ultrasound", "agnostic-relay"}

	var wg sync.WaitGroup

	for i, relay := range relays {
		relay := relay
		slot := phase0.Slot(i) //nolint:gosec // test data

		wg.Add(2)

		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				cache.Set(relay, slot, "0xhash")
			}
		}()

		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				cache.Has(relay, slot, "0xhash")
			}
		}()
	}

	wg.Wait()

	for i, relay := range relays {
		if !cache.Has(relay, phase0.Slot(i), "0xhash") { //nolint:gosec // test data
			t.Fatalf("expected Has to return true for relay %s after Set", relay)
		}
	}
}

// Two relays racing on their very first Set call is the specific scenario
// that used to trigger a concurrent map write in production during startup.
func TestDuplicateBidCache_ConcurrentFirstInsert(t *testing.T) {
	cache := NewDuplicateBidCache(time.Minute)

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		relay := "relay"
		i := i

		wg.Add(1)

		go func() {
			defer wg.Done()

			cache.Set(relay, phase0.Slot(i), "0xhash") //nolint:gosec // test data
		}()
	}

	wg.Wait()
}
