package coordinator

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	perrors "github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/persistence/cannon"
	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	n "github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
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

	healthServer *health.Server
}

func NewClient(ctx context.Context, log logrus.FieldLogger, conf *Config, p *persistence.Client, geoipProvider geoip.Provider, healthServer *health.Server) (*Client, error) {
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
		healthServer:  healthServer,
	}

	return e, nil
}

// Name returns the name of this service
func (c *Client) Name() string {
	return "coordinator"
}

func (c *Client) Start(ctx context.Context, grpcServer *grpc.Server) error {
	c.log.Info("Starting module")

	xatu.RegisterCoordinatorServer(grpcServer, c)

	if err := c.nodeRecord.Start(ctx); err != nil {
		return status.Error(codes.Internal, perrors.Wrap(err, "failed to start node record processor").Error())
	}

	c.healthServer.SetServingStatus(c.Name(), grpc_health_v1.HealthCheckResponse_SERVING)

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.nodeRecord != nil {
		if err := c.nodeRecord.Stop(ctx); err != nil {
			c.log.WithError(err).Error("failed to shutdown node record processor")
		}
	}

	c.log.Info("Module stopped")

	c.healthServer.SetServingStatus(c.Name(), grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	return nil
}

func (c *Client) CreateNodeRecords(ctx context.Context, req *xatu.CreateNodeRecordsRequest) (*xatu.CreateNodeRecordsResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	for _, record := range req.NodeRecords {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		c.metrics.AddNodeRecordReceived(1, "unknown")

		pRecord, err := node.Parse(record)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, perrors.Wrap(err, "failed to parse node record").Error())
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
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	pageSize := int(req.PageSize)
	if pageSize == 0 {
		pageSize = 100
	}

	if pageSize > 1000 {
		pageSize = 1000
	}

	nodeRecords, err := c.persistence.CheckoutStalledExecutionNodeRecords(ctx, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get stalled execution node records from db").Error())
	}

	response := &xatu.ListStalledExecutionNodeRecordsResponse{
		NodeRecords: []string{},
	}

	for _, record := range nodeRecords {
		response.NodeRecords = append(response.NodeRecords, record.Enr)
	}

	return response, nil
}

func (c *Client) ListStalledConsensusNodeRecords(ctx context.Context, req *xatu.ListStalledConsensusNodeRecordsRequest) (*xatu.ListStalledConsensusNodeRecordsResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	pageSize := int(req.PageSize)
	if pageSize == 0 {
		pageSize = 100
	}

	if pageSize > 1000 {
		pageSize = 1000
	}

	nodeRecords, err := c.persistence.CheckoutStalledConsensusNodeRecords(ctx, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get stalled consensus node records from db").Error())
	}

	response := &xatu.ListStalledConsensusNodeRecordsResponse{
		NodeRecords: []string{},
	}

	for _, record := range nodeRecords {
		response.NodeRecords = append(response.NodeRecords, record.Enr)
	}

	return response, nil
}

func (c *Client) CreateExecutionNodeRecordStatus(ctx context.Context, req *xatu.CreateExecutionNodeRecordStatusRequest) (*xatu.CreateExecutionNodeRecordStatusResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

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
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to insert node record execution").Error())
	}

	result = "success"

	nodeRecord := &node.Record{
		Enr:                     req.Status.NodeRecord,
		LastConnectTime:         sql.NullTime{Time: time.Now(), Valid: true},
		ConsecutiveDialAttempts: 0,
	}

	err = c.persistence.UpdateNodeRecord(ctx, nodeRecord)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to update node record").Error())
	}

	return &xatu.CreateExecutionNodeRecordStatusResponse{}, nil
}

func (c *Client) CreateConsensusNodeRecordStatus(ctx context.Context, req *xatu.CreateConsensusNodeRecordStatusRequest) (*xatu.CreateConsensusNodeRecordStatusResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	if req.Status == nil {
		return nil, status.Errorf(codes.InvalidArgument, "status is required")
	}

	st := node.Consensus{
		Enr:            req.Status.NodeRecord,
		NodeID:         req.Status.NodeId,
		PeerID:         req.Status.PeerId,
		CreateTime:     time.Now(),
		Name:           req.Status.Name,
		ForkDigest:     req.Status.ForkDigest,
		NextForkDigest: req.Status.NextForkDigest,
		FinalizedRoot:  req.Status.FinalizedRoot,
		FinalizedEpoch: req.Status.FinalizedEpoch,
		HeadRoot:       req.Status.HeadRoot,
		HeadSlot:       req.Status.HeadSlot,
		CGC:            req.Status.Cgc,
		NetworkID:      fmt.Sprintf("%v", req.Status.NetworkId),
	}

	result := "error"

	defer func() {
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		c.metrics.AddConsensusNodeRecordStatusReceived(1, "unknown", result, st.NetworkID, fmt.Sprintf("0x%x", st.ForkDigest))
	}()

	err := c.persistence.InsertNodeRecordConsensus(ctx, &st)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to insert node record consensus").Error())
	}

	result = "success"

	nodeRecord := &node.Record{
		Enr:                     req.Status.NodeRecord,
		LastConnectTime:         sql.NullTime{Time: time.Now(), Valid: true},
		ConsecutiveDialAttempts: 0,
		CGC:                     &req.Status.Cgc,
	}

	err = c.persistence.UpdateNodeRecord(ctx, nodeRecord)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to update node record").Error())
	}

	return &xatu.CreateConsensusNodeRecordStatusResponse{}, nil
}

func (c *Client) CreateConsensusNodeRecordStatuses(
	ctx context.Context,
	req *xatu.CreateConsensusNodeRecordStatusesRequest,
) (*xatu.CreateConsensusNodeRecordStatusesResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	// Prepare consensus records and node records for bulk operations.
	var (
		consensusRecords = make([]*node.Consensus, 0, len(req.Statuses))
		nodeRecords      = make([]*node.Record, 0, len(req.Statuses))
		now              = time.Now()
	)

	for _, st := range req.Statuses {
		if st == nil {
			continue
		}

		consensusRecords = append(consensusRecords, &node.Consensus{
			Enr:            st.NodeRecord,
			NodeID:         st.NodeId,
			PeerID:         st.PeerId,
			CreateTime:     now,
			Name:           st.Name,
			ForkDigest:     st.ForkDigest,
			NextForkDigest: st.NextForkDigest,
			FinalizedRoot:  st.FinalizedRoot,
			FinalizedEpoch: st.FinalizedEpoch,
			HeadRoot:       st.HeadRoot,
			HeadSlot:       st.HeadSlot,
			CGC:            st.Cgc,
			NetworkID:      fmt.Sprintf("%v", st.NetworkId),
		})

		nodeRecords = append(nodeRecords, &node.Record{
			Enr:                     st.NodeRecord,
			LastConnectTime:         sql.NullTime{Time: now, Valid: true},
			ConsecutiveDialAttempts: 0,
			CGC:                     &st.Cgc,
		})
	}

	// Bulk insert consensus records.
	if len(consensusRecords) > 0 {
		if err := c.persistence.BulkInsertNodeRecordConsensus(ctx, consensusRecords); err != nil {
			return nil, status.Error(
				codes.Internal,
				perrors.Wrap(err, "failed to bulk insert node record consensus").Error(),
			)
		}

		// Update metrics for all successfully inserted records.
		for _, record := range consensusRecords {
			c.metrics.AddConsensusNodeRecordStatusReceived(1, "unknown", "success", record.NetworkID, fmt.Sprintf("0x%x", record.ForkDigest))
		}

		// Update node records individually.
		for _, nodeRecord := range nodeRecords {
			if err := c.persistence.UpdateNodeRecord(ctx, nodeRecord); err != nil {
				return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to update node record").Error())
			}
		}
	}

	return &xatu.CreateConsensusNodeRecordStatusesResponse{}, nil
}

func (c *Client) CoordinateExecutionNodeRecords(ctx context.Context, req *xatu.CoordinateExecutionNodeRecordsRequest) (*xatu.CoordinateExecutionNodeRecordsResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

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
		newNodeRecords, err := c.persistence.ListAvailableExecutionNodeRecords(ctx, req.ClientId, ignoredNodeRecords, req.NetworkIds, req.ForkIdHashes, req.Capabilities, int(limit))
		if err != nil {
			return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get available execution node records from db").Error())
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
			return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to upsert node record activities to db").Error())
		}
	}

	return &xatu.CoordinateExecutionNodeRecordsResponse{
		NodeRecords: targetedNodes,
		RetryDelay:  5,
	}, nil
}

func (c *Client) CoordinateConsensusNodeRecords(ctx context.Context, req *xatu.CoordinateConsensusNodeRecordsRequest) (*xatu.CoordinateConsensusNodeRecordsResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

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

	//nolint:gosec // conversion is fine here.
	limit -= uint32(len(targetedNodes))

	if limit > 0 {
		// Note: fork_id_hashes field will not be used for consensus nodes
		newNodeRecords, err := c.persistence.ListAvailableConsensusNodeRecords(ctx, req.ClientId, ignoredNodeRecords, req.NetworkIds, nil, int(limit))
		if err != nil {
			return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get available consensus node records from db").Error())
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
			return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to upsert node record activities to db").Error())
		}
	}

	return &xatu.CoordinateConsensusNodeRecordsResponse{
		NodeRecords: targetedNodes,
		RetryDelay:  5,
	}, nil
}

func (c *Client) GetDiscoveryNodeRecord(ctx context.Context, req *xatu.GetDiscoveryNodeRecordRequest) (*xatu.GetDiscoveryNodeRecordResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	records, err := c.persistence.ListNodeRecordExecutions(ctx, req.NetworkIds, req.ForkIdHashes, 100)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get discovery node records from db").Error())
	}

	if len(records) == 0 {
		return nil, status.Errorf(codes.NotFound, "no records found")
	}

	randomIndex, err := c.secureRandomInt(len(records))
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to generate random index").Error())
	}

	randomRecord := records[randomIndex]

	return &xatu.GetDiscoveryNodeRecordResponse{
		NodeRecord: randomRecord.Enr,
	}, nil
}

func (c *Client) GetDiscoveryConsensusNodeRecord(ctx context.Context, req *xatu.GetDiscoveryConsensusNodeRecordRequest) (*xatu.GetDiscoveryConsensusNodeRecordResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	records, err := c.persistence.ListNodeRecordConsensus(ctx, req.NetworkIds, req.ForkDigests, 100)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to list node record consensus from db").Error())
	}

	if len(records) == 0 {
		return nil, status.Errorf(codes.NotFound, "no records found")
	}

	randomIndex, err := c.secureRandomInt(len(records))
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to generate random index").Error())
	}

	randomRecord := records[randomIndex]

	return &xatu.GetDiscoveryConsensusNodeRecordResponse{
		NodeRecord: randomRecord.Enr,
	}, nil
}

func (c *Client) GetDiscoveryExecutionNodeRecord(ctx context.Context, req *xatu.GetDiscoveryExecutionNodeRecordRequest) (*xatu.GetDiscoveryExecutionNodeRecordResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	records, err := c.persistence.ListNodeRecordExecutions(ctx, req.NetworkIds, req.ForkIdHashes, 100)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get discovery node records from db").Error())
	}

	if len(records) == 0 {
		return nil, status.Errorf(codes.NotFound, "no records found")
	}

	randomIndex, err := c.secureRandomInt(len(records))
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to generate random index").Error())
	}

	randomRecord := records[randomIndex]

	return &xatu.GetDiscoveryExecutionNodeRecordResponse{
		NodeRecord: randomRecord.Enr,
	}, nil
}

func (c *Client) GetCannonLocation(ctx context.Context, req *xatu.GetCannonLocationRequest) (*xatu.GetCannonLocationResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	location, err := c.persistence.GetCannonLocationByNetworkIDAndType(ctx, req.NetworkId, req.Type.Enum().String())
	if err != nil && err != persistence.ErrCannonLocationNotFound {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to get cannon location from db").Error())
	}

	rsp := &xatu.GetCannonLocationResponse{}

	if location == nil {
		return rsp, nil
	}

	protoLoc, err := location.Unmarshal()
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to unmarshal cannon location").Error())
	}

	return &xatu.GetCannonLocationResponse{
		Location: protoLoc,
	}, nil
}

func (c *Client) UpsertCannonLocation(ctx context.Context, req *xatu.UpsertCannonLocationRequest) (*xatu.UpsertCannonLocationResponse, error) {
	if c.config.Auth.Enabled != nil && *c.config.Auth.Enabled {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		if err := c.validateAuth(ctx, md); err != nil {
			return nil, err
		}
	}

	newLocation := &cannon.Location{}

	err := newLocation.Marshal(req.Location)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, perrors.Wrap(err, "failed to marshal cannon location").Error())
	}

	err = c.persistence.UpsertCannonLocation(ctx, newLocation)
	if err != nil {
		return nil, status.Error(codes.Internal, perrors.Wrap(err, "failed to upsert cannon location to db").Error())
	}

	return &xatu.UpsertCannonLocationResponse{}, nil
}

func (c *Client) secureRandomInt(input int) (int, error) {
	if input <= 0 {
		return 0, fmt.Errorf("invalid range for random int: %d", input)
	}

	bigIndex, err := crand.Int(crand.Reader, big.NewInt(int64(input)))
	if err != nil {
		return 0, perrors.Wrap(err, "failed to generate secure random number")
	}

	return int(bigIndex.Int64()), nil
}
