package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionContractsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_CONTRACTS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionContractsTableName,
		canonicalExecutionContractsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionContractsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of contracts) into one row per
// contract in canonical_execution_contracts.
func (b *canonicalExecutionContractsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalContracts()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_contracts payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, contract := range payload.GetContracts() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(contract.GetBlockNumber())
		b.TransactionHash.Append([]byte(contract.GetTransactionHash()))
		b.InternalIndex.Append(contract.GetInternalIndex())
		b.CreateIndex.Append(contract.GetCreateIndex())
		b.ContractAddress.Append(contract.GetContractAddress())
		b.Deployer.Append(contract.GetDeployer())
		b.Factory.Append(contract.GetFactory())
		b.InitCode.Append(contract.GetInitCode())
		b.Code.Append(ceNullStr(contract.GetCode()))
		b.InitCodeHash.Append(contract.GetInitCodeHash())
		b.NInitCodeBytes.Append(contract.GetNInitCodeBytes())
		b.NCodeBytes.Append(contract.GetNCodeBytes())
		b.CodeHash.Append(contract.GetCodeHash())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
