package coordinator

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/persistence"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	ServiceType = "coordinator"
)

type Coordinator struct {
	xatu.UnimplementedCoordinatorServer

	log    logrus.FieldLogger
	config *Config

	persistence *persistence.Client

	metrics *Metrics

	nodeRecordProc *processor.BatchItemProcessor[node.Record]
}

func New(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Coordinator, error) {
	p, err := persistence.New(ctx, log, &conf.Persistence)
	if err != nil {
		return nil, err
	}

	e := &Coordinator{
		log:         log.WithField("server/module", ServiceType),
		config:      conf,
		persistence: p,
		metrics:     NewMetrics("xatu_coordinator"),
	}

	return e, nil
}

func (e *Coordinator) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("starting module")

	xatu.RegisterCoordinatorServer(grpcServer, e)

	err := e.persistence.Start(ctx)
	if err != nil {
		return err
	}

	exporter, err := persistence.NewNodeRecordExporter(e.persistence, e.log)
	if err != nil {
		return err
	}

	e.nodeRecordProc = processor.NewBatchItemProcessor[node.Record](exporter,
		e.log,
		processor.WithMaxQueueSize(e.config.Persistence.MaxQueueSize),
		processor.WithBatchTimeout(e.config.Persistence.BatchTimeout),
		processor.WithExportTimeout(e.config.Persistence.ExportTimeout),
		processor.WithMaxExportBatchSize(e.config.Persistence.MaxExportBatchSize),
	)

	return nil
}

func (e *Coordinator) Stop(ctx context.Context) error {
	if e.nodeRecordProc != nil {
		if err := e.nodeRecordProc.Shutdown(ctx); err != nil {
			e.log.WithError(err).Error("failed to shutdown node record processor")
		}
	}

	if e.persistence != nil {
		if err := e.persistence.Stop(ctx); err != nil {
			return err
		}
	}

	e.log.Info("module stopped")

	return nil
}

func (e *Coordinator) CreateNodeRecords(ctx context.Context, req *xatu.CreateNodeRecordsRequest) (*xatu.CreateNodeRecordsResponse, error) {
	for _, record := range req.NodeRecords {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		e.metrics.AddNodeRecordReceived(1, "unknown")

		pRecord, err := node.Parse(record)
		if err != nil {
			return nil, err
		}

		e.nodeRecordProc.Write(pRecord)
	}

	return &xatu.CreateNodeRecordsResponse{}, nil
}
