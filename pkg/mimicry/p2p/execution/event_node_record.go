package execution

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/noderecord"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// createNodeRecordEvent converts the status captured during the eth handshake
// into a NODE_RECORD_EXECUTION event, matching the shape emitted by the
// discovery module so both feed the same pipeline.
func (p *Peer) createNodeRecordEvent(ctx context.Context, status *xatu.ExecutionNodeStatus) (*xatu.DecoratedEvent, error) {
	meta, err := p.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	capabilities := make([]string, 0, len(status.GetCapabilities()))
	for _, capability := range status.GetCapabilities() {
		capabilities = append(capabilities, fmt.Sprintf("%s/%d", capability.GetName(), capability.GetVersion()))
	}

	executionData := &noderecord.Execution{
		Enr:             &wrapperspb.StringValue{Value: status.GetNodeRecord()},
		Timestamp:       timestamppb.New(now),
		Name:            &wrapperspb.StringValue{Value: status.GetName()},
		Capabilities:    &wrapperspb.StringValue{Value: strings.Join(capabilities, ",")},
		ProtocolVersion: &wrapperspb.StringValue{Value: fmt.Sprintf("%d", status.GetProtocolVersion())},
		Head:            &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetHead())},
		Genesis:         &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetGenesis())},
		ForkIdHash:      &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetForkId().GetHash())},
		ForkIdNext:      &wrapperspb.StringValue{Value: fmt.Sprintf("%d", status.GetForkId().GetNext())},
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_NODE_RECORD_EXECUTION,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_NodeRecordExecution{
			NodeRecordExecution: executionData,
		},
	}, nil
}
