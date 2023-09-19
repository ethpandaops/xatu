package node

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/sirupsen/logrus"
)

type Record struct {
	log         logrus.FieldLogger
	config      *Config
	persistence *persistence.Client

	proc *processor.BatchItemProcessor[node.Record]
}

func NewRecord(ctx context.Context, log logrus.FieldLogger, conf *Config, p *persistence.Client) (*Record, error) {
	return &Record{
		log:         log.WithField("component", "node_record"),
		config:      conf,
		persistence: p,
	}, nil
}

func (r *Record) Start(ctx context.Context) error {
	exporter, err := NewRecordExporter(r.log, r.persistence)
	if err != nil {
		return err
	}

	r.proc, err = processor.NewBatchItemProcessor[node.Record](exporter,
		xatu.ImplementationLower()+"_coordinator_node_record",
		r.log,
		processor.WithMaxQueueSize(r.config.MaxQueueSize),
		processor.WithBatchTimeout(r.config.BatchTimeout),
		processor.WithExportTimeout(r.config.ExportTimeout),
		processor.WithMaxExportBatchSize(r.config.MaxExportBatchSize),
		processor.WithShippingMethod(processor.ShippingMethodAsync),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Record) Stop(ctx context.Context) error {
	if r.proc != nil {
		if err := r.proc.Shutdown(ctx); err != nil {
			r.log.WithError(err).Error("failed to shutdown processor")
		}
	}

	r.log.Info("Component stopped")

	return nil
}

func (r *Record) Write(ctx context.Context, record *node.Record) {
	if err := r.proc.Write(ctx, []*node.Record{record}); err != nil {
		r.log.WithError(err).Error("failed to write record")
	}
}
