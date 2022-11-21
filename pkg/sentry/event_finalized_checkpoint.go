package sentry

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleFinalizedCheckpoint(ctx context.Context, event *v1.FinalizedCheckpointEvent) error {
	s.log.Debug("FinalizedCheckpoint received")

	hash, err := hashstructure.Hash(event, hashstructure.FormatV2, nil)
	if err != nil {
		return err
	}

	item, retrieved := s.duplicateCache.FinalizedCheckpoint.GetOrSet(fmt.Sprint(hash), time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		s.log.WithFields(logrus.Fields{
			"hash":                   hash,
			"time_since_first_event": time.Since(item.Value()),
			"epoch":                  event.Epoch,
		}).Debug("Duplicate finalized checkpoint event received")
		// TODO(savid): add metrics
		return nil
	}

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_EthV1FinalizedCheckpoint{
			EthV1FinalizedCheckpoint: &xatuethv1.EventFinalizedCheckpoint{
				Epoch: uint64(event.Epoch),
				State: xatuethv1.RootAsString(event.State),
				Block: xatuethv1.RootAsString(event.Block),
			},
		},
	}

	additionalData, err := s.getFinalizedCheckpointData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra head data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_FinalizedCheckpoint{
			FinalizedCheckpoint: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (s *Sentry) getFinalizedCheckpointData(ctx context.Context, event *v1.FinalizedCheckpointEvent, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalFinalizedCheckpointData, error) {
	extra := &xatu.ClientMeta_AdditionalFinalizedCheckpointData{}

	epoch := s.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(event.Epoch))

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
