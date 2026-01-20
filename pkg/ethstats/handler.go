package ethstats

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/geoip"
	"github.com/ethpandaops/xatu/pkg/geoip/lookup"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/ethstats"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Handler processes ethstats messages and converts them to decorated events.
type Handler struct {
	log            logrus.FieldLogger
	sinks          []output.Sink
	clientMeta     func(node *Node) *xatu.ClientMeta
	metrics        *Metrics
	clientNameSalt string
	geoipProvider  geoip.Provider
}

// NewHandler creates a new Handler instance.
func NewHandler(
	log logrus.FieldLogger,
	sinks []output.Sink,
	clientMeta func(node *Node) *xatu.ClientMeta,
	metrics *Metrics,
	clientNameSalt string,
	geoipProvider geoip.Provider,
) *Handler {
	return &Handler{
		log:            log.WithField("component", "handler"),
		sinks:          sinks,
		clientMeta:     clientMeta,
		metrics:        metrics,
		clientNameSalt: clientNameSalt,
		geoipProvider:  geoipProvider,
	}
}

// Handle dispatches a message to the appropriate handler.
func (h *Handler) Handle(ctx context.Context, node *Node, cmd string, payload json.RawMessage) error {
	switch cmd {
	case CommandBlock:
		return h.handleBlock(ctx, node, payload)
	case CommandStats:
		return h.handleStats(ctx, node, payload)
	case CommandPending:
		return h.handlePending(ctx, node, payload)
	case CommandLatency:
		return h.handleLatency(ctx, node, payload)
	case CommandBlockNewPayload:
		return h.handleNewPayload(ctx, node, payload)
	case CommandHistory:
		return h.handleHistory(ctx, node, payload)
	case CommandNodePing:
		return h.handleNodePing(ctx, node, payload)
	default:
		h.log.WithField("command", cmd).Debug("Unknown command")

		return nil
	}
}

// HandleHello handles the hello event.
func (h *Handler) HandleHello(ctx context.Context, node *Node, authMsg *AuthMsg) error {
	now := time.Now()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_HELLO,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsHello{
			EthstatsHello: &ethstats.Hello{
				NodeId: authMsg.ID,
				Info:   h.convertNodeInfo(&authMsg.Info),
			},
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handleBlock handles a block message.
func (h *Handler) handleBlock(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report BlockReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse block report: %w", err)
	}

	if report.Block == nil {
		return fmt.Errorf("block is nil")
	}

	node.SetLatestBlock(report.Block)

	now := time.Now()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_BLOCK,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsBlock{
			EthstatsBlock: h.convertBlock(report.Block),
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handleStats handles a stats message.
func (h *Handler) handleStats(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report StatsReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse stats report: %w", err)
	}

	if report.Stats == nil {
		return fmt.Errorf("stats is nil")
	}

	node.SetLatestStats(report.Stats)

	now := time.Now()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_NODE_STATS,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsNodeStats{
			EthstatsNodeStats: h.convertNodeStats(report.Stats),
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handlePending handles a pending message.
func (h *Handler) handlePending(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report PendingReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse pending report: %w", err)
	}

	if report.Stats == nil {
		return fmt.Errorf("pending stats is nil")
	}

	node.SetLatestPending(report.Stats)

	now := time.Now()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_PENDING,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsPending{
			EthstatsPending: &ethstats.PendingStats{
				Pending: wrapperspb.UInt32(uint32(report.Stats.Pending)), //nolint:gosec // pending count is bounded
			},
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handleLatency handles a latency message.
func (h *Handler) handleLatency(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report LatencyReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse latency report: %w", err)
	}

	node.SetLatency(report.Latency)

	now := time.Now()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_LATENCY,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsLatency{
			EthstatsLatency: &ethstats.Latency{
				LatencyMs: report.Latency,
			},
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handleNewPayload handles a block_new_payload message.
func (h *Handler) handleNewPayload(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report NewPayloadReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse new payload report: %w", err)
	}

	if report.Block == nil {
		return fmt.Errorf("new payload block is nil")
	}

	node.SetLatestNewPayload(report.Block)

	now := time.Now()

	blockNum, _ := report.Block.Number.Int64()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_NEW_PAYLOAD,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsNewPayload{
			EthstatsNewPayload: &ethstats.NewPayload{
				Number:           wrapperspb.UInt64(uint64(blockNum)), //nolint:gosec // block number is non-negative
				Hash:             report.Block.Hash,
				ProcessingTimeNs: wrapperspb.UInt64(report.Block.ProcessingTime),
			},
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handleHistory handles a history message.
func (h *Handler) handleHistory(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report HistoryReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse history report: %w", err)
	}

	now := time.Now()

	blocks := make([]*ethstats.Block, 0, len(report.History))
	for _, block := range report.History {
		blocks = append(blocks, h.convertBlock(block))
	}

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_ETHSTATS_HISTORY,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: h.clientMeta(node),
		},
		Data: &xatu.DecoratedEvent_EthstatsHistory{
			EthstatsHistory: &ethstats.History{
				Blocks: blocks,
			},
		},
	}

	return h.publishEvent(ctx, node, event)
}

// handleNodePing handles a node-ping message and sends a pong response.
func (h *Handler) handleNodePing(ctx context.Context, node *Node, payload json.RawMessage) error {
	var report PingReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return fmt.Errorf("failed to parse ping report: %w", err)
	}

	// Send pong response
	pong := map[string]any{
		"id":         report.ID,
		"clientTime": report.ClientTime,
		"serverTime": time.Now().UnixMilli(),
	}

	pongBytes, err := json.Marshal(pong)
	if err != nil {
		return fmt.Errorf("failed to marshal pong: %w", err)
	}

	msg := Message{
		Emit: []json.RawMessage{
			json.RawMessage(`"node-pong"`),
			pongBytes,
		},
	}

	if err := node.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send pong: %w", err)
	}

	return nil
}

// publishEvent publishes an event to all sinks after applying user/group filters.
func (h *Handler) publishEvent(ctx context.Context, node *Node, event *xatu.DecoratedEvent) error {
	events := []*xatu.DecoratedEvent{event}

	// Apply user filter if user exists
	if user := node.User(); user != nil {
		filtered, err := user.ApplyFilter(ctx, events)
		if err != nil {
			return fmt.Errorf("failed to apply user filter: %w", err)
		}

		events = filtered
	}

	// Apply group filter if group exists
	if group := node.Group(); group != nil {
		filtered, err := group.ApplyFilter(ctx, events)
		if err != nil {
			return fmt.Errorf("failed to apply group filter: %w", err)
		}

		events = filtered
	}

	// If all events were filtered out, return early
	if len(events) == 0 {
		h.metrics.IncEventsFiltered(event.GetEvent().GetName().String())

		return nil
	}

	// Populate ClientMeta.Client with IP, geo, and auth info
	h.enrichClientMeta(ctx, node, event)

	for _, e := range events {
		for _, sink := range h.sinks {
			if err := sink.HandleNewDecoratedEvent(ctx, e); err != nil {
				h.log.WithError(err).WithField("sink", sink.Name()).Debug("Failed to publish event")
				h.metrics.IncEventErrors(e.GetEvent().GetName().String(), "sink_error")

				continue
			}

			h.metrics.IncEventsExported(e.GetEvent().GetName().String(), sink.Name())
		}
	}

	return nil
}

// enrichClientMeta populates ClientMeta.Client with IP, geo, user, and group information.
func (h *Handler) enrichClientMeta(ctx context.Context, node *Node, event *xatu.DecoratedEvent) {
	if event.Meta == nil || event.Meta.Client == nil {
		return
	}

	clientInfo := &xatu.ClientMeta_Client{}

	// Set IP address
	if clientIP := node.ClientIP(); clientIP != nil {
		clientInfo.Ip = clientIP.String()

		// Perform GeoIP lookup if provider is available
		if h.geoipProvider != nil {
			// Determine precision from group
			precision := lookup.PrecisionFull
			if group := node.Group(); group != nil {
				precision = group.GetGeoPrecision()
			}

			geoipResult, err := h.geoipProvider.LookupIP(ctx, clientIP, precision)
			if err != nil {
				h.log.WithField("ip", clientIP.String()).WithError(err).Debug("failed to lookup geoip data")
			}

			if geoipResult != nil {
				clientInfo.Geo = &xatu.ClientMeta_Geo{
					City:                         geoipResult.CityName,
					Country:                      geoipResult.CountryName,
					CountryCode:                  geoipResult.CountryCode,
					ContinentCode:                geoipResult.ContinentCode,
					Latitude:                     geoipResult.Latitude,
					Longitude:                    geoipResult.Longitude,
					AutonomousSystemNumber:       geoipResult.AutonomousSystemNumber,
					AutonomousSystemOrganization: geoipResult.AutonomousSystemOrganization,
				}
			}
		}
	}

	// Set user, group, and name
	var username string

	if user := node.User(); user != nil {
		username = user.Username()
		clientInfo.User = username
	}

	if group := node.Group(); group != nil {
		clientInfo.Group = group.Name()
	}

	// Set the node ID as the client name (potentially obscured)
	clientName := node.ID()
	if group := node.Group(); group != nil && group.ShouldObscureClientName() && username != "" {
		clientName = group.ComputeClientName(username, h.clientNameSalt, clientName)
	}

	clientInfo.Name = clientName

	event.Meta.Client.Client = clientInfo
}

// convertNodeInfo converts protocol NodeInfo to proto NodeInfo.
func (h *Handler) convertNodeInfo(info *NodeInfo) *ethstats.NodeInfo {
	return &ethstats.NodeInfo{
		Name:             info.Name,
		Node:             info.Node,
		Port:             wrapperspb.UInt32(uint32(info.Port)), //nolint:gosec // port is bounded
		Network:          info.Network,
		Protocol:         info.Protocol,
		Api:              info.API,
		Os:               info.Os,
		OsVersion:        info.OsVer,
		Client:           info.Client,
		CanUpdateHistory: wrapperspb.Bool(info.History),
	}
}

// convertBlock converts protocol BlockStats to proto Block.
func (h *Handler) convertBlock(block *BlockStats) *ethstats.Block {
	blockNum, _ := strconv.ParseUint(block.Number.String(), 10, 64)
	timestamp, _ := strconv.ParseInt(block.Timestamp.String(), 10, 64)

	return &ethstats.Block{
		Number:           wrapperspb.UInt64(blockNum),
		Hash:             block.Hash,
		ParentHash:       block.ParentHash,
		Timestamp:        timestamppb.New(time.Unix(timestamp, 0)),
		Miner:            block.Miner,
		GasUsed:          wrapperspb.UInt64(block.GasUsed),
		GasLimit:         wrapperspb.UInt64(block.GasLimit),
		Difficulty:       block.Diff,
		TotalDifficulty:  block.TotalDiff,
		TransactionCount: wrapperspb.UInt32(uint32(len(block.Txs))), //nolint:gosec // length is bounded
		TransactionsRoot: block.TxHash,
		StateRoot:        block.Root,
		UncleCount:       wrapperspb.UInt32(uint32(len(block.Uncles))), //nolint:gosec // length is bounded
	}
}

// convertNodeStats converts protocol NodeStats to proto NodeStats.
func (h *Handler) convertNodeStats(stats *NodeStats) *ethstats.NodeStats {
	return &ethstats.NodeStats{
		Active:   wrapperspb.Bool(stats.Active),
		Syncing:  wrapperspb.Bool(stats.Syncing),
		Peers:    wrapperspb.UInt32(uint32(stats.Peers)),    //nolint:gosec // peer count is bounded
		GasPrice: wrapperspb.UInt64(uint64(stats.GasPrice)), //nolint:gosec // gas price is non-negative
		Uptime:   wrapperspb.UInt32(uint32(stats.Uptime)),   //nolint:gosec // uptime is non-negative
	}
}
