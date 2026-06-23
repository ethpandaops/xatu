package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionAddressAppearancesEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_ADDRESS_APPEARANCES,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionAddressAppearancesTableName,
		canonicalExecutionAddressAppearancesEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionAddressAppearancesBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of address appearances) into
// one row per appearance in canonical_execution_address_appearances.
func (b *canonicalExecutionAddressAppearancesBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalAddressAppearances()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_address_appearances payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, appearance := range payload.GetAddressAppearances() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(appearance.GetBlockNumber())
		b.TransactionHash.Append([]byte(appearance.GetTransactionHash()))
		b.InternalIndex.Append(appearance.GetInternalIndex())
		b.Address.Append(appearance.GetAddress())
		b.Relationship.Append(appearance.GetRelationship())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
