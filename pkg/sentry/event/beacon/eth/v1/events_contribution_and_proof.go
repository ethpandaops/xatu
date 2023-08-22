package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/altair"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type EventsContributionAndProof struct {
	log logrus.FieldLogger

	now time.Time

	event          *altair.SignedContributionAndProof
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsContributionAndProof(log logrus.FieldLogger, event *altair.SignedContributionAndProof, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsContributionAndProof {
	return &EventsContributionAndProof{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsContributionAndProof) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsContributionAndProofV2{
			EthV1EventsContributionAndProofV2: &xatuethv1.EventContributionAndProofV2{
				Signature: xatuethv1.TrimmedString(xatuethv1.BLSSignatureToString(&e.event.Signature)),
				Message: &xatuethv1.ContributionAndProofV2{
					AggregatorIndex: &wrapperspb.UInt64Value{Value: uint64(e.event.Message.AggregatorIndex)},
					SelectionProof:  xatuethv1.TrimmedString(xatuethv1.BLSSignatureToString(&e.event.Message.SelectionProof)),
					Contribution: &xatuethv1.SyncCommitteeContributionV2{
						Slot:              &wrapperspb.UInt64Value{Value: uint64(e.event.Message.Contribution.Slot)},
						SubcommitteeIndex: &wrapperspb.UInt64Value{Value: e.event.Message.Contribution.SubcommitteeIndex},
						AggregationBits:   xatuethv1.BytesToString(e.event.Message.Contribution.AggregationBits.Bytes()),
						Signature:         xatuethv1.TrimmedString(xatuethv1.BLSSignatureToString(&e.event.Message.Contribution.Signature)),
						BeaconBlockRoot:   xatuethv1.RootAsString(e.event.Message.Contribution.BeaconBlockRoot),
					},
				},
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra contribution and proof data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsContributionAndProofV2{
			EthV1EventsContributionAndProofV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsContributionAndProof) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.DefaultTTL)
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
		}).Debug("Duplicate contribution and proof event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsContributionAndProof) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsContributionAndProofV2Data, error) {
	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Message.Contribution.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(e.event.Message.Contribution.Slot))

	extra := &xatu.ClientMeta_AdditionalEthV1EventsContributionAndProofV2Data{
		Contribution: &xatu.ClientMeta_AdditionalEthV1EventsContributionAndProofContributionV2Data{
			Slot: &xatu.SlotV2{
				Number:        &wrapperspb.UInt64Value{Value: slot.Number()},
				StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
			},
			Epoch: &xatu.EpochV2{
				Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
				StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
			},
			Propagation: &xatu.PropagationV2{
				SlotStartDiff: &wrapperspb.UInt64Value{
					Value: uint64(e.now.Sub(slot.TimeWindow().Start()).Milliseconds()),
				},
			},
		},
	}

	return extra, nil
}
