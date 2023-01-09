package persistence

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/huandu/go-sqlbuilder"
)

var nodeRecordExecutionStruct = sqlbuilder.NewStruct(new(node.Execution)).For(sqlbuilder.PostgreSQL)

func (e *Client) InsertNodeRecordExecution(ctx context.Context, record *node.Execution) error {
	ib := nodeRecordExecutionStruct.InsertInto("node_record_execution")

	items := []interface{}{
		sqlbuilder.Raw("DEFAULT"),
		record.Enr,
		time.Now(),
		record.Name,
		record.Capabilities,
		record.ProtocolVersion,
		record.NetworkID,
		record.TotalDifficulty,
		record.Head,
		record.Genesis,
		record.ForkIDHash,
		record.ForkIDNext,
	}

	ib.Cols(nodeRecordExecutionStruct.Columns()...).Values(items...)

	sql, args := ib.Build()

	_, err := e.db.Exec(sql, args...)

	if err != nil {
		e.log.WithError(err).Error("failed to insert node record execution")
	}

	return err
}
