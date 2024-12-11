package registrations_test

import (
	"testing"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/registrations"
)

func TestValidatorSetWalker(t *testing.T) {
	t.Run("NewValidatorSetWalker", func(t *testing.T) {
		walker := registrations.NewValidatorSetWalker(0, 4)
		if walker.Shard() != 0 {
			t.Errorf("Expected shard 0, got %d", walker.Shard())
		}
	})

	t.Run("Update and walk through validators", func(t *testing.T) {
		walker := registrations.NewValidatorSetWalker(1, 4) // Shard 1 of 4

		validators := make(map[phase0.ValidatorIndex]*apiv1.Validator)
		for i := 0; i < 100; i++ {
			validators[phase0.ValidatorIndex(i)] = &apiv1.Validator{}
		}

		walker.Update(validators)

		// Check shard bounds
		min := walker.Min()
		max := walker.Max()
		if min != 25 { // 100/4 * 1
			t.Errorf("Expected min 25, got %d", min)
		}
		if max != 50 { // 100/4 * 2
			t.Errorf("Expected max 50, got %d", max)
		}

		// Test walking through validators
		seen := make(map[phase0.ValidatorIndex]bool)
		for i := 0; i < 25; i++ {
			idx, err := walker.Next()
			if err != nil {
				t.Errorf("Error getting next validator index: %v", err)
			}

			if idx < min || idx >= max {
				t.Errorf("Validator index %d outside shard bounds [%d,%d)", idx, min, max)
			}
			seen[idx] = true
		}

		// Should have seen all validators in our shard exactly once
		if len(seen) != 25 {
			t.Errorf("Expected to see 25 unique validators, saw %d", len(seen))
		}
	})

	t.Run("Uneven validator distribution", func(t *testing.T) {
		walker := registrations.NewValidatorSetWalker(0, 3) // First shard of 3

		validators := make(map[phase0.ValidatorIndex]*apiv1.Validator)
		for i := 0; i < 10; i++ { // 10 validators split into 3 shards
			validators[phase0.ValidatorIndex(i)] = &apiv1.Validator{}
		}

		walker.Update(validators)

		// First shard should get 3 validators (floor of 10/3)
		if walker.Min() != 0 {
			t.Errorf("Expected min 0, got %d", walker.Min())
		}
		if walker.Max() != 3 {
			t.Errorf("Expected max 3, got %d", walker.Max())
		}

		// Middle shard
		walker = registrations.NewValidatorSetWalker(1, 3)
		walker.Update(validators)
		if walker.Min() != 3 {
			t.Errorf("Expected min 3, got %d", walker.Min())
		}
		if walker.Max() != 6 {
			t.Errorf("Expected max 6, got %d", walker.Max())
		}

		// Last shard gets remainder
		walker = registrations.NewValidatorSetWalker(2, 3)
		walker.Update(validators)
		if walker.Min() != 6 {
			t.Errorf("Expected min 6, got %d", walker.Min())
		}
		if walker.Max() != 10 { // Gets the remaining 4 validators
			t.Errorf("Expected max 10, got %d", walker.Max())
		}
	})

	t.Run("Last shard handles remainder", func(t *testing.T) {
		walker := registrations.NewValidatorSetWalker(3, 4) // Last shard

		validators := make(map[phase0.ValidatorIndex]*apiv1.Validator)
		for i := 0; i < 102; i++ { // Not evenly divisible by 4
			validators[phase0.ValidatorIndex(i)] = &apiv1.Validator{}
		}

		walker.Update(validators)

		min := walker.Min()
		max := walker.Max()
		if min != 75 {
			t.Errorf("Expected min 75, got %d", min)
		}
		if max != 102 { // Should include remainder
			t.Errorf("Expected max 102, got %d", max)
		}
	})

	t.Run("Marker stays in bounds after update", func(t *testing.T) {
		walker := registrations.NewValidatorSetWalker(0, 2)

		validators := make(map[phase0.ValidatorIndex]*apiv1.Validator)
		for i := 0; i < 10; i++ {
			validators[phase0.ValidatorIndex(i)] = &apiv1.Validator{}
		}

		walker.Update(validators)

		// Walk past halfway point
		for i := 0; i < 6; i++ {
			_, err := walker.Next()
			if err != nil {
				t.Errorf("Error getting next validator index: %v", err)
			}
		}

		// Update with new validator set
		newValidators := make(map[phase0.ValidatorIndex]*apiv1.Validator)
		for i := 0; i < 5; i++ {
			newValidators[phase0.ValidatorIndex(i)] = &apiv1.Validator{}
		}

		walker.Update(newValidators)

		// Marker should still work
		_, err := walker.Next()
		if err != nil {
			t.Errorf("Error getting next validator index: %v", err)
		}
	})
}
