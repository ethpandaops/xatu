package ethstats

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/auth"
	"github.com/ethpandaops/xatu/pkg/ethstats/connection"
	"github.com/ethpandaops/xatu/pkg/ethstats/protocol"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config        *Config
	log           logrus.FieldLogger
	auth          *auth.Authorization
	manager       *connection.Manager
	handler       *protocol.Handler
	metrics       *Metrics
	upgrader      websocket.Upgrader
	httpServer    *http.Server
	metricsServer *http.Server
}

func NewServer(log logrus.FieldLogger, config *Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create metrics
	metrics := NewMetrics("xatu_ethstats")

	// Create authorization
	authz, err := auth.NewAuthorization(log.WithField("component", "auth"), config.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization: %w", err)
	}

	// Create rate limiter
	var rateLimiter *connection.RateLimiter
	if config.RateLimit.Enabled {
		rateLimiter = connection.NewRateLimiter(
			config.RateLimit.WindowDuration,
			config.RateLimit.ConnectionsPerIP,
			config.RateLimit.FailuresBeforeWarn,
			log.WithField("component", "ratelimiter"),
		)
	}

	// Create connection manager
	manager := connection.NewManager(rateLimiter, metrics, log.WithField("component", "manager"))

	// Create protocol handler
	handler := protocol.NewHandler(log.WithField("component", "handler"), metrics, authz, manager)

	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins for ethstats compatibility
			return true
		},
	}

	server := &Server{
		config:   config,
		log:      log,
		auth:     authz,
		manager:  manager,
		handler:  handler,
		metrics:  metrics,
		upgrader: upgrader,
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleWebSocket)
	server.httpServer = &http.Server{
		Addr:              config.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Create metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	server.metricsServer = &http.Server{
		Addr:              config.MetricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return server, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.log.WithField("addr", s.config.Addr).Info("Starting ethstats server")

	// Start authorization
	if err := s.auth.Start(ctx); err != nil {
		return fmt.Errorf("failed to start authorization: %w", err)
	}

	// Start metrics server
	go func() {
		s.log.WithField("addr", s.config.MetricsAddr).Info("Starting metrics server")

		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.WithError(err).Error("Metrics server error")
		}
	}()

	// Start ping manager
	pingManager := protocol.NewPingManager(s.manager, s.config.PingInterval, s.log.WithField("component", "ping"))
	go pingManager.Start(ctx)

	// Start WebSocket server
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.WithError(err).Error("HTTP server error")
		}
	}()

	<-ctx.Done()

	return s.Stop(context.Background())
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("Stopping ethstats server")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Shutdown metrics server
	if err := s.metricsServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown metrics server: %w", err)
	}

	return nil
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract client IP
	ip := s.extractIPAddress(r)

	// Log connection attempt
	s.log.WithFields(logrus.Fields{
		"ip":      ip,
		"uri":     r.RequestURI,
		"path":    r.URL.Path,
		"origin":  r.Header.Get("Origin"),
		"headers": r.Header,
	}).Info("WebSocket connection attempt")

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"headers": r.Header,
			"method":  r.Method,
		}).Error("Failed to upgrade connection")

		return
	}

	s.log.WithFields(logrus.Fields{
		"remote_addr": conn.RemoteAddr().String(),
		"local_addr":  conn.LocalAddr().String(),
	}).Info("WebSocket upgrade successful")

	// Create client
	client := connection.NewClient(conn, ip)

	// Add to manager
	if err := s.manager.AddClient(client); err != nil {
		s.log.WithError(err).WithField("ip", ip).Error("Failed to add client")
		conn.Close()

		return
	}

	s.log.WithFields(logrus.Fields{
		"client_id": client.ID(),
		"ip":        ip,
	}).Info("Client connected")

	// Handle client in goroutine with background context
	// Don't use r.Context() as it gets cancelled when the HTTP handler returns
	go s.handleClient(context.Background(), client)
}

func (s *Server) handleClient(ctx context.Context, client *connection.Client) {
	// Store the initial temporary ID assigned by the manager
	tempID := fmt.Sprintf("temp_%p", client)
	s.log.WithField("temp_id", tempID).Info("Starting client handler")

	defer func() {
		clientID := client.ID()
		// Use the actual client ID if set, otherwise use the temp ID
		removeID := clientID
		if removeID == "" {
			removeID = tempID
		}

		s.log.WithFields(logrus.Fields{
			"client_id": clientID,
			"temp_id":   tempID,
			"remove_id": removeID,
		}).Info("Cleaning up client")
		client.Close()
		s.manager.RemoveClient(removeID)
	}()

	// Create client context
	clientCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start write pump
	go s.writePump(clientCtx, client)

	// Start read pump
	s.readPump(clientCtx, client)
}

func (s *Server) readPump(ctx context.Context, client *connection.Client) {
	// Configure connection
	conn := client.GetConn()
	conn.SetReadLimit(s.config.MaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done():
			return
		default:
			// Read message
			_, message, err := conn.ReadMessage()
			if err != nil {
				// Check if this is a normal close or an error we should log
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.log.WithError(err).Warn("WebSocket unexpected close error")
				} else if !strings.Contains(err.Error(), "use of closed network connection") {
					// Only log if it's not the expected "use of closed network connection" error
					s.log.WithError(err).Debug("WebSocket connection closed")
				}

				return
			}

			// Update last seen
			client.UpdateLastSeen()

			_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

			clientID := client.ID()
			if clientID == "" {
				clientID = "unauthenticated"
			}

			s.log.WithFields(logrus.Fields{
				"client_id": clientID,
				"size":      len(message),
				"preview":   string(message[:minInt(100, len(message))]),
			}).Info("Received message")

			// Handle message
			if err := s.handler.HandleMessage(ctx, client, message); err != nil {
				s.log.WithError(err).Error("Failed to handle message")
			}
		}
	}
}

func (s *Server) writePump(ctx context.Context, client *connection.Client) {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done():
			return
		case <-ticker.C:
			conn := client.GetConn()
			_ = conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))

			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) extractIPAddress(r *http.Request) string {
	// Try X-Forwarded-For first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Try X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host := r.RemoteAddr
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	return host
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
