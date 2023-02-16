package persistence

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/huandu/go-sqlbuilder"
)

var nodeRecordStruct = sqlbuilder.NewStruct(new(node.Record)).For(sqlbuilder.PostgreSQL)

func (c *Client) InsertNodeRecords(ctx context.Context, records []*node.Record) error {
	values := make([]interface{}, len(records))
	for i, record := range records {
		values[i] = record
	}

	sb := nodeRecordStruct.InsertInto("node_record", values...)
	sql, args := sb.Build()
	sql += " ON CONFLICT (enr) DO NOTHING;"
	_, err := c.db.Exec(sql, args...)

	return err
}

func (c *Client) UpdateNodeRecord(ctx context.Context, record *node.Record) error {
	sb := nodeRecordStruct.Update("node_record", record)
	sb.Where(sb.E("enr", record.Enr))

	sql, args := sb.Build()

	_, err := c.db.Exec(sql, args...)

	return err
}

func (c *Client) CheckoutStalledExecutionNodeRecords(ctx context.Context, limit int) ([]*node.Record, error) {
	sb := nodeRecordStruct.SelectFrom("node_record")

	sb.Where("eth2 IS NULL")
	sb.Where("consecutive_dial_attempts < 1000")
	sb.Where("(last_dial_time < now() - interval '1 minute' OR last_dial_time IS NULL)")
	sb.Where("(last_connect_time < now() - interval '1 day' OR last_connect_time IS NULL)")

	sb.OrderBy("last_dial_time DESC")
	sb.Limit(limit)

	sql, args := sb.Build()

	rows, err := c.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var records []*node.Record

	var enrs []interface{}

	for rows.Next() {
		var record node.Record

		err = rows.Scan(nodeRecordStruct.Addr(&record)...)
		if err != nil {
			return nil, err
		}

		records = append(records, &record)

		enrs = append(enrs, record.Enr)
	}

	if len(records) == 0 {
		return records, nil
	}

	// TODO: this should be a transaction, but not a huge deal for now
	ub := nodeRecordStruct.Update("node_record", &node.Record{})
	ub.Set(ub.Add("consecutive_dial_attempts", "1"), ub.Assign("last_dial_time", "now()"))
	ub.Where(ub.In("enr", enrs...))
	sql, args = ub.Build()

	_, err = c.db.Exec(sql, args...)
	if err != nil {
		return nil, err
	}

	return records, nil
}
