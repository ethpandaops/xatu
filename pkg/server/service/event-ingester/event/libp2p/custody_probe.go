package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const TypeLibp2pRPCDataColumnCustodyProbe = "LIBP2P_RPC_DATA_COLUMN_CUSTODY_PROBE"

type DataColumnCustodyProbe struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewDataColumnCustodyProbe(log logrus.FieldLogger, event *xatu.DecoratedEvent) *DataColumnCustodyProbe {
	return &DataColumnCustodyProbe{
		log:   log.WithField("event_type", TypeLibp2pRPCDataColumnCustodyProbe),
		event: event,
	}
}

func (e *DataColumnCustodyProbe) Type() string {
	return TypeLibp2pRPCDataColumnCustodyProbe
}

func (e *DataColumnCustodyProbe) Validate(ctx context.Context) error {
	probe, ok := e.event.GetData().(*xatu.DecoratedEvent_Libp2PRpcDataColumnCustodyProbe)
	if !ok {
		return errors.New("failed to cast event data to custody probe")
	}

	data := probe.Libp2PRpcDataColumnCustodyProbe

	// Validate required fields
	if data.GetPeerId() == "" {
		return errors.New("peer_id is required")
	}

	if data.GetProbeTime() == nil {
		return errors.New("probe_time is required")
	}

	return nil
}

func (e *DataColumnCustodyProbe) Filter(ctx context.Context) bool {
	return false
}

func (e *DataColumnCustodyProbe) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
