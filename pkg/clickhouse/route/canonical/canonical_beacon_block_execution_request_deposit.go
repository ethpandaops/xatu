package canonical

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TODO: Add the xatu.Event_* name(s) that route events to the canonical_beacon_block_execution_request_deposit table.
var canonicalBeaconBlockExecutionRequestDepositEventNames = []xatu.Event_Name{}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionRequestDepositTableName,
		canonicalBeaconBlockExecutionRequestDepositEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionRequestDepositBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionRequestDepositBatch) FlattenTo(
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
	return fmt.Errorf("canonicalBeaconBlockExecutionRequestDeposit: FlattenTo not implemented")
}
