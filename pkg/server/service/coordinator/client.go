package coordinator

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/persistence/cannon"
	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	n "github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ServiceType = "coordinator"
)

type Client struct {
	xatu.UnimplementedCoordinatorServer

	log           logrus.FieldLogger
	config        *Config
	persistence   *persistence.Client
	geoipProvider geoip.Provider

	metrics *Metrics

	nodeRecord *n.Record
}

func NewClient(ctx context.Context, log logrus.FieldLogger, conf *Config, p *persistence.Client, geoipProvider geoip.Provider) (*Client, error) {
	if p == nil {
		return nil, fmt.Errorf("%s: persistence is required", ServiceType)
	}

	logger := log.WithField("server/module", ServiceType)

	nodeRecord, err := n.NewRecord(ctx, log, &conf.NodeRecord, p)
	if err != nil {
		return nil, err
	}

	e := &Client{
		log:           logger,
		config:        conf,
		persistence:   p,
		geoipProvider: geoipProvider,
		nodeRecord:    nodeRecord,
		metrics:       NewMetrics("xatu_server_coordinator"),
	}

	return e, nil
}

func (c *Client) Start(ctx context.Context, grpcServer *grpc.Server) error {
	c.log.Info("Starting module")

	xatu.RegisterCoordinatorServer(grpcServer, c)

	if err := c.nodeRecord.Start(ctx); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.nodeRecord != nil {
		if err := c.nodeRecord.Stop(ctx); err != nil {
			c.log.WithError(err).Error("failed to shutdown node record processor")
		}
	}

	c.log.Info("Module stopped")

	return nil
}

func (c *Client) CreateNodeRecords(ctx context.Context, req *xatu.CreateNodeRecordsRequest) (*xatu.CreateNodeRecordsResponse, error) {
	for _, record := range req.NodeRecords {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		c.metrics.AddNodeRecordReceived(1, "unknown")

		pRecord, err := node.Parse(record)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		if c.geoipProvider != nil {
			ipAddress := pRecord.IP4
			if ipAddress == nil {
				ipAddress = pRecord.IP6
			}

			if ipAddress != nil {
				ip := net.ParseIP(*ipAddress)
				if ip != nil {
					geoipLookupResult, err := c.geoipProvider.LookupIP(ctx, ip)
					if err != nil {
						c.log.WithField("ip", *ipAddress).WithError(err).Warn("failed to lookup geoip data")
					}

					if geoipLookupResult != nil && !(geoipLookupResult.Longitude == 0 && geoipLookupResult.Latitude == 0) {
						pRecord.GeoCity = &geoipLookupResult.CityName
						pRecord.GeoCountry = &geoipLookupResult.CountryName
						pRecord.GeoCountryCode = &geoipLookupResult.CountryCode
						pRecord.GeoContinentCode = &geoipLookupResult.ContinentCode
						pRecord.GeoLongitude = &geoipLookupResult.Longitude
						pRecord.GeoLatitude = &geoipLookupResult.Latitude
						pRecord.GeoAutonomousSystemNumber = &geoipLookupResult.AutonomousSystemNumber
						pRecord.GeoAutonomousSystemOrganization = &geoipLookupResult.AutonomousSystemOrganization
					}
				}
			}
		}

		c.nodeRecord.Write(ctx, pRecord)
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

	nodeRecords, err := c.persistence.CheckoutStalledExecutionNodeRecords(ctx, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Errorf(codes.InvalidArgument, "status is required")
	}

	st := node.Execution{
		Enr:             req.Status.NodeRecord,
		Name:            req.Status.Name,
		ProtocolVersion: fmt.Sprintf("%v", req.Status.ProtocolVersion),
		NetworkID:       fmt.Sprintf("%v", req.Status.NetworkId),
		TotalDifficulty: req.Status.TotalDifficulty,
		Head:            req.Status.Head,
		Genesis:         req.Status.Genesis,
	}

	if req.Status.ForkId != nil {
		st.ForkIDHash = req.Status.ForkId.Hash
		st.ForkIDNext = fmt.Sprintf("%v", req.Status.ForkId.Next)
	}

	if req.Status.Capabilities != nil {
		capabilitiesStr := []string{}
		for _, cap := range req.Status.Capabilities {
			capabilitiesStr = append(capabilitiesStr, cap.GetName()+"/"+fmt.Sprint(cap.GetVersion()))
		}

		st.Capabilities = strings.Join(capabilitiesStr, ",")
	}

	result := "error"

	defer func() {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		c.metrics.AddExecutionNodeRecordStatusReceived(1, "unknown", result, st.NetworkID, fmt.Sprintf("0x%x", st.ForkIDHash))
	}()

	err := c.persistence.InsertNodeRecordExecution(ctx, &st)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result = "success"

	nodeRecord := &node.Record{
		Enr:                     req.Status.NodeRecord,
		LastConnectTime:         sql.NullTime{Time: time.Now(), Valid: true},
		ConsecutiveDialAttempts: 0,
	}

	err = c.persistence.UpdateNodeRecord(ctx, nodeRecord)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &xatu.CreateExecutionNodeRecordStatusResponse{}, nil
}

func (c *Client) CoordinateExecutionNodeRecords(ctx context.Context, req *xatu.CoordinateExecutionNodeRecordsRequest) (*xatu.CoordinateExecutionNodeRecordsResponse, error) {
	targetedNodes := []string{}
	ignoredNodeRecords := []string{}
	activities := []*node.Activity{}

	if req.ClientId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client id is required")
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
			return nil, status.Error(codes.Internal, err.Error())
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
			return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(records) == 0 {
		return nil, status.Errorf(codes.NotFound, "no records found")
	}

	//nolint:gosec // not a security issue
	randomRecord := records[rand.Intn(len(records))]

	return &xatu.GetDiscoveryNodeRecordResponse{
		NodeRecord: randomRecord.Enr,
	}, nil
}

func (c *Client) GetCannonLocation(ctx context.Context, req *xatu.GetCannonLocationRequest) (*xatu.GetCannonLocationResponse, error) {
	location, err := c.persistence.GetCannonLocationByNetworkIDAndType(ctx, req.NetworkId, req.Type.Enum().String())
	if err != nil && err != persistence.ErrCannonLocationNotFound {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rsp := &xatu.GetCannonLocationResponse{}

	if location == nil {
		return rsp, nil
	}

	protoLoc, err := location.Unmarshal()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &xatu.GetCannonLocationResponse{
		Location: protoLoc,
	}, nil
}

func (c *Client) UpsertCannonLocation(ctx context.Context, req *xatu.UpsertCannonLocationRequest) (*xatu.UpsertCannonLocationResponse, error) {
	newLocation := &cannon.Location{}

	err := newLocation.Marshal(req.Location)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = c.persistence.UpsertCannonLocation(ctx, newLocation)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &xatu.UpsertCannonLocationResponse{}, nil
}
