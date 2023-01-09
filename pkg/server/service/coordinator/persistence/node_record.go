package persistence

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/huandu/go-sqlbuilder"
	"github.com/sirupsen/logrus"
)

var nodeRecordStruct = sqlbuilder.NewStruct(new(node.Record)).For(sqlbuilder.PostgreSQL)

type NodeRecordExporter struct {
	log    logrus.FieldLogger
	client *Client
}

func (e *Client) InsertNodeRecords(ctx context.Context, records []*node.Record) error {
	values := make([]interface{}, len(records))
	for i, record := range records {
		values[i] = record
	}

	sb := nodeRecordStruct.InsertInto("node_record", values...)
	sql, args := sb.Build()
	sql += " ON CONFLICT (enr) DO NOTHING;"
	_, err := e.db.Exec(sql, args...)

	return err
}

func NewNodeRecordExporter(client *Client, log logrus.FieldLogger) (*NodeRecordExporter, error) {
	return &NodeRecordExporter{
		client: client,
		log:    log,
	}, nil
}

func (e NodeRecordExporter) ExportItems(ctx context.Context, items []*node.Record) error {
	e.log.WithField("items", len(items)).Info("Sending batch of node records to db")

	if err := e.sendUpstream(ctx, items); err != nil {
		return err
	}

	return nil
}

func (e NodeRecordExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *NodeRecordExporter) sendUpstream(ctx context.Context, items []*node.Record) error {
	return e.client.InsertNodeRecords(ctx, items)
}

func (e *Client) UpdateNodeRecord(ctx context.Context, record *node.Record) error {
	sb := nodeRecordStruct.Update("node_record", record)
	sb.Where(sb.E("enr", record.Enr))

	sql, args := sb.Build()

	_, err := e.db.Exec(sql, args...)

	return err
}

func (e *Client) CheckoutStalledExecutionNodeRecords(ctx context.Context, limit int) ([]*node.Record, error) {
	sb := nodeRecordStruct.SelectFrom("node_record")

	sb.Where("eth2 IS NULL")
	sb.Where("consecutive_dial_attempts < 100")
	sb.Where("(last_dial_time < now() - interval '30 minute' OR last_dial_time IS NULL)")
	sb.Where("(last_connect_time < now() - interval '1 day' OR last_connect_time IS NULL)")

	sb.OrderBy("last_dial_time DESC")
	sb.Limit(limit)

	sql, args := sb.Build()

	rows, err := e.db.Query(sql, args...)
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

	_, err = e.db.Exec(sql, args...)
	if err != nil {
		return nil, err
	}

	return records, nil
}
