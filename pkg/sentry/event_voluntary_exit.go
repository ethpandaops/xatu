package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleVoluntaryExit(ctx context.Context, event *phase0.VoluntaryExit) error {
	if err := s.beacon.Synced(ctx); err != nil {
		return nil
	}

	hash, err := hashstructure.Hash(event, hashstructure.FormatV2, nil)
	if err != nil {
		return err
	}

	item, retrieved := s.duplicateCache.VoluntaryExit.GetOrSet(fmt.Sprint(hash), time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		s.log.WithFields(logrus.Fields{
			"hash":                   hash,
			"time_since_first_event": time.Since(item.Value()),
			"epoch":                  event.Epoch,
		}).Debug("Duplicate voluntary exit event received")
		// TODO(savid): add metrics
		return nil
	}

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_EthV1VoluntaryExit{
			EthV1VoluntaryExit: &xatuethv1.EventVoluntaryExit{
				Epoch:          uint64(event.Epoch),
				ValidatorIndex: uint64(event.ValidatorIndex),
			},
		},
	}

	additionalData, err := s.getVoluntaryExitData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra voluntary exit data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_VoluntaryExit{
			VoluntaryExit: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (s *Sentry) getVoluntaryExitData(ctx context.Context, event *phase0.VoluntaryExit, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalVoluntaryExitData, error) {
	extra := &xatu.ClientMeta_AdditionalVoluntaryExitData{}

	epoch := s.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(event.Epoch))

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
