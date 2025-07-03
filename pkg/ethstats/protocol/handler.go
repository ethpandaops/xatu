package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type Handler struct {
	log     logrus.FieldLogger
	metrics MetricsInterface
	auth    AuthInterface
	manager ConnectionManagerInterface
}

func NewHandler(log logrus.FieldLogger, metrics MetricsInterface, auth AuthInterface, manager ConnectionManagerInterface) *Handler {
	return &Handler{
		log:     log,
		metrics: metrics,
		auth:    auth,
		manager: manager,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, client ClientInterface, data []byte) error {
	// Parse base message
	msg, err := ParseMessage(data)
	if err != nil {
		h.log.WithError(err).Debug("Failed to parse message")
		h.metrics.IncProtocolError("parse_error")

		return fmt.Errorf("failed to parse message: %w", err)
	}

	if len(msg.Emit) == 0 {
		h.metrics.IncProtocolError("empty_emit")

		return fmt.Errorf("empty emit array")
	}

	// Get message type
	msgType, ok := msg.Emit[0].(string)
	if !ok {
		h.metrics.IncProtocolError("invalid_message_type")

		return fmt.Errorf("invalid message type")
	}

	// Log message
	h.log.WithFields(logrus.Fields{
		"type":      msgType,
		"client_id": client.ID(),
	}).Debug("Received message")

	// Handle message based on type
	switch msgType {
	case "hello":
		return h.handleHello(ctx, client, msg.Emit)
	case "block":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		return h.handleBlock(ctx, client, msg.Emit)
	case "stats":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		return h.handleStats(ctx, client, msg.Emit)
	case "pending":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		return h.handlePending(ctx, client, msg.Emit)
	case "node-ping":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		return h.handleNodePing(ctx, client, msg.Emit)
	case "latency":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		return h.handleLatency(ctx, client, msg.Emit)
	case "history":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		return h.handleHistory(ctx, client, msg.Emit)
	case "primus::pong::":
		if !client.IsAuthenticated() {
			return fmt.Errorf("client not authenticated")
		}

		if len(msg.Emit) > 1 {
			if timestamp, ok := msg.Emit[1].(string); ok {
				return h.handlePrimusPong(ctx, client, timestamp)
			}
		}

		return nil
	default:
		h.log.WithField("type", msgType).Warn("Unknown message type")
		h.metrics.IncProtocolError("unknown_message_type")

		return nil
	}
}

func (h *Handler) handleHello(ctx context.Context, client ClientInterface, emit []interface{}) error {
	hello, err := ParseHelloMessage(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_hello")

		return fmt.Errorf("failed to parse hello message: %w", err)
	}

	// Authorize client
	username, group, err := h.auth.AuthorizeSecret(hello.Secret)
	if err != nil {
		h.metrics.IncAuthentication("failed", "")
		h.log.WithError(err).WithField("node", hello.ID).Warn("Authentication failed")

		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set client as authenticated
	client.SetAuthenticated(hello.ID, username, group, &hello.Info)
	h.metrics.IncAuthentication("success", group)
	h.metrics.IncConnectedClients(hello.Info.Client, hello.Info.Net)

	h.log.WithFields(logrus.Fields{
		"id":       hello.ID,
		"username": username,
		"group":    group,
		"client":   hello.Info.Client,
		"network":  hello.Info.Net,
	}).Info("Client authenticated")

	// Send ready message
	readyMsg := FormatReadyMessage()
	if err := client.SendMessage(readyMsg); err != nil {
		return fmt.Errorf("failed to send ready message: %w", err)
	}

	h.metrics.IncMessagesSent("ready")

	return nil
}

func (h *Handler) handleBlock(ctx context.Context, client ClientInterface, emit []interface{}) error {
	report, err := ParseBlockReport(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_block")

		return fmt.Errorf("failed to parse block report: %w", err)
	}

	h.metrics.IncMessagesReceived("block", client.ID())

	h.log.WithFields(logrus.Fields{
		"client_id":   client.ID(),
		"block_num":   report.Block.Number.Value(),
		"block_hash":  report.Block.Hash,
		"parent_hash": report.Block.ParentHash,
		"miner":       report.Block.Miner,
		"gas_used":    report.Block.GasUsed,
		"gas_limit":   report.Block.GasLimit,
		"tx_count":    len(report.Block.Transactions),
	}).Debug("Received block report")

	return nil
}

func (h *Handler) handleStats(ctx context.Context, client ClientInterface, emit []interface{}) error {
	report, err := ParseStatsReport(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_stats")

		return fmt.Errorf("failed to parse stats report: %w", err)
	}

	h.metrics.IncMessagesReceived("stats", client.ID())

	h.log.WithFields(logrus.Fields{
		"client_id": client.ID(),
		"active":    report.Stats.Active,
		"syncing":   report.Stats.Syncing,
		"mining":    report.Stats.Mining,
		"hashrate":  report.Stats.Hashrate,
		"peers":     report.Stats.Peers,
		"gas_price": report.Stats.GasPrice,
		"uptime":    report.Stats.Uptime,
	}).Debug("Received stats report")

	return nil
}

func (h *Handler) handlePending(ctx context.Context, client ClientInterface, emit []interface{}) error {
	report, err := ParsePendingReport(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_pending")

		return fmt.Errorf("failed to parse pending report: %w", err)
	}

	h.metrics.IncMessagesReceived("pending", client.ID())

	h.log.WithFields(logrus.Fields{
		"client_id": client.ID(),
		"pending":   report.Stats.Pending,
	}).Debug("Received pending report")

	return nil
}

func (h *Handler) handleNodePing(ctx context.Context, client ClientInterface, emit []interface{}) error {
	ping, err := ParseNodePing(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_node_ping")

		return fmt.Errorf("failed to parse node ping: %w", err)
	}

	h.metrics.IncMessagesReceived("node-ping", client.ID())

	// Send pong response
	serverTime := fmt.Sprintf("%d", time.Now().UnixMilli())
	pongMsg := FormatNodePong(ping, serverTime)

	if err := client.SendMessage(pongMsg); err != nil {
		return fmt.Errorf("failed to send node pong: %w", err)
	}

	h.metrics.IncMessagesSent("node-pong")

	return nil
}

func (h *Handler) handleLatency(ctx context.Context, client ClientInterface, emit []interface{}) error {
	report, err := ParseLatencyReport(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_latency")

		return fmt.Errorf("failed to parse latency report: %w", err)
	}

	h.metrics.IncMessagesReceived("latency", client.ID())

	h.log.WithFields(logrus.Fields{
		"client_id": client.ID(),
		"latency":   report.Latency,
	}).Debug("Received latency report")

	return nil
}

func (h *Handler) handleHistory(ctx context.Context, client ClientInterface, emit []interface{}) error {
	report, err := ParseHistoryReport(emit)
	if err != nil {
		h.metrics.IncProtocolError("invalid_history")

		return fmt.Errorf("failed to parse history report: %w", err)
	}

	h.metrics.IncMessagesReceived("history", client.ID())

	h.log.WithFields(logrus.Fields{
		"client_id":   client.ID(),
		"block_count": len(report.History),
	}).Debug("Received history report")

	return nil
}

func (h *Handler) handlePrimusPong(ctx context.Context, client ClientInterface, timestamp string) error {
	h.metrics.IncMessagesReceived("primus::pong::", client.ID())

	h.log.WithFields(logrus.Fields{
		"client_id": client.ID(),
		"timestamp": timestamp,
	}).Debug("Received primus pong")

	return nil
}
