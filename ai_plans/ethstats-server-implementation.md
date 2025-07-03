# Ethstats Server Implementation Plan

## Executive Summary
> The ethstats server will be a new Xatu subcommand that accepts websocket connections from Ethereum execution clients following the ethstats protocol. Unlike the existing gRPC-based server, this will implement a websocket server to maintain compatibility with the established ethstats protocol used by go-ethereum, Nethermind, Erigon, and Besu. The server will authenticate clients using base64-encoded credentials, store connection metadata, handle protocol messages, and provide metrics about connected nodes.

## Goals & Objectives
### Primary Goals
- Implement a websocket server that fully supports the ethstats protocol for Ethereum execution clients
- Provide authentication and authorization using the existing auth patterns from the event-ingester service
- Collect and store IP-based connection information with basic rate limiting detection

### Secondary Objectives
- Maintain consistency with Xatu's existing architecture and patterns
- Enable future message handling and data pipeline integration
- Provide comprehensive metrics for monitoring connected nodes
- Support both ws:// and wss:// connections

## Solution Overview
### Approach
The ethstats server will be implemented as a new package (`pkg/ethstats`) and subcommand, following the existing Xatu patterns. It will use the Gorilla WebSocket library for server implementation, adapt the event-ingester's authentication system for ethstats protocol compatibility, and maintain per-connection state for handling the protocol's ping/pong and message requirements.

### Key Components
1. **Command Integration**: New `ethstats` subcommand in cmd/ following Cobra patterns
2. **WebSocket Server**: Gorilla WebSocket-based server handling protocol requirements
3. **Authentication**: Adapted from event-ingester with base64 username:password parsing
4. **Connection Manager**: Tracks client connections, IPs, and protocol state
5. **Message Handler**: Processes ethstats protocol messages with proper JSON formatting
6. **Metrics**: Prometheus metrics for connections, messages, and rate limiting

### Architecture Diagram
```
[Execution Clients] --ws/wss--> [Ethstats Server]
                                       |
                                       ├── Auth Manager (Groups/Users)
                                       ├── Connection Manager
                                       ├── Message Handler
                                       └── Metrics Exporter
```

### Architectural Notes
- Consider implementing under `pkg/server/service/ethstats/` to follow existing Xatu service patterns
- The protocol types will include custom unmarshalers for fields like `Block.Number` that can be either hex strings or numbers

### Data Flow
```
Client Connect → WebSocket Upgrade → Hello Message → Auth Check → Ready Response → Message Loop
                                                                                    ↓
                                                              ← Ping/Pong → Message Processing
```

### Expected Outcomes
- Execution clients can connect using standard ethstats protocol URLs (nodename:secret@host:port)
- Server authenticates clients and maintains connection state with IP tracking
- All protocol messages are logged with proper handling of format variations
- Metrics expose connection counts, message rates, and authentication failures
- Foundation for future message processing and data pipeline integration

## Implementation Tasks

### CRITICAL IMPLEMENTATION RULES
1. **NO PLACEHOLDER CODE**: Every implementation must be production-ready. NEVER write "TODO", "in a real implementation", or similar placeholders unless explicitly requested by the user.
2. **CROSS-DIRECTORY TASKS**: Group related changes across directories into single tasks to ensure consistency. Never create isolated changes that require follow-up work in sibling directories.
3. **COMPLETE IMPLEMENTATIONS**: Each task must fully implement its feature including all consumers, type updates, and integration points.
4. **DETAILED SPECIFICATIONS**: Each task must include EXACTLY what to implement, including specific functions, types, and integration points to avoid "breaking change" confusion.
5. **CONTEXT AWARENESS**: Each task is part of a larger system - specify how it connects to other parts.
6. **MAKE BREAKING CHANGES**: Unless explicitly requested by the user, you MUST make breaking changes.

### Visual Dependency Tree
```
cmd/
├── ethstats.go (Task #6: Create ethstats subcommand with Cobra integration)
│
pkg/
├── ethstats/
│   ├── config.go (Task #1: Configuration structs and validation)
│   ├── server.go (Task #4: Main server implementation with WebSocket handling)
│   ├── metrics.go (Task #0: Prometheus metrics for ethstats)
│   │
│   ├── auth/
│   │   ├── config.go (Task #2: Auth configuration structures)
│   │   ├── authorization.go (Task #2: Authorization logic for ethstats)
│   │   ├── groups.go (Task #2: Groups management)
│   │   └── user.go (Task #2: User authentication)
│   │
│   ├── connection/
│   │   ├── manager.go (Task #3: Connection state management)
│   │   ├── client.go (Task #3: Individual client connection state)
│   │   └── ratelimit.go (Task #3: IP-based rate limit detection)
│   │
│   └── protocol/
│       ├── types.go (Task #0: Ethstats protocol message types)
│       ├── handler.go (Task #5: Message handling and routing)
│       └── ping.go (Task #5: Ping/pong protocol handling)
│
example_ethstats.yaml (Task #7: Example configuration file)
```

### Execution Plan

#### Group A: Foundation (Execute all in parallel)
- [x] **Task #0**: Create Prometheus metrics for ethstats
  - Folder: `pkg/ethstats/`
  - File: `metrics.go`
  - Implements:
    ```go
    type Metrics struct {
        connectedClients      *prometheus.GaugeVec    // Labels: node_type, network
        authenticationTotal   *prometheus.CounterVec   // Labels: status, group
        messagesReceivedTotal *prometheus.CounterVec   // Labels: type, node_id
        messagesSentTotal     *prometheus.CounterVec   // Labels: type
        protocolErrors        *prometheus.CounterVec   // Labels: error_type
        connectionDuration    *prometheus.HistogramVec // Labels: node_type
        ipRateLimitWarnings   *prometheus.CounterVec   // Labels: ip
    }
    func NewMetrics(namespace string) *Metrics
    func (m *Metrics) IncConnectedClients(nodeType, network string)
    func (m *Metrics) DecConnectedClients(nodeType, network string)
    func (m *Metrics) IncAuthentication(status, group string)
    func (m *Metrics) IncMessagesReceived(msgType, nodeID string)
    func (m *Metrics) IncMessagesSent(msgType string)
    func (m *Metrics) IncProtocolError(errorType string)
    func (m *Metrics) ObserveConnectionDuration(duration float64, nodeType string)
    func (m *Metrics) IncIPRateLimitWarning(ip string)
    ```
  - Exports: Metrics type and constructor
  - Context: Central metrics collection for the ethstats server

- [x] **Task #0**: Create ethstats protocol types
  - Folder: `pkg/ethstats/protocol/`
  - File: `types.go`
  - Implements:
    ```go
    // Wrapper for all messages
    type Message struct {
        Emit []interface{} `json:"emit"`
    }
    
    // Hello message structures
    type HelloMessage struct {
        ID     string      `json:"id"`
        Info   NodeInfo    `json:"info"`
        Secret string      `json:"secret"`
    }
    
    type NodeInfo struct {
        Name             string `json:"name"`
        Node             string `json:"node"`
        Port             int    `json:"port"`
        Net              string `json:"net"`
        Protocol         string `json:"protocol"`
        API              string `json:"api"`
        OS               string `json:"os"`
        OSVersion        string `json:"os_v"`
        Client           string `json:"client"`
        CanUpdateHistory bool   `json:"canUpdateHistory"`
        Contact          string `json:"contact,omitempty"`
    }
    
    // Block report structures
    type BlockReport struct {
        ID    string `json:"id"`
        Block Block  `json:"block"`
    }
    
    // BlockNumber handles both hex string and number formats
    type BlockNumber struct {
        value uint64
    }
    
    // UnmarshalJSON implements json.Unmarshaler
    func (bn *BlockNumber) UnmarshalJSON(data []byte) error
    func (bn BlockNumber) Value() uint64
    func (bn BlockNumber) MarshalJSON() ([]byte, error)
    
    type Block struct {
        Number           BlockNumber `json:"number"` // Custom type for hex/number handling
        Hash             string      `json:"hash"`
        ParentHash       string      `json:"parentHash"`
        Timestamp        int64       `json:"timestamp"`
        Miner            string      `json:"miner"`
        GasUsed          int64       `json:"gasUsed"`
        GasLimit         int64       `json:"gasLimit"`
        Difficulty       string      `json:"difficulty"`
        TotalDifficulty  string      `json:"totalDifficulty"`
        Transactions     []TxHash    `json:"transactions"`
        TransactionsRoot string      `json:"transactionsRoot"`
        StateRoot        string      `json:"stateRoot"`
        Uncles           []string    `json:"uncles"`
    }
    
    type TxHash struct {
        Hash string `json:"hash"`
    }
    
    // Stats structures
    type StatsReport struct {
        ID    string    `json:"id"`
        Stats NodeStats `json:"stats"`
    }
    
    type NodeStats struct {
        Active   bool  `json:"active"`
        Syncing  bool  `json:"syncing"`
        Mining   bool  `json:"mining"`
        Hashrate int64 `json:"hashrate"`
        Peers    int   `json:"peers"`
        GasPrice int64 `json:"gasPrice"`
        Uptime   int   `json:"uptime"`
    }
    
    // Pending transactions
    type PendingReport struct {
        ID    string       `json:"id"`
        Stats PendingStats `json:"stats"`
    }
    
    type PendingStats struct {
        Pending int `json:"pending"`
    }
    
    // Latency structures
    type NodePing struct {
        ID         string `json:"id"`
        ClientTime string `json:"clientTime"`
    }
    
    type NodePong struct {
        ID         string `json:"id"`
        ClientTime string `json:"clientTime"`
        ServerTime string `json:"serverTime"`
    }
    
    type LatencyReport struct {
        ID      string `json:"id"`
        Latency int    `json:"latency"`
    }
    
    // History structures
    type HistoryRequest struct {
        Max int `json:"max"`
        Min int `json:"min"`
    }
    
    type HistoryReport struct {
        ID      string  `json:"id"`
        History []Block `json:"history"`
    }
    
    // Helper functions
    func ParseMessage(data []byte) (*Message, error)
    func ParseHelloMessage(emit []interface{}) (*HelloMessage, error)
    func ParseBlockReport(emit []interface{}) (*BlockReport, error)
    func ParseStatsReport(emit []interface{}) (*StatsReport, error)
    func ParsePendingReport(emit []interface{}) (*PendingReport, error)
    func ParseNodePing(emit []interface{}) (*NodePing, error)
    func ParseLatencyReport(emit []interface{}) (*LatencyReport, error)
    func ParseHistoryReport(emit []interface{}) (*HistoryReport, error)
    func FormatReadyMessage() []byte
    func FormatNodePong(ping *NodePing, serverTime string) []byte
    func FormatHistoryRequest(max, min int) []byte
    func FormatPrimusPing(timestamp int64) []byte
    ```
  - Exports: All protocol types and parsing functions
  - Context: Core protocol definitions for ethstats communication

#### Group B: Authentication System (Execute after Group A)
- [x] **Task #1**: Create ethstats configuration
  - Folder: `pkg/ethstats/`
  - File: `config.go`
  - Imports:
    - `fmt`
    - `time`
    - `github.com/ethpandaops/xatu/pkg/ethstats/auth` (after Task #2)
  - Implements:
    ```go
    type Config struct {
        Enabled      bool                 `yaml:"enabled" default:"true"`
        Addr         string               `yaml:"addr" default:":8081"`
        MetricsAddr  string               `yaml:"metricsAddr" default:":9090"`
        LoggingLevel string               `yaml:"logging" default:"info"`
        
        // WebSocket settings
        MaxMessageSize int64         `yaml:"maxMessageSize" default:"15728640"` // 15MB
        ReadTimeout    time.Duration `yaml:"readTimeout" default:"60s"`
        WriteTimeout   time.Duration `yaml:"writeTimeout" default:"10s"`
        PingInterval   time.Duration `yaml:"pingInterval" default:"30s"`
        
        // Authentication
        Auth auth.Config `yaml:"auth"`
        
        // Rate limiting
        RateLimit RateLimitConfig `yaml:"rateLimit"`
        
        // Labels for metrics
        Labels map[string]string `yaml:"labels"`
    }
    
    type RateLimitConfig struct {
        Enabled              bool          `yaml:"enabled" default:"true"`
        ConnectionsPerIP     int           `yaml:"connectionsPerIP" default:"10"`
        WindowDuration       time.Duration `yaml:"windowDuration" default:"1m"`
        FailuresBeforeWarn   int           `yaml:"failuresBeforeWarn" default:"5"`
    }
    
    func (c *Config) Validate() error
    func NewDefaultConfig() *Config
    ```
  - Exports: Config, RateLimitConfig types and validation
  - Context: Main configuration for the ethstats server

- [x] **Task #2**: Create ethstats authentication system
  - Folder: `pkg/ethstats/auth/`
  - Files: `config.go`, `authorization.go`, `groups.go`, `user.go`
  - File: `config.go`
  - Implements:
    ```go
    type Config struct {
        Enabled bool                    `yaml:"enabled" default:"true"`
        Groups  map[string]GroupConfig  `yaml:"groups"`
    }
    
    type GroupConfig struct {
        Users map[string]UserConfig `yaml:"users"`
    }
    
    type UserConfig struct {
        Password string `yaml:"password"`
    }
    
    func (c *Config) Validate() error
    func (c *GroupConfig) Validate() error
    func (c *UserConfig) Validate() error
    ```
  - File: `user.go`
  - Implements:
    ```go
    type User struct {
        username string
        password string
    }
    
    func NewUser(username string, cfg UserConfig) (*User, error)
    func (u *User) Username() string
    func (u *User) ValidatePassword(password string) bool
    ```
  - File: `groups.go`
  - Implements:
    ```go
    type Group struct {
        name  string
        users map[string]*User
    }
    
    func NewGroup(name string, cfg GroupConfig) (*Group, error)
    func (g *Group) Name() string
    func (g *Group) ValidateUser(username, password string) bool
    func (g *Group) HasUser(username string) bool
    ```
  - File: `authorization.go`
  - Implements:
    ```go
    type Authorization struct {
        enabled bool
        groups  map[string]*Group
        log     logrus.FieldLogger
    }
    
    func NewAuthorization(log logrus.FieldLogger, cfg Config) (*Authorization, error)
    func (a *Authorization) Start(ctx context.Context) error
    func (a *Authorization) AuthorizeSecret(secret string) (username, group string, err error)
    func (a *Authorization) parseSecret(secret string) (username, password string, err error)
    ```
  - Exports: Authorization system adapted for ethstats protocol
  - Context: Handles base64 encoded username:password authentication

#### Group C: Connection Management (Execute after Group B)
- [x] **Task #3**: Create connection management system
  - Folder: `pkg/ethstats/connection/`
  - Files: `manager.go`, `client.go`, `ratelimit.go`
  - File: `client.go`
  - Imports:
    - `time`
    - `sync`
    - `github.com/gorilla/websocket`
    - `github.com/ethpandaops/xatu/pkg/ethstats/protocol`
  - Implements:
    ```go
    type Client struct {
        mu            sync.RWMutex
        conn          *websocket.Conn
        id            string
        nodeInfo      *protocol.NodeInfo
        username      string
        group         string
        ip            string
        connectedAt   time.Time
        lastSeen      time.Time
        lastPing      time.Time
        authenticated bool
        closeOnce     sync.Once
        done          chan struct{}
    }
    
    func NewClient(conn *websocket.Conn, ip string) *Client
    func (c *Client) ID() string
    func (c *Client) SetAuthenticated(id, username, group string, nodeInfo *protocol.NodeInfo)
    func (c *Client) IsAuthenticated() bool
    func (c *Client) UpdateLastSeen()
    func (c *Client) SendMessage(msg []byte) error
    func (c *Client) Close() error
    func (c *Client) Done() <-chan struct{}
    ```
  - File: `ratelimit.go`
  - Implements:
    ```go
    type RateLimiter struct {
        mu               sync.RWMutex
        connections      map[string]int
        failures         map[string]int
        windowStart      time.Time
        windowDuration   time.Duration
        maxConnections   int
        maxFailures      int
        log              logrus.FieldLogger
    }
    
    func NewRateLimiter(windowDuration time.Duration, maxConn, maxFail int, log logrus.FieldLogger) *RateLimiter
    func (r *RateLimiter) AddConnection(ip string) bool
    func (r *RateLimiter) RemoveConnection(ip string)
    func (r *RateLimiter) AddFailure(ip string) bool
    func (r *RateLimiter) cleanup()
    func (r *RateLimiter) getConnectionCount(ip string) int
    func (r *RateLimiter) getFailureCount(ip string) int
    ```
  - File: `manager.go`
  - Implements:
    ```go
    type Manager struct {
        mu          sync.RWMutex
        clients     map[string]*Client
        clientsByIP map[string]map[string]*Client
        rateLimiter *RateLimiter
        metrics     *ethstats.Metrics
        log         logrus.FieldLogger
    }
    
    func NewManager(rateLimiter *RateLimiter, metrics *ethstats.Metrics, log logrus.FieldLogger) *Manager
    func (m *Manager) AddClient(client *Client) error
    func (m *Manager) RemoveClient(id string)
    func (m *Manager) GetClient(id string) (*Client, bool)
    func (m *Manager) GetClientsByIP(ip string) []*Client
    func (m *Manager) BroadcastMessage(msg []byte)
    func (m *Manager) GetStats() (total, authenticated int)
    ```
  - Exports: Connection management system with rate limiting
  - Context: Manages all client connections and tracks IPs

#### Group D: Server Implementation (Execute after Group C)
- [x] **Task #4**: Create main ethstats server
  - Folder: `pkg/ethstats/`
  - File: `server.go`
  - Imports:
    - `context`
    - `fmt`
    - `net/http`
    - `strings`
    - `time`
    - `github.com/gorilla/websocket`
    - `github.com/sirupsen/logrus`
    - `github.com/ethpandaops/xatu/pkg/ethstats/auth`
    - `github.com/ethpandaops/xatu/pkg/ethstats/connection`
    - `github.com/ethpandaops/xatu/pkg/ethstats/protocol`
    - `github.com/ethpandaops/xatu/pkg/observability`
    - `github.com/prometheus/client_golang/prometheus/promhttp`
  - Implements:
    ```go
    type Server struct {
        config       *Config
        log          logrus.FieldLogger
        auth         *auth.Authorization
        manager      *connection.Manager
        handler      *protocol.Handler
        metrics      *Metrics
        upgrader     websocket.Upgrader
        httpServer   *http.Server
        metricsServer *http.Server
    }
    
    func NewServer(log logrus.FieldLogger, config *Config) (*Server, error)
    func (s *Server) Start(ctx context.Context) error
    func (s *Server) Stop(ctx context.Context) error
    func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)
    func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request)
    func (s *Server) handleClient(ctx context.Context, client *connection.Client)
    func (s *Server) readPump(ctx context.Context, client *connection.Client)
    func (s *Server) writePump(ctx context.Context, client *connection.Client)
    func (s *Server) extractIPAddress(r *http.Request) string
    func (s *Server) startMetricsServer(ctx context.Context) error
    ```
  - Exports: Server type and constructor
  - Context: Main server implementation with WebSocket upgrade and client handling

- [x] **Task #5**: Create protocol message handler
  - Folder: `pkg/ethstats/protocol/`
  - Files: `handler.go`, `ping.go`
  - File: `handler.go`
  - Imports:
    - `context`
    - `encoding/json`
    - `fmt`
    - `time`
    - `github.com/sirupsen/logrus`
    - `github.com/ethpandaops/xatu/pkg/ethstats/connection`
  - Implements:
    ```go
    type Handler struct {
        log     logrus.FieldLogger
        metrics *ethstats.Metrics
        auth    *auth.Authorization
        manager *connection.Manager
    }
    
    func NewHandler(log logrus.FieldLogger, metrics *ethstats.Metrics, auth *auth.Authorization, manager *connection.Manager) *Handler
    func (h *Handler) HandleMessage(ctx context.Context, client *connection.Client, data []byte) error
    func (h *Handler) handleHello(ctx context.Context, client *connection.Client, msg *HelloMessage) error
    func (h *Handler) handleBlock(ctx context.Context, client *connection.Client, msg *BlockReport) error
    func (h *Handler) handleStats(ctx context.Context, client *connection.Client, msg *StatsReport) error
    func (h *Handler) handlePending(ctx context.Context, client *connection.Client, msg *PendingReport) error
    func (h *Handler) handleNodePing(ctx context.Context, client *connection.Client, msg *NodePing) error
    func (h *Handler) handleLatency(ctx context.Context, client *connection.Client, msg *LatencyReport) error
    func (h *Handler) handleHistory(ctx context.Context, client *connection.Client, msg *HistoryReport) error
    func (h *Handler) handlePrimusPong(ctx context.Context, client *connection.Client, timestamp string) error
    ```
  - File: `ping.go`
  - Implements:
    ```go
    type PingManager struct {
        ticker   *time.Ticker
        done     chan struct{}
        manager  *connection.Manager
        interval time.Duration
        log      logrus.FieldLogger
    }
    
    func NewPingManager(manager *connection.Manager, interval time.Duration, log logrus.FieldLogger) *PingManager
    func (p *PingManager) Start(ctx context.Context)
    func (p *PingManager) Stop()
    func (p *PingManager) sendPings()
    ```
  - Exports: Message handling and ping/pong management
  - Context: Processes all ethstats protocol messages

#### Group E: Integration (Execute after Group D)
- [x] **Task #6**: Create ethstats command
  - Folder: `cmd/`
  - File: `ethstats.go`
  - Imports:
    - `context`
    - `os`
    - `os/signal`
    - `syscall`
    - `github.com/sirupsen/logrus`
    - `github.com/spf13/cobra`
    - `github.com/ethpandaops/xatu/pkg/ethstats`
    - `github.com/ethpandaops/xatu/pkg/observability`
  - Implements:
    ```go
    var (
        ethstatsCfgFile string
        ethstatsCmd = &cobra.Command{
            Use:   "ethstats",
            Short: "Runs Xatu in Ethstats server mode",
            Long: `Ethstats server mode accepts WebSocket connections from Ethereum execution clients
    following the ethstats protocol. It authenticates clients, collects node information,
    and handles protocol messages.`,
            Run: func(cmd *cobra.Command, args []string) {
                // Load configuration
                // Initialize logger
                // Setup observability
                // Create and start server
                // Handle shutdown
            },
        }
    )
    
    func init() {
        rootCmd.AddCommand(ethstatsCmd)
        
        ethstatsCmd.Flags().StringVar(&ethstatsCfgFile, "config", "ethstats.yaml", "config file")
    }
    
    func loadEthstatsConfigFromFile(file string) (*ethstats.Config, error)
    ```
  - Exports: Cobra command for ethstats server
  - Context: CLI integration for the ethstats server

- [x] **Task #7**: Create example configuration
  - Folder: Repository root
  - File: `example_ethstats.yaml`
  - Implements:
    ```yaml
    # Ethstats server configuration example
    enabled: true
    addr: ":8081"
    metricsAddr: ":9090"
    logging: "info"
    
    # WebSocket configuration
    maxMessageSize: 15728640  # 15MB (go-ethereum requirement)
    readTimeout: 60s
    writeTimeout: 10s
    pingInterval: 30s
    
    # Authentication configuration
    auth:
      enabled: true
      groups:
        mainnet:
          users:
            node1:
              password: "securepassword1"
            node2:
              password: "securepassword2"
        testnet:
          users:
            testnode1:
              password: "testpass1"
    
    # Rate limiting configuration
    rateLimit:
      enabled: true
      connectionsPerIP: 10
      windowDuration: 1m
      failuresBeforeWarn: 5
    
    # Labels for metrics
    labels:
      environment: production
      service: ethstats
    ```
  - Context: Example configuration demonstrating all features

---

## Implementation Workflow

This plan file serves as the authoritative checklist for implementation. When implementing:

### Required Process
1. **Load Plan**: Read this entire plan file before starting
2. **Sync Tasks**: Create TodoWrite tasks matching the checkboxes below
3. **Execute & Update**: For each task:
   - Mark TodoWrite as `in_progress` when starting
   - Update checkbox `[ ]` to `[x]` when completing
   - Mark TodoWrite as `completed` when done
4. **Maintain Sync**: Keep this file and TodoWrite synchronized throughout

### Critical Rules
- This plan file is the source of truth for progress
- Update checkboxes in real-time as work progresses
- Never lose synchronization between plan file and TodoWrite
- Mark tasks complete only when fully implemented (no placeholders)
- Tasks should be run in parallel, unless there are dependencies, using subtasks, to avoid context bloat.

### Progress Tracking
The checkboxes above represent the authoritative status of each task. Keep them updated as you work.