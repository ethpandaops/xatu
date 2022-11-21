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

func (s *Sentry) handleChainReOrg(ctx context.Context, event *v1.ChainReorgEvent) error {
	s.log.Debug("ChainReorg received")

	hash, err := hashstructure.Hash(event, hashstructure.FormatV2, nil)
	if err != nil {
		return err
	}

	item, retrieved := s.duplicateCache.ChainReorg.GetOrSet(fmt.Sprint(hash), time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		s.log.WithFields(logrus.Fields{
			"hash":                   hash,
			"time_since_first_event": time.Since(item.Value()),
			"slot":                   event.Slot,
		}).Debug("Duplicate chain reorg event received")
		// TODO(savid): add metrics
		return nil
	}

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_EthV1ChainReorg{
			EthV1ChainReorg: &xatuethv1.EventChainReorg{
				Slot:         uint64(event.Slot),
				Epoch:        uint64(event.Epoch),
				OldHeadBlock: xatuethv1.RootAsString(event.OldHeadBlock),
				OldHeadState: xatuethv1.RootAsString(event.OldHeadState),
				NewHeadBlock: xatuethv1.RootAsString(event.NewHeadBlock),
				NewHeadState: xatuethv1.RootAsString(event.NewHeadState),
				Depth:        event.Depth,
			},
		},
	}

	additionalData, err := s.getChainReorgData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_ChainReorg{
			ChainReorg: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

//nolint:dupl // Not worth refactoring to save a few lines.
func (s *Sentry) getChainReorgData(ctx context.Context, event *v1.ChainReorgEvent, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalChainReorgData, error) {
	extra := &xatu.ClientMeta_AdditionalChainReorgData{}

	slot := s.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(event.Slot))
	epoch := s.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(event.Slot))

	extra.Slot = &xatu.Slot{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.Propagation{
		SlotStartDiff: uint64(meta.Event.DateTime.AsTime().Sub(slot.TimeWindow().Start()).Milliseconds()),
	}

	return extra, nil
}
