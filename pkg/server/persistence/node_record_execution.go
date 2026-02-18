package persistence

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/huandu/go-sqlbuilder"
)

var nodeRecordExecutionStruct = sqlbuilder.NewStruct(new(node.Execution)).For(sqlbuilder.PostgreSQL)

func (c *Client) InsertNodeRecordExecution(ctx context.Context, record *node.Execution) error {
	ib := nodeRecordExecutionStruct.InsertInto("node_record_execution")

	items := []any{
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

	_, err := c.db.ExecContext(ctx, sql, args...)
	if err != nil {
		c.log.WithError(err).Error("failed to insert node record execution")
	}

	return err
}

func (c *Client) ListNodeRecordExecutions(ctx context.Context, networkIds []uint64, forkIDHashes [][]byte, limit int) ([]*node.Execution, error) {
	sb := nodeRecordExecutionStruct.SelectFrom("node_record_execution")

	if len(networkIds) > 0 {
		nids := make([]any, 0, len(networkIds))
		for _, nid := range networkIds {
			nids = append(nids, nid)
		}

		sb.Where(sb.In("network_id", nids...))
	}

	if len(forkIDHashes) > 0 {
		fidhs := make([]any, 0, len(forkIDHashes))
		for _, fidh := range forkIDHashes {
			fidhs = append(fidhs, fidh)
		}

		sb.Where(sb.In("fork_id_hash", fidhs...))
	}

	sb.OrderBy("create_time DESC")
	sb.Limit(limit)

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var records []*node.Execution

	for rows.Next() {
		var record node.Execution

		err = rows.Scan(nodeRecordExecutionStruct.Addr(&record)...)
		if err != nil {
			return nil, err
		}

		records = append(records, &record)
	}

	return records, nil
}
