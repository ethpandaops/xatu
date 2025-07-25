package persistence

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/huandu/go-sqlbuilder"
)

var nodeRecordConsensusStruct = sqlbuilder.NewStruct(new(node.Consensus)).For(sqlbuilder.PostgreSQL)

func (c *Client) InsertNodeRecordConsensus(ctx context.Context, record *node.Consensus) error {
	ib := nodeRecordConsensusStruct.InsertInto("node_record_consensus")

	items := []interface{}{
		sqlbuilder.Raw("DEFAULT"),
		record.Enr,
		record.NodeID,
		record.PeerID,
		record.CreateTime,
		record.Name,
		record.ForkDigest,
		record.NextForkDigest,
		record.FinalizedRoot,
		record.FinalizedEpoch,
		record.HeadRoot,
		record.HeadSlot,
		record.CGC,
		record.NetworkID,
	}

	ib.Cols(nodeRecordConsensusStruct.Columns()...).Values(items...)

	sql, args := ib.Build()

	_, err := c.db.ExecContext(ctx, sql, args...)

	if err != nil {
		c.log.WithError(err).Error("failed to insert node record consensus")
	}

	return err
}

func (c *Client) ListNodeRecordConsensus(ctx context.Context, networkIds []uint64, forkDigests [][]byte, limit int) ([]*node.Consensus, error) {
	sb := nodeRecordConsensusStruct.SelectFrom("node_record_consensus")

	if len(networkIds) > 0 {
		nids := make([]interface{}, 0, len(networkIds))
		for _, nid := range networkIds {
			nids = append(nids, nid)
		}

		sb.Where(sb.In("network_id", nids...))
	}

	if len(forkDigests) > 0 {
		fds := make([]interface{}, 0, len(forkDigests))
		for _, fd := range forkDigests {
			fds = append(fds, fd)
		}

		sb.Where(sb.In("fork_digest", fds...))
	}

	sb.OrderBy("create_time DESC")
	sb.Limit(limit)

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var records []*node.Consensus

	for rows.Next() {
		var record node.Consensus

		err = rows.Scan(nodeRecordConsensusStruct.Addr(&record)...)
		if err != nil {
			return nil, err
		}

		records = append(records, &record)
	}

	return records, nil
}

func (c *Client) BulkInsertNodeRecordConsensus(ctx context.Context, records []*node.Consensus) error {
	if len(records) == 0 {
		return nil
	}

	const maxBatchSize = 500 // Conservative batch size to avoid any psql parameter limits.

	// Process in batches if necessary.
	for i := 0; i < len(records); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(records) {
			end = len(records)
		}

		var (
			batch = records[i:end]
			ib    = nodeRecordConsensusStruct.InsertInto("node_record_consensus")
		)

		ib.Cols(nodeRecordConsensusStruct.Columns()...)

		for _, record := range batch {
			values := []interface{}{
				sqlbuilder.Raw("DEFAULT"),
				record.Enr,
				record.NodeID,
				record.PeerID,
				record.CreateTime,
				record.Name,
				record.ForkDigest,
				record.NextForkDigest,
				record.FinalizedRoot,
				record.FinalizedEpoch,
				record.HeadRoot,
				record.HeadSlot,
				record.CGC,
				record.NetworkID,
			}
			ib.Values(values...)
		}

		sql, args := ib.Build()
		if _, err := c.db.ExecContext(ctx, sql, args...); err != nil {
			c.log.WithError(err).Errorf("failed to bulk insert node record consensus batch %d-%d", i, end-1)

			return err
		}
	}

	return nil
}
