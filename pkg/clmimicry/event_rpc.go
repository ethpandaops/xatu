package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleMetadataEvent(ctx context.Context,
	event *HandleMetadataEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.HandleMetadataEnabled {
		return nil
	}

	// Build protobuf message directly from typed event
	data := &libp2p.HandleMetadata{
		PeerId:     wrapperspb.String(event.PeerID.String()),
		ProtocolId: wrapperspb.String(string(event.ProtocolID)),
		Latency:    wrapperspb.Float(float32(event.LatencyS)),
		Metadata: &libp2p.Metadata{
			SeqNumber:         wrapperspb.UInt64(event.SeqNumber),
			Attnets:           wrapperspb.String(event.Attnets),
			Syncnets:          wrapperspb.String(event.Syncnets),
			CustodyGroupCount: wrapperspb.UInt64(event.CustodyGroupCount),
		},
		Direction: wrapperspb.String(event.Direction),
	}

	if event.Error != "" {
		data.Error = wrapperspb.String(event.Error)
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceHandleMetadata{
		Libp2PTraceHandleMetadata: &xatu.ClientMeta_AdditionalLibP2PTraceHandleMetadataData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_HANDLE_METADATA,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceHandleMetadata{
			Libp2PTraceHandleMetadata: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleStatusEvent(ctx context.Context,
	event *HandleStatusEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.HandleStatusEnabled {
		return nil
	}

	// Build protobuf message directly from typed event
	data := &libp2p.HandleStatus{
		PeerId:     wrapperspb.String(event.PeerID.String()),
		ProtocolId: wrapperspb.String(string(event.ProtocolID)),
		Latency:    wrapperspb.Float(float32(event.LatencyS)),
		Direction:  wrapperspb.String(event.Direction),
	}

	if event.Request != nil {
		data.Request = &libp2p.Status{
			ForkDigest:     wrapperspb.String(event.Request.ForkDigest),
			FinalizedRoot:  wrapperspb.String(event.Request.FinalizedRoot),
			FinalizedEpoch: wrapperspb.UInt64(event.Request.FinalizedEpoch),
			HeadRoot:       wrapperspb.String(event.Request.HeadRoot),
			HeadSlot:       wrapperspb.UInt64(event.Request.HeadSlot),
		}
	}

	if event.Response != nil {
		data.Response = &libp2p.Status{
			ForkDigest:     wrapperspb.String(event.Response.ForkDigest),
			FinalizedRoot:  wrapperspb.String(event.Response.FinalizedRoot),
			FinalizedEpoch: wrapperspb.UInt64(event.Response.FinalizedEpoch),
			HeadRoot:       wrapperspb.String(event.Response.HeadRoot),
			HeadSlot:       wrapperspb.UInt64(event.Response.HeadSlot),
		}
	}

	if event.Error != "" {
		data.Error = wrapperspb.String(event.Error)
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceHandleStatus{
		Libp2PTraceHandleStatus: &xatu.ClientMeta_AdditionalLibP2PTraceHandleStatusData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_HANDLE_STATUS,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceHandleStatus{
			Libp2PTraceHandleStatus: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleCustodyProbeEvent(ctx context.Context,
	event *CustodyProbeEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.CustodyProbeEnabled {
		return nil
	}

	// Build protobuf message directly from typed event
	//nolint:gosec // G115: int -> uint32 conversions are safe for slot/epoch/index values
	data := &libp2p.DataColumnCustodyProbe{
		JobStartTimestamp: timestamppb.New(event.JobStartTimestamp),
		PeerId:            wrapperspb.String(event.PeerIDStr),
		Slot:              wrapperspb.UInt32(uint32(event.Slot)),
		Epoch:             wrapperspb.UInt32(uint32(event.Epoch)),
		ColumnIndex:       wrapperspb.UInt32(uint32(event.ColumnIndex)),
		Result:            wrapperspb.String(event.Result),
		ResponseTimeMs:    wrapperspb.Int64(event.DurationMs),
		BeaconBlockRoot:   wrapperspb.String(event.BlockHash),
		ColumnRowsCount:   wrapperspb.UInt32(uint32(event.ColumnSize)),
	}

	if event.Error != "" {
		data.Error = wrapperspb.String(event.Error)
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	extra, err := p.deriveAdditionalDataForCustodyProbeEvent(event, data, traceMeta)
	if err != nil {
		return errors.Wrapf(err, "failed to derive additional data for custody probe event")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcDataColumnCustodyProbe{
		Libp2PTraceRpcDataColumnCustodyProbe: extra,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceRpcDataColumnCustodyProbe{
			Libp2PTraceRpcDataColumnCustodyProbe: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) deriveAdditionalDataForCustodyProbeEvent(
	event *CustodyProbeEvent,
	data *libp2p.DataColumnCustodyProbe,
	traceMeta *libp2p.TraceEventMetadata,
) (*xatu.ClientMeta_AdditionalLibP2PTraceRpcDataColumnCustodyProbeData, error) {
	extra := &xatu.ClientMeta_AdditionalLibP2PTraceRpcDataColumnCustodyProbeData{
		Metadata: traceMeta,
	}

	epoch := p.wallclock.Epochs().FromNumber(uint64(data.GetEpoch().GetValue()))
	slot := p.wallclock.Slots().FromNumber(uint64(data.GetSlot().GetValue()))

	extra.Epoch = &xatu.EpochV2{
		Number:        wrapperspb.UInt64(epoch.Number()),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}
	extra.Slot = &xatu.SlotV2{
		Number:        wrapperspb.UInt64(slot.Number()),
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
	}

	// Calculate the wallclock time when the request was sent.
	requestTime := event.Timestamp.Add(-time.Duration(data.GetResponseTimeMs().GetValue()) * time.Millisecond).Add(p.clockDrift)

	wallclockSlot := p.wallclock.Slots().FromTime(requestTime)
	extra.WallclockSlot = &xatu.SlotV2{
		Number:        wrapperspb.UInt64(wallclockSlot.Number()),
		StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
	}

	wallclockEpoch := p.wallclock.Epochs().FromTime(requestTime)
	extra.WallclockEpoch = &xatu.EpochV2{
		Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
		StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
	}

	return extra, nil
}
