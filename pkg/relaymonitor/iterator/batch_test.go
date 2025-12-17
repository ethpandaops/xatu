package iterator

import (
	"testing"
)

func TestBatchRequest(t *testing.T) {
	t.Run("batch request fields", func(t *testing.T) {
		batch := &BatchRequest{
			CurrentSlot: 1000,
			TargetSlot:  1100,
		}

		batch.Params = make(map[string][]string)
		batch.Params["cursor"] = []string{"1100"}
		batch.Params["limit"] = []string{"200"}

		if batch.CurrentSlot != 1000 {
			t.Errorf("expected CurrentSlot 1000, got %d", batch.CurrentSlot)
		}

		if batch.TargetSlot != 1100 {
			t.Errorf("expected TargetSlot 1100, got %d", batch.TargetSlot)
		}

		if batch.Params.Get("cursor") != "1100" {
			t.Errorf("expected cursor 1100, got %s", batch.Params.Get("cursor"))
		}

		if batch.Params.Get("limit") != "200" {
			t.Errorf("expected limit 200, got %s", batch.Params.Get("limit"))
		}
	})
}

func TestDefaultBatchSize(t *testing.T) {
	if DefaultBatchSize != 200 {
		t.Errorf("expected DefaultBatchSize to be 200, got %d", DefaultBatchSize)
	}
}
