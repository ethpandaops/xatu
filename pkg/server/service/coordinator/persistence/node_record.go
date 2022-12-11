package persistence

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/sirupsen/logrus"
)

const preparedStatement = `
INSERT INTO node_record (
	enr,
	signature,
	seq,
	created_timestamp,
	id,
	secp256k1,
	ip4,
	ip6,
	tcp4,
	tcp6,
	udp4,
	udp6,
	eth2,
	attnets,
	syncnets,
	node_id,
	peer_id
) VALUES (
	$1, $2, $3, now(), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) ON CONFLICT (enr) DO NOTHING;
`

type NodeRecordExporter struct {
	log    logrus.FieldLogger
	client *Client
}

func (e *Client) upsertNodeRecords(ctx context.Context, records []*node.Record) error {
	txn, err := e.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		rErr := txn.Rollback()
		if rErr != nil {
			err = rErr
		}
	}()

	stmt, err := txn.Prepare(preparedStatement)
	if err != nil {
		return err
	}

	for _, record := range records {
		_, err = stmt.Exec(
			record.Enr,
			record.Signature,
			record.Seq,
			record.ID,
			record.Secp256k1,
			record.IP4,
			record.IP6,
			record.TCP4,
			record.TCP6,
			record.UDP4,
			record.UDP6,
			record.ETH2,
			record.Attnets,
			record.Syncnets,
			record.NodeID,
			record.PeerID,
		)
		if err != nil {
			return err
		}
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
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
	return e.client.upsertNodeRecords(ctx, items)
}
