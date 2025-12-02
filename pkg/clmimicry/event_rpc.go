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

// Map of RPC event types to Xatu event types.
var rpcToXatuEventMap = map[string]string{
	TraceEvent_HANDLE_METADATA: xatu.Event_LIBP2P_TRACE_HANDLE_METADATA.String(),
	TraceEvent_HANDLE_STATUS:   xatu.Event_LIBP2P_TRACE_HANDLE_STATUS.String(),
	TraceEvent_CUSTODY_PROBE:   xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE.String(),
}

// handleHermesRPCEvent handles Request/Response (RPC) protocol events.
func (p *Processor) handleHermesRPCEvent(
	ctx context.Context,
	event *TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	// Map libp2p event to Xatu event.
	xatuEvent, err := mapRPCEventToXatuEvent(event.Type)
	if err != nil {
		p.log.WithField("event", event.Type).Tracef("unsupported event in handleHermesRPCEvent event")

		//nolint:nilerr // we don't want to return an error here.
		return nil
	}

	switch xatuEvent {
	case xatu.Event_LIBP2P_TRACE_HANDLE_METADATA.String():
		if !p.events.HandleMetadataEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleHandleMetadataEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_HANDLE_STATUS.String():
		if !p.events.HandleStatusEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleHandleStatusEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE.String():
		if !p.events.CustodyProbeEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleCustodyProbeEvent(ctx, clientMeta, traceMeta, event)
	}

	return nil
}

func (p *Processor) handleHandleMetadataEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *TraceEvent,
) error {
	data, err := TraceEventToHandleMetadata(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to handle metadata event")
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

func (p *Processor) handleHandleStatusEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *TraceEvent,
) error {
	data, err := TraceEventToHandleStatus(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to handle status event")
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
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *TraceEvent,
) error {
	data, err := TraceEventToCustodyProbe(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to custody probe event")
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
	event *TraceEvent,
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

func mapRPCEventToXatuEvent(event string) (string, error) {
	if xatuEvent, exists := rpcToXatuEventMap[event]; exists {
		return xatuEvent, nil
	}

	return "", fmt.Errorf("unknown libp2p rpc event: %s", event)
}
