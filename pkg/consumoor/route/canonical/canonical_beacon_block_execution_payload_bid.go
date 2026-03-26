package canonical

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TODO: Add the xatu.Event_* name(s) that route events to the canonical_beacon_block_execution_payload_bid table.
var canonicalBeaconBlockExecutionPayloadBidEventNames = []xatu.Event_Name{}

func init() {
	route, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionPayloadBidTableName,
		canonicalBeaconBlockExecutionPayloadBidEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionPayloadBidBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(route); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionPayloadBidBatch) FlattenTo(
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
	return fmt.Errorf("canonicalBeaconBlockExecutionPayloadBid: FlattenTo not implemented")
}
