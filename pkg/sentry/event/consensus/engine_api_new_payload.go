package consensus

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// EngineAPINewPayloadData contains the data for an engine_newPayload call event.
type EngineAPINewPayloadData struct {
	// Timing
	RequestedAt time.Time
	DurationMs  uint64

	// Beacon context
	Slot            uint64
	BlockRoot       string
	ParentBlockRoot string
	ProposerIndex   uint64

	// Execution payload
	BlockNumber uint64
	BlockHash   string
	ParentHash  string
	GasUsed     uint64
	GasLimit    uint64
	TxCount     uint32
	BlobCount   uint32

	// Response
	Status          string
	LatestValidHash string
	ValidationError string

	// Meta
	MethodVersion string
}

// EngineAPINewPayload is an event that represents timing data for an engine_newPayload call.
type EngineAPINewPayload struct {
	log            logrus.FieldLogger
	now            time.Time
	data           *EngineAPINewPayloadData
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

// NewEngineAPINewPayload creates a new EngineAPINewPayload event.
func NewEngineAPINewPayload(
	log logrus.FieldLogger,
	data *EngineAPINewPayloadData,
	now time.Time,
	beacon *ethereum.BeaconNode,
	duplicateCache *ttlcache.Cache[string, time.Time],
	clientMeta *xatu.ClientMeta,
) *EngineAPINewPayload {
	return &EngineAPINewPayload{
		log:            log.WithField("event", "CONSENSUS_ENGINE_API_NEW_PAYLOAD"),
		now:            now,
		data:           data,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

// Decorate decorates the event with additional metadata and returns a DecoratedEvent.
func (e *EngineAPINewPayload) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_ConsensusEngineApiNewPayload{
			ConsensusEngineApiNewPayload: &xatu.ConsensusEngineAPINewPayload{
				RequestedAt:     timestamppb.New(e.data.RequestedAt),
				DurationMs:      wrapperspb.UInt64(e.data.DurationMs),
				Slot:            wrapperspb.UInt64(e.data.Slot),
				BlockRoot:       e.data.BlockRoot,
				ParentBlockRoot: e.data.ParentBlockRoot,
				ProposerIndex:   wrapperspb.UInt64(e.data.ProposerIndex),
				BlockNumber:     wrapperspb.UInt64(e.data.BlockNumber),
				BlockHash:       e.data.BlockHash,
				ParentHash:      e.data.ParentHash,
				GasUsed:         wrapperspb.UInt64(e.data.GasUsed),
				GasLimit:        wrapperspb.UInt64(e.data.GasLimit),
				TxCount:         wrapperspb.UInt32(e.data.TxCount),
				BlobCount:       wrapperspb.UInt32(e.data.BlobCount),
				Status:          e.data.Status,
				LatestValidHash: e.data.LatestValidHash,
				ValidationError: e.data.ValidationError,
				MethodVersion:   e.data.MethodVersion,
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get additional engine_newPayload data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_ConsensusEngineApiNewPayload{
			ConsensusEngineApiNewPayload: additionalData,
		}
	}

	return decoratedEvent, nil
}

// ShouldIgnore checks if the event should be ignored based on duplicate detection.
// We use a hash of the event data as the cache key since each unique call should only be reported once.
func (e *EngineAPINewPayload) ShouldIgnore(ctx context.Context) (bool, error) {
	hash, err := hashstructure.Hash(e.data, hashstructure.FormatV2, nil)
	if err != nil {
		return true, fmt.Errorf("failed to hash event data: %w", err)
	}

	cacheKey := fmt.Sprint(hash)

	existing := e.duplicateCache.Get(cacheKey)
	if existing != nil {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"slot":                  e.data.Slot,
			"block_hash":            e.data.BlockHash,
			"time_since_first_seen": time.Since(existing.Value()),
		}).Debug("Ignoring duplicate engine_newPayload event")

		return true, nil
	}

	e.duplicateCache.Set(cacheKey, e.now, ttlcache.DefaultTTL)

	return false, nil
}

func (e *EngineAPINewPayload) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalConsensusEngineAPINewPayloadData, error) {
	extra := &xatu.ClientMeta_AdditionalConsensusEngineAPINewPayloadData{}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(e.data.Slot)
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(e.data.Slot)

	extra.Slot = &xatu.SlotV2{
		Number:        wrapperspb.UInt64(slot.Number()),
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        wrapperspb.UInt64(epoch.Number()),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
