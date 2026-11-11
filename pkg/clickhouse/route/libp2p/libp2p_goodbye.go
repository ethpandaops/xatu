package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGoodbyeEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOODBYE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGoodbyeTableName,
		libp2pGoodbyeEventNames,
		func() route.ColumnarBatch { return newlibp2pGoodbyeBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGoodbyeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceGoodbye()
	if payload == nil {
		return fmt.Errorf("nil libp2p_trace_goodbye payload: %w", route.ErrInvalidEvent)
	}

	if payload.GetRemotePeer() == nil {
		return fmt.Errorf("nil RemotePeer: %w", route.ErrInvalidEvent)
	}

	if payload.GetCode() == nil {
		return fmt.Errorf("nil Code: %w", route.ErrInvalidEvent)
	}

	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.RemotePeerIDUniqueKey.Append(computePeerIDUniqueKey(payload.GetRemotePeer().GetValue(), networkName))
	b.Code.Append(payload.GetCode().GetValue())
	b.Reason.Append(wrappedStringValue(payload.GetReason()))

	agent := parseAgentVersion(wrappedStringValue(payload.GetAgentVersion()))
	b.RemoteAgentImplementation.Append(agent.Implementation)
	b.RemoteAgentVersion.Append(agent.Version)
	b.RemoteAgentVersionMajor.Append(agent.Major)
	b.RemoteAgentVersionMinor.Append(agent.Minor)
	b.RemoteAgentVersionPatch.Append(agent.Patch)
	b.RemoteAgentPlatform.Append(agent.Platform)

	b.appendMetadata(event)
	b.rows++

	return nil
}
