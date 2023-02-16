package coordinator

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	n "github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	ServiceType = "coordinator"
)

type Client struct {
	xatu.UnimplementedCoordinatorServer

	log         logrus.FieldLogger
	config      *Config
	persistence *persistence.Client

	metrics *Metrics

	nodeRecord *n.Record
}

func NewClient(ctx context.Context, log logrus.FieldLogger, conf *Config, p *persistence.Client) (*Client, error) {
	if p == nil {
		return nil, fmt.Errorf("%s: persistence is required", ServiceType)
	}

	logger := log.WithField("server/module", ServiceType)

	nodeRecord, err := n.NewRecord(ctx, log, &conf.NodeRecord, p)
	if err != nil {
		return nil, err
	}

	e := &Client{
		log:         logger,
		config:      conf,
		persistence: p,
		nodeRecord:  nodeRecord,
		metrics:     NewMetrics("xatu_coordinator"),
	}

	return e, nil
}

func (c *Client) Start(ctx context.Context, grpcServer *grpc.Server) error {
	c.log.Info("starting module")

	xatu.RegisterCoordinatorServer(grpcServer, c)

	if err := c.nodeRecord.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.nodeRecord != nil {
		if err := c.nodeRecord.Stop(ctx); err != nil {
			c.log.WithError(err).Error("failed to shutdown node record processor")
		}
	}

	c.log.Info("module stopped")

	return nil
}

func (c *Client) CreateNodeRecords(ctx context.Context, req *xatu.CreateNodeRecordsRequest) (*xatu.CreateNodeRecordsResponse, error) {
	for _, record := range req.NodeRecords {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		c.metrics.AddNodeRecordReceived(1, "unknown")

		pRecord, err := node.Parse(record)
		if err != nil {
			return nil, err
		}

		c.nodeRecord.Write(pRecord)
	}

	return &xatu.CreateNodeRecordsResponse{}, nil
}

func (c *Client) ListStalledExecutionNodeRecords(ctx context.Context, req *xatu.ListStalledExecutionNodeRecordsRequest) (*xatu.ListStalledExecutionNodeRecordsResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize == 0 {
		pageSize = 100
	}

	if pageSize > 1000 {
		pageSize = 1000
	}

	nodeRecords, err := c.persistence.CheckoutStalledExecutionNodeRecords(ctx, int(req.PageSize))
	if err != nil {
		return nil, err
	}

	response := &xatu.ListStalledExecutionNodeRecordsResponse{
		NodeRecords: []string{},
	}

	for _, record := range nodeRecords {
		response.NodeRecords = append(response.NodeRecords, record.Enr)
	}

	return response, nil
}

func (c *Client) CreateExecutionNodeRecordStatus(ctx context.Context, req *xatu.CreateExecutionNodeRecordStatusRequest) (*xatu.CreateExecutionNodeRecordStatusResponse, error) {
	if req.Status == nil {
		return nil, fmt.Errorf("status is required")
	}

	status := node.Execution{
		Enr:             req.Status.NodeRecord,
		Name:            req.Status.Name,
		ProtocolVersion: fmt.Sprintf("%v", req.Status.ProtocolVersion),
		NetworkID:       fmt.Sprintf("%v", req.Status.NetworkId),
		TotalDifficulty: req.Status.TotalDifficulty,
		Head:            req.Status.Head,
		Genesis:         req.Status.Genesis,
	}

	if req.Status.ForkId != nil {
		status.ForkIDHash = req.Status.ForkId.Hash
		status.ForkIDNext = fmt.Sprintf("%v", req.Status.ForkId.Next)
	}

	if req.Status.Capabilities != nil {
		capabilitiesStr := []string{}
		for _, cap := range req.Status.Capabilities {
			capabilitiesStr = append(capabilitiesStr, cap.GetName()+"/"+fmt.Sprint(cap.GetVersion()))
		}

		status.Capabilities = strings.Join(capabilitiesStr, ",")
	}

	result := "error"

	defer func() {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		c.metrics.AddExecutionNodeRecordStatusReceived(1, "unknown", result, status.NetworkID, fmt.Sprintf("0x%x", status.ForkIDHash))
	}()

	err := c.persistence.InsertNodeRecordExecution(ctx, &status)
	if err != nil {
		return nil, err
	}

	result = "success"

	nodeRecord := &node.Record{
		Enr:                     req.Status.NodeRecord,
		LastConnectTime:         sql.NullTime{Time: time.Now(), Valid: true},
		ConsecutiveDialAttempts: 0,
	}

	err = c.persistence.UpdateNodeRecord(ctx, nodeRecord)
	if err != nil {
		return nil, err
	}

	return &xatu.CreateExecutionNodeRecordStatusResponse{}, nil
}

func (c *Client) CoordinateExecutionNodeRecords(ctx context.Context, req *xatu.CoordinateExecutionNodeRecordsRequest) (*xatu.CoordinateExecutionNodeRecordsResponse, error) {
	targetedNodes := []string{}
	ignoredNodeRecords := []string{}
	activities := []*node.Activity{}

	if req.ClientId == "" {
		return nil, fmt.Errorf("client id is required")
	}

	for _, record := range req.NodeRecords {
		activity := &node.Activity{
			Enr:        record.NodeRecord,
			ClientID:   req.ClientId,
			UpdateTime: time.Now(),
			Connected:  record.Connected,
		}

		activities = append(activities, activity)

		// ignore nodes that have been connected to too many times
		if record.ConnectionAttempts < 100 {
			targetedNodes = append(targetedNodes, record.NodeRecord)
		}

		ignoredNodeRecords = append(ignoredNodeRecords, record.NodeRecord)
	}

	limit := req.Limit
	if limit == 0 {
		limit = 100
	}

	if limit > 1000 {
		limit = 1000
	}

	limit -= uint32(len(targetedNodes))

	if limit > 0 {
		newNodeRecords, err := c.persistence.ListAvailableExecutionNodeRecords(ctx, req.ClientId, ignoredNodeRecords, req.NetworkIds, req.ForkIdHashes, int(limit))
		if err != nil {
			return nil, err
		}

		for _, record := range newNodeRecords {
			activity := &node.Activity{
				Enr:        *record,
				ClientID:   req.ClientId,
				UpdateTime: time.Now(),
				Connected:  false,
			}

			activities = append(activities, activity)
			targetedNodes = append(targetedNodes, *record)
		}
	}

	if len(activities) != 0 {
		err := c.persistence.UpsertNodeRecordActivities(ctx, activities)
		if err != nil {
			return nil, err
		}
	}

	return &xatu.CoordinateExecutionNodeRecordsResponse{
		NodeRecords: targetedNodes,
		RetryDelay:  5,
	}, nil
}

func (c *Client) GetDiscoveryNodeRecord(ctx context.Context, req *xatu.GetDiscoveryNodeRecordRequest) (*xatu.GetDiscoveryNodeRecordResponse, error) {
	records, err := c.persistence.ListNodeRecordExecutions(ctx, req.NetworkIds, req.ForkIdHashes, 100)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no records found")
	}

	//nolint:gosec // not a security issue
	randomRecord := records[rand.Intn(len(records))]

	return &xatu.GetDiscoveryNodeRecordResponse{
		NodeRecord: randomRecord.Enr,
	}, nil
}
