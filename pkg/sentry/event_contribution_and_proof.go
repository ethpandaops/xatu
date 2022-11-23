package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleContributionAndProof(ctx context.Context, event *altair.SignedContributionAndProof) error {
	hash, err := hashstructure.Hash(event, hashstructure.FormatV2, nil)
	if err != nil {
		return err
	}

	item, retrieved := s.duplicateCache.ContributionAndProof.GetOrSet(fmt.Sprint(hash), time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		s.log.WithFields(logrus.Fields{
			"hash":                   hash,
			"time_since_first_event": time.Since(item.Value()),
		}).Debug("Duplicate contribution_and_proof event received")
		// TODO(savid): add metrics
		return nil
	}

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_EthV1ContributionAndProof{
			EthV1ContributionAndProof: &xatuethv1.EventContributionAndProof{
				Signature: xatuethv1.BLSSignatureToString(&event.Signature),
				Message: &xatuethv1.ContributionAndProof{
					AggregatorIndex: uint64(event.Message.AggregatorIndex),
					SelectionProof:  xatuethv1.BLSSignatureToString(&event.Message.SelectionProof),
					Contribution: &xatuethv1.ContributionAndProof_SyncCommitteeContribution{
						Slot:              uint64(event.Message.Contribution.Slot),
						SubcommitteeIndex: uint64(event.Message.Contribution.SubcommitteeIndex),
						AggregationBits:   xatuethv1.BytesToString(event.Message.Contribution.AggregationBits.Bytes()),
						Signature:         xatuethv1.BLSSignatureToString(&event.Message.Contribution.Signature),
						BeaconBlockRoot:   xatuethv1.RootAsString(event.Message.Contribution.BeaconBlockRoot),
					},
				},
			},
		},
	}

	additionalData, err := s.getContributionAndProofData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra voluntary exit data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_ContributionAndProof{
			ContributionAndProof: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (s *Sentry) getContributionAndProofData(ctx context.Context, event *altair.SignedContributionAndProof, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalContributionAndProofData, error) {
	slot := s.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(event.Message.Contribution.Slot))
	epoch := s.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(event.Message.Contribution.Slot))

	eventTime := meta.Event.DateTime.AsTime()

	extra := &xatu.ClientMeta_AdditionalContributionAndProofData{
		Contribution: &xatu.ClientMeta_AdditionalContributionAndProofContributionData{
			Slot: &xatu.Slot{
				Number:        slot.Number(),
				StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
			},
			Epoch: &xatu.Epoch{
				Number:        epoch.Number(),
				StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
			},
			Propagation: &xatu.Propagation{
				SlotStartDiff: uint64(eventTime.Sub(slot.TimeWindow().Start()).Milliseconds()),
			},
		},
	}

	return extra, nil
}
