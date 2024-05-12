package clmimicry

import (
	"context"
	"fmt"
	"time"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/probe-lab/hermes/host"
	ethtypes "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *Mimicry) handleGossipDataColumnSidecar(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent, payload map[string]any) error {
	// Extract sidecar data
	eSidecar, ok := payload["Sidecar"].(*ethtypes.DataColumnSidecar)
	if !ok {
		return fmt.Errorf("invalid sidecar")
	}

	dataColumn := ""
	for _, bytes := range eSidecar.DataColumn {
		dataColumn += string(bytes)
	}

	kzgCommitments := make([]string, len(eSidecar.KzgCommitments))
	for i, bytes := range eSidecar.KzgCommitments {
		kzgCommitments[i] = fmt.Sprintf("0x%x", bytes)
	}

	kzgProof := ""
	for _, bytes := range eSidecar.KzgProof {
		kzgProof += string(bytes)
	}

	kzgCommitmentsInclusionProof := ""
	for _, bytes := range eSidecar.KzgCommitmentsInclusionProof {
		kzgCommitmentsInclusionProof += string(bytes)
	}

	signedBlockHeader := &xatuethv1.SignedBeaconBlockHeaderV2{
		Message: &xatuethv1.BeaconBlockHeaderV2{
			Slot:          wrapperspb.UInt64(uint64(eSidecar.SignedBlockHeader.GetHeader().GetSlot())),
			ProposerIndex: wrapperspb.UInt64(uint64(eSidecar.SignedBlockHeader.GetHeader().GetProposerIndex())),
			ParentRoot:    fmt.Sprintf("%x", eSidecar.SignedBlockHeader.GetHeader().GetParentRoot()),
			StateRoot:     fmt.Sprintf("%x", eSidecar.SignedBlockHeader.GetHeader().GetStateRoot()),
			BodyRoot:      fmt.Sprintf("%x", eSidecar.SignedBlockHeader.GetHeader().GetBodyRoot()),
		},
		Signature: fmt.Sprintf("%x", eSidecar.SignedBlockHeader.Signature),
	}

	sidecar := &gossipsub.DataColumnSidecar{
		ColumnIndex:                  wrapperspb.UInt64(eSidecar.ColumnIndex),
		DataColumn:                   dataColumn,
		KzgCommitments:               kzgCommitments,
		KzgProof:                     kzgProof,
		SignedBlockHeader:            signedBlockHeader,
		KzgCommitmentsInclusionProof: kzgCommitmentsInclusionProof,
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubDataColumnSidecarData(ctx, payload, eSidecar)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBeaconDataColumnSidecar{
		Libp2PTraceGossipsubBeaconDataColumnSidecar: additionalData,
	}

	timestamp, ok := payload["Timestamp"].(time.Time)
	if !ok {
		return fmt.Errorf("invalid timestamp")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
			DateTime: timestamppb.New(timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconDataColumnSidecar{
			Libp2PTraceGossipsubBeaconDataColumnSidecar: sidecar,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) createAdditionalGossipSubDataColumnSidecarData(ctx context.Context,
	payload map[string]any,
	sidecar *ethtypes.DataColumnSidecar,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconDataColumnSidecarData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().Now()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	timestamp, ok := payload["Timestamp"].(time.Time)
	if !ok {
		return nil, fmt.Errorf("invalid timestamp")
	}

	// Add Clock Drift
	timestampAdjusted := timestamp.Add(m.clockDrift)

	sidecarSlot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(sidecar.SignedBlockHeader.GetHeader().GetSlot()))
	epoch := m.ethereum.Metadata().Wallclock().Epochs().FromSlot(uint64(sidecar.SignedBlockHeader.GetHeader().GetSlot()))

	extra := &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconDataColumnSidecarData{
		WallclockSlot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: wallclockSlot.Number()},
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: wallclockEpoch.Number()},
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: sidecarSlot.Number()},
			StartDateTime: timestamppb.New(sidecarSlot.TimeWindow().Start()),
		},
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: &wrapperspb.UInt64Value{
				Value: uint64(timestampAdjusted.Sub(sidecarSlot.TimeWindow().Start()).Milliseconds()),
			},
		},
	}

	peerID, ok := payload["PeerID"].(string)
	if ok {
		extra.Metadata = &libp2p.TraceEventMetadata{
			PeerId: wrapperspb.String(peerID),
		}
	}

	topic, ok := payload["Topic"].(string)
	if ok {
		extra.Topic = wrapperspb.String(topic)
	}

	msgID, ok := payload["MsgID"].(string)
	if ok {
		extra.MessageId = wrapperspb.String(msgID)
	}

	msgSize, ok := payload["MsgSize"].(int)
	if ok {
		extra.MessageSize = wrapperspb.UInt32(uint32(msgSize))
	}

	return extra, nil
}
