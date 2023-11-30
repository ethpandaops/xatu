package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/seer/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type TrackedAttestation struct {
	MsgID  string
	Sender string
	Subnet int

	ArrivalTime time.Time
	TimeInSlot  time.Duration

	Slot int64
}

type HostInfo struct {
	IP       string
	ID       string
	Port     int
	PeerInfo struct {
		UserAgent       string
		ProtocolVersion string
		Protocols       []string
		Latency         time.Duration
	}
}

type WrappedAttestation struct {
	Attestation        *phase0.Attestation
	TrackedAttestation *TrackedAttestation
	HostInfo           *HostInfo
}

func (a *WrappedAttestation) AttestationHash() (string, error) {
	hash, err := hashstructure.Hash(a.Attestation, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(hash), nil
}

type AttestationHydrator struct {
	log logrus.FieldLogger

	event *WrappedAttestation

	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, int]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewAttestationHydrator(log logrus.FieldLogger, event *WrappedAttestation, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, int], clientMeta *xatu.ClientMeta) *AttestationHydrator {
	return &AttestationHydrator{
		log:            log.WithField("event", "BEACON_P2P_ATTESTATION"),
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (h *AttestationHydrator) CreateXatuEvent(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_P2P_ATTESTATION,
			DateTime: timestamppb.New(h.event.TrackedAttestation.ArrivalTime), // Take the time from the event
			Id:       h.id.String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta,
		},
		Data: &xatu.DecoratedEvent_BeaconP2PAttestation{
			BeaconP2PAttestation: &xatuethv1.AttestationV2{
				AggregationBits: xatuethv1.BytesToString(h.event.Attestation.AggregationBits),
				Data: &xatuethv1.AttestationDataV2{
					Slot:            &wrapperspb.UInt64Value{Value: uint64(h.event.Attestation.Data.Slot)},
					Index:           &wrapperspb.UInt64Value{Value: uint64(h.event.Attestation.Data.Index)},
					BeaconBlockRoot: xatuethv1.RootAsString(h.event.Attestation.Data.BeaconBlockRoot),
					Source: &xatuethv1.CheckpointV2{
						Epoch: &wrapperspb.UInt64Value{Value: uint64(h.event.Attestation.Data.Source.Epoch)},
						Root:  xatuethv1.RootAsString(h.event.Attestation.Data.Source.Root),
					},
					Target: &xatuethv1.CheckpointV2{
						Epoch: &wrapperspb.UInt64Value{Value: uint64(h.event.Attestation.Data.Target.Epoch)},
						Root:  xatuethv1.RootAsString(h.event.Attestation.Data.Target.Root),
					},
				},
				Signature: xatuethv1.TrimmedString(fmt.Sprintf("%#x", h.event.Attestation.Signature)),
			},
		},
	}

	additionalData, err := h.getAdditionalData(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get extra attestation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_BEACON_P2P_ATTESTATION{
			BEACON_P2P_ATTESTATION: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (h *AttestationHydrator) TimesSeen(ctx context.Context) (int, error) {
	hash, err := h.event.AttestationHash()
	if err != nil {
		return 0, err
	}

	// If we have already seen this attestation 3 times, ignore it
	timesSeenBefore := h.duplicateCache.Get(hash)
	if timesSeenBefore != nil {
		return timesSeenBefore.Value(), nil
	}

	return 0, nil
}

func (h *AttestationHydrator) MarkAttestationAsSeen(ctx context.Context) error {
	hash, err := h.event.AttestationHash()
	if err != nil {
		return err
	}

	timesSeenBefore := h.duplicateCache.Get(hash)
	if timesSeenBefore == nil {
		h.duplicateCache.Set(hash, 1, time.Minute*14)
	} else {
		h.duplicateCache.Set(hash, timesSeenBefore.Value()+1, time.Minute*14)
	}

	return nil
}

func (h *AttestationHydrator) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalBeaconP2PAttestationData, error) {
	extra := &xatu.ClientMeta_AdditionalBeaconP2PAttestationData{
		Peer: &libp2p.Peer{
			Id:              h.event.HostInfo.ID,
			Ip:              h.event.HostInfo.IP,
			Port:            wrapperspb.UInt32(uint32(h.event.HostInfo.Port)),
			UserAgent:       h.event.HostInfo.PeerInfo.UserAgent,
			Protocols:       h.event.HostInfo.PeerInfo.Protocols,
			ProtocolVersion: h.event.HostInfo.PeerInfo.ProtocolVersion,
			Latency:         wrapperspb.UInt64(uint64(h.event.HostInfo.PeerInfo.Latency.Milliseconds())),
		},
		Subnet:    wrapperspb.UInt32(uint32(h.event.TrackedAttestation.Subnet)),
		Validated: wrapperspb.Bool(false),
	}

	attestionSlot := h.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(h.event.Attestation.Data.Slot))
	epoch := h.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(h.event.Attestation.Data.Slot))

	extra.Slot = &xatu.SlotV2{
		Number:        &wrapperspb.UInt64Value{Value: attestionSlot.Number()},
		StartDateTime: timestamppb.New(attestionSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.PropagationV2{
		SlotStartDiff: &wrapperspb.UInt64Value{
			Value: uint64(h.event.TrackedAttestation.ArrivalTime.Sub(attestionSlot.TimeWindow().Start()).Milliseconds()),
		},
	}

	// Build out the target section
	targetEpoch := h.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(h.event.Attestation.Data.Target.Epoch))
	extra.Target = &xatu.ClientMeta_AdditionalEthV1AttestationTargetV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := h.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(h.event.Attestation.Data.Source.Epoch))
	extra.Source = &xatu.ClientMeta_AdditionalEthV1AttestationSourceV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: sourceEpoch.Number()},
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	// If the attestation is unaggreated, we can append the validator position within the committee
	if h.event.Attestation.AggregationBits.Count() == 1 {
		position := uint64(h.event.Attestation.AggregationBits.BitIndices()[0])

		validatorIndex, err := h.beacon.Duties().GetValidatorIndex(
			phase0.Epoch(epoch.Number()),
			h.event.Attestation.Data.Slot,
			h.event.Attestation.Data.Index,
			position,
		)
		if err == nil {
			extra.AttestingValidator = &xatu.AttestingValidatorV2{
				CommitteeIndex: &wrapperspb.UInt64Value{Value: position},
				Index:          &wrapperspb.UInt64Value{Value: uint64(validatorIndex)},
			}
		}
	}

	return extra, nil
}
