package execution

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	executionClient "github.com/ethpandaops/xatu/pkg/sentry/execution"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecutionDebugStateSize is an event that represents state size data from the execution layer.
type ExecutionDebugStateSize struct {
	log            logrus.FieldLogger
	now            time.Time
	data           *executionClient.DebugStateSizeResponse
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

// NewExecutionDebugStateSize creates a new ExecutionDebugStateSize event.
func NewExecutionDebugStateSize(
	log logrus.FieldLogger,
	data *executionClient.DebugStateSizeResponse,
	now time.Time,
	duplicateCache *ttlcache.Cache[string, time.Time],
	clientMeta *xatu.ClientMeta,
) *ExecutionDebugStateSize {
	return &ExecutionDebugStateSize{
		log:            log.WithField("event", "EXECUTION_DEBUG_STATE_SIZE"),
		now:            now,
		data:           data,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

// Decorate decorates the event with additional metadata and returns a DecoratedEvent.
func (e *ExecutionDebugStateSize) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_DEBUG_STATE_SIZE,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_ExecutionDebugStateSize{
			ExecutionDebugStateSize: &xatu.ExecutionDebugStateSize{
				AccountBytes:         e.data.AccountBytes,
				AccountTrienodeBytes: e.data.AccountTrienodeBytes,
				AccountTrienodes:     e.data.AccountTrienodes,
				Accounts:             e.data.Accounts,
				BlockNumber:          e.data.BlockNumber,
				ContractCodeBytes:    e.data.ContractCodeBytes,
				ContractCodes:        e.data.ContractCodes,
				StateRoot:            e.data.StateRoot,
				StorageBytes:         e.data.StorageBytes,
				StorageTrienodeBytes: e.data.StorageTrienodeBytes,
				StorageTrienodes:     e.data.StorageTrienodes,
				Storages:             e.data.Storages,
			},
		},
	}

	return decoratedEvent, nil
}

// ShouldIgnore checks if the event should be ignored based on duplicate detection.
// We use state_root as the cache key since each unique state should only be reported once.
func (e *ExecutionDebugStateSize) ShouldIgnore(ctx context.Context) (bool, error) {
	cacheKey := e.data.StateRoot

	// Check if this state root is already in the cache.
	existing := e.duplicateCache.Get(cacheKey)
	if existing != nil {
		e.log.WithFields(logrus.Fields{
			"state_root":            e.data.StateRoot,
			"block_number":          e.data.BlockNumber,
			"time_since_first_seen": time.Since(existing.Value()),
		}).Debug("Ignoring duplicate state size event")

		return true, nil
	}

	// Add this state root to the cache with the default TTL.
	e.duplicateCache.Set(cacheKey, e.now, ttlcache.DefaultTTL)

	return false, nil
}
