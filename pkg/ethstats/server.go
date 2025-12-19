package ethstats

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/auth"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Server handles WebSocket connections from ethstats clients.
type Server struct {
	config        *Config
	log           logrus.FieldLogger
	authorization *auth.Authorization
	handler       *Handler
	metrics       *Metrics

	upgrader websocket.Upgrader

	nodes   map[string]*Node
	nodesMu sync.RWMutex

	httpServer *http.Server
}

// NewServer creates a new ethstats WebSocket server.
func NewServer(
	config *Config,
	log logrus.FieldLogger,
	authorization *auth.Authorization,
	handler *Handler,
	metrics *Metrics,
) *Server {
	return &Server{
		config:        config,
		log:           log.WithField("component", "server"),
		authorization: authorization,
		handler:       handler,
		metrics:       metrics,
		nodes:         make(map[string]*Node, 100),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.WebSocket.ReadBufferSize,
			WriteBufferSize: config.WebSocket.WriteBufferSize,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// Start starts the WebSocket server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", s.handleWebSocket)
	mux.HandleFunc("/", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:              s.config.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.log.WithField("addr", s.config.Addr).Info("Starting ethstats WebSocket server")

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.WithError(err).Error("WebSocket server error")
		}
	}()

	return nil
}

// Stop stops the WebSocket server.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("Stopping ethstats WebSocket server")

	// Close all node connections
	s.nodesMu.Lock()

	for _, node := range s.nodes {
		if err := node.Close(); err != nil {
			s.log.WithError(err).WithField("node_id", node.ID()).Warn("Error closing node connection")
		}
	}

	s.nodes = make(map[string]*Node)
	s.nodesMu.Unlock()

	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	return nil
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.nodesMu.RLock()
	nodeCount := len(s.nodes)
	s.nodesMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"nodes":  nodeCount,
	}); err != nil {
		s.log.WithError(err).Error("Error encoding health response")
	}
}

// handleWebSocket handles WebSocket upgrade and connection.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.WithError(err).Error("WebSocket upgrade failed")
		s.metrics.IncConnectionsTotal("upgrade_failed")

		return
	}

	// Extract client IP
	clientIP := s.extractClientIP(r)

	s.log.WithField("client_ip", clientIP).Debug("New WebSocket connection")

	// Handle the connection with a background context
	// We don't use r.Context() because it gets cancelled after the HTTP handler returns
	go s.handleConnection(context.Background(), conn, clientIP)
}

// extractClientIP extracts the client IP from the request.
func (s *Server) extractClientIP(r *http.Request) net.IP {
	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if ip := net.ParseIP(realIP); ip != nil {
			return ip
		}
	}

	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := net.ParseIP(strings.TrimSpace(parts[0])); ip != nil {
				return ip
			}
		}
	}

	// Fall back to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}

	return net.ParseIP(host)
}

// handleConnection handles a WebSocket connection lifecycle.
func (s *Server) handleConnection(ctx context.Context, conn *websocket.Conn, clientIP net.IP) {
	defer func() {
		if err := conn.Close(); err != nil {
			s.log.WithError(err).Debug("Error closing WebSocket connection")
		}
	}()

	conn.SetReadLimit(s.config.WebSocket.ReadLimit)

	// Wait for hello message
	node, err := s.handleHello(ctx, conn, clientIP)
	if err != nil {
		s.log.WithError(err).Warn("Hello handshake failed")
		s.metrics.IncConnectionsTotal("auth_failed")

		return
	}

	s.metrics.IncConnectionsTotal("success")
	s.metrics.IncConnectionsActive()

	defer func() {
		s.removeNode(node.ID())
		s.metrics.DecConnectionsActive()
	}()

	// Send ready message
	if err := s.sendReady(node); err != nil {
		s.log.WithError(err).Error("Failed to send ready message")

		return
	}

	// Start ping goroutine
	go s.pingLoop(ctx, node)

	// Read messages
	s.readLoop(ctx, node)
}

// handleHello handles the hello handshake.
func (s *Server) handleHello(ctx context.Context, conn *websocket.Conn, clientIP net.IP) (*Node, error) {
	// Set read deadline for hello
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read hello message: %w", err)
	}

	// Reset read deadline
	err = conn.SetReadDeadline(time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to reset read deadline: %w", err)
	}

	var msg Message

	err = json.Unmarshal(msgBytes, &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	if len(msg.Emit) < 2 {
		return nil, fmt.Errorf("invalid message format")
	}

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	if cmd != CommandHello {
		return nil, fmt.Errorf("expected hello command, got: %s", cmd)
	}

	var authMsg AuthMsg

	err = json.Unmarshal(msg.Emit[1], &authMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse auth message: %w", err)
	}

	if authMsg.ID == "" {
		return nil, fmt.Errorf("node ID is required")
	}

	// Decode credentials from secret
	// Format: nodename:base64(username:password)@server:port
	// authMsg.ID = nodename, authMsg.Secret = base64(username:password)
	decodedSecret, err := base64.StdEncoding.DecodeString(authMsg.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	credentials := strings.SplitN(string(decodedSecret), ":", 2)
	if len(credentials) != 2 {
		return nil, fmt.Errorf("invalid credential format, expected username:password")
	}

	username, password := credentials[0], credentials[1]

	// Authenticate using the decoded credentials
	var user *auth.User

	var group *auth.Group

	if s.authorization != nil {
		authorized, err := s.authorization.IsAuthorizedBasic(username, password)
		if err != nil {
			return nil, fmt.Errorf("authorization error: %w", err)
		}

		if !authorized {
			return nil, fmt.Errorf("authentication failed for user %s (node %s)", username, authMsg.ID)
		}

		user, group, err = s.authorization.GetUserAndGroup(username)
		if err != nil {
			return nil, fmt.Errorf("failed to get user and group: %w", err)
		}
	}

	// Create node
	node := NewNode(authMsg.ID, &authMsg.Info, conn, clientIP, s.config.WebSocket.WriteWait, s.log)
	node.SetAuth(user, group)

	// Register node (close existing connection if exists)
	s.registerNode(node)

	// Handle hello event
	if err := s.handler.HandleHello(ctx, node, &authMsg); err != nil {
		s.log.WithError(err).Error("Failed to handle hello event")
	}

	s.log.WithFields(logrus.Fields{
		"node_id":   authMsg.ID,
		"node_name": authMsg.Info.Name,
		"client":    authMsg.Info.Client,
		"network":   authMsg.Info.Network,
		"client_ip": clientIP,
	}).Info("Node connected")

	return node, nil
}

// sendReady sends the ready message to the client.
func (s *Server) sendReady(node *Node) error {
	msg := Message{
		Emit: []json.RawMessage{
			json.RawMessage(`"ready"`),
		},
	}

	if err := node.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to write ready message: %w", err)
	}

	return nil
}

// registerNode registers a node, closing any existing connection with the same ID.
func (s *Server) registerNode(node *Node) {
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	// Close existing connection if exists
	if existing, ok := s.nodes[node.ID()]; ok {
		s.log.WithField("node_id", node.ID()).Debug("Closing existing connection")

		if err := existing.Close(); err != nil {
			s.log.WithError(err).Debug("Error closing existing connection")
		}
	}

	s.nodes[node.ID()] = node
}

// removeNode removes a node from the registry.
func (s *Server) removeNode(nodeID string) {
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	delete(s.nodes, nodeID)

	s.log.WithField("node_id", nodeID).Info("Node disconnected")
}

// pingLoop sends periodic pings to keep the connection alive.
func (s *Server) pingLoop(ctx context.Context, node *Node) {
	ticker := time.NewTicker(s.config.WebSocket.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			// Primus ping is sent as a raw JSON string, not wrapped in emit
			pingMsg := fmt.Sprintf("%s%d", PrimusPingPrefix, t.UnixMilli())

			if err := node.WriteJSON(pingMsg); err != nil {
				s.log.WithError(err).Debug("Failed to send ping")

				return
			}
		}
	}
}

// readLoop reads messages from the WebSocket connection.
func (s *Server) readLoop(ctx context.Context, node *Node) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msgBytes, err := node.Conn().ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.log.WithError(err).Debug("WebSocket read error")
				}

				return
			}

			s.handleMessage(ctx, node, msgBytes)
		}
	}
}

// handleMessage handles an incoming message from a node.
func (s *Server) handleMessage(ctx context.Context, node *Node, msgBytes []byte) {
	start := time.Now()

	// First, check if this is a raw primus pong string (not wrapped in emit)
	var rawString string
	if err := json.Unmarshal(msgBytes, &rawString); err == nil {
		if strings.HasPrefix(rawString, PrimusPongPrefix) {
			// Primus pong received, connection is alive
			return
		}
	}

	var msg Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		s.log.WithError(err).Debug("Failed to parse message")

		return
	}

	if len(msg.Emit) < 1 {
		s.log.Debug("Invalid message format: empty emit array")

		return
	}

	var cmd string
	if err := json.Unmarshal(msg.Emit[0], &cmd); err != nil {
		s.log.WithError(err).Debug("Failed to parse command")

		return
	}

	// Handle primus pong (in case it's wrapped in emit for some clients)
	if strings.HasPrefix(cmd, PrimusPongPrefix) {
		return
	}

	s.metrics.IncEventsReceived(cmd)

	// Get payload if exists
	var payload json.RawMessage
	if len(msg.Emit) > 1 {
		payload = msg.Emit[1]
	}

	// Dispatch to handler
	if err := s.handler.Handle(ctx, node, cmd, payload); err != nil {
		s.log.WithError(err).WithField("command", cmd).Debug("Failed to handle message")
		s.metrics.IncEventErrors(cmd, "handler_error")
	}

	s.metrics.ObserveMessageLatency(cmd, time.Since(start).Seconds())
}

// Nodes returns a copy of the current nodes map.
func (s *Server) Nodes() map[string]*Node {
	s.nodesMu.RLock()
	defer s.nodesMu.RUnlock()

	nodes := make(map[string]*Node, len(s.nodes))
	for k, v := range s.nodes {
		nodes[k] = v
	}

	return nodes
}
