package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// BlobData represents blob metadata derived from block's blob_kzg_commitments.
type BlobData struct {
	Slot            phase0.Slot
	Index           uint64
	BlockRoot       phase0.Root
	BlockParentRoot phase0.Root
	ProposerIndex   phase0.ValidatorIndex
	KZGCommitment   deneb.KZGCommitment
	VersionedHash   deneb.VersionedHash
}

type BeaconBlob struct {
	log logrus.FieldLogger

	now time.Time

	event          *BlobData
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewBeaconBlob(log logrus.FieldLogger, event *BlobData, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *BeaconBlob {
	return &BeaconBlob{
		log:            log.WithField("event", "BEACON_API_ETH_V1_BEACON_BLOB"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *BeaconBlob) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconBlob{
			EthV1BeaconBlob: &xatuethv1.Blob{
				Slot:            &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
				Index:           &wrapperspb.UInt64Value{Value: e.event.Index},
				BlockRoot:       xatuethv1.RootAsString(e.event.BlockRoot),
				BlockParentRoot: xatuethv1.RootAsString(e.event.BlockParentRoot),
				ProposerIndex:   &wrapperspb.UInt64Value{Value: uint64(e.event.ProposerIndex)},
				KzgCommitment:   xatuethv1.KzgCommitmentToString(e.event.KZGCommitment),
				VersionedHash:   xatuethv1.VersionedHashToString(e.event.VersionedHash),
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra beacon blob data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconBlob{
			EthV1BeaconBlob: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *BeaconBlob) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
			"slot":                  e.event.Slot,
			"index":                 e.event.Index,
		}).Debug("Duplicate beacon blob event received")

		return true, nil
	}

	return false, nil
}

func (e *BeaconBlob) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1BeaconBlobData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1BeaconBlobData{}

	blobSlot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

	extra.Slot = &xatu.SlotV2{
		Number:        &wrapperspb.UInt64Value{Value: blobSlot.Number()},
		StartDateTime: timestamppb.New(blobSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
