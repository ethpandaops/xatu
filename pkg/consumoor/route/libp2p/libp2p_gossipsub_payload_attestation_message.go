package libp2p

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TODO: Add the xatu.Event_* name(s) that route events to the libp2p_gossipsub_payload_attestation_message table.
var libp2pGossipsubPayloadAttestationMessageEventNames = []xatu.Event_Name{}

func init() {
	route, err := route.NewStaticRoute(
		libp2pGossipsubPayloadAttestationMessageTableName,
		libp2pGossipsubPayloadAttestationMessageEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubPayloadAttestationMessageBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(route); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubPayloadAttestationMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	// TODO: Implement this method to flatten the event into columnar batch columns.
	// The generated .gen.go file contains the available column fields for this table.
	//
	// Typical structure:
	//   b.appendRuntime(event)
	//   b.appendMetadata(event)
	//   b.appendPayload(event)
	//   b.rows++
	//   return nil
	return fmt.Errorf("libp2pGossipsubPayloadAttestationMessage: FlattenTo not implemented")
}
