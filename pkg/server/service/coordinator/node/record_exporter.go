package node

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/sirupsen/logrus"
)

type RecordExporter struct {
	log         logrus.FieldLogger
	persistence *persistence.Client
}

func NewRecordExporter(log logrus.FieldLogger, p *persistence.Client) (*RecordExporter, error) {
	return &RecordExporter{
		persistence: p,
		log:         log,
	}, nil
}

func (r RecordExporter) ExportItems(ctx context.Context, items []*node.Record) error {
	r.log.WithField("items", len(items)).Info("Sending batch of node records to db")

	if err := r.sendUpstream(ctx, items); err != nil {
		return err
	}

	return nil
}

func (r RecordExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (r *RecordExporter) sendUpstream(ctx context.Context, items []*node.Record) error {
	return r.persistence.InsertNodeRecords(ctx, items)
}
