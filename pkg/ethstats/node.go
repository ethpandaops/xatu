package ethstats

import (
	"net"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/auth"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Node represents a connected ethstats client.
type Node struct {
	id   string
	info *NodeInfo
	conn *websocket.Conn
	log  logrus.FieldLogger

	// Authentication context
	user  *auth.User
	group *auth.Group

	// Client IP for geo enrichment
	clientIP net.IP

	// Write configuration
	writeWait time.Duration

	// Mutex for state access
	mu sync.RWMutex

	// Mutex for write operations (separate to avoid blocking reads during writes)
	writeMu sync.Mutex

	// Latest state from node
	latestBlock      *BlockStats
	latestStats      *NodeStats
	latestPending    *PendingStats
	latestNewPayload *NewPayloadStats
	latency          string

	// Connection timestamps
	connectedAt time.Time
}

// NewNode creates a new Node instance.
func NewNode(id string, info *NodeInfo, conn *websocket.Conn, clientIP net.IP, writeWait time.Duration, log logrus.FieldLogger) *Node {
	return &Node{
		id:          id,
		info:        info,
		conn:        conn,
		clientIP:    clientIP,
		writeWait:   writeWait,
		log:         log.WithField("node_id", id),
		connectedAt: time.Now(),
	}
}

// ID returns the node ID.
func (n *Node) ID() string {
	return n.id
}

// Info returns the node info.
func (n *Node) Info() *NodeInfo {
	return n.info
}

// Conn returns the WebSocket connection.
func (n *Node) Conn() *websocket.Conn {
	return n.conn
}

// ClientIP returns the client IP address.
func (n *Node) ClientIP() net.IP {
	return n.clientIP
}

// User returns the authenticated user.
func (n *Node) User() *auth.User {
	return n.user
}

// Group returns the authenticated group.
func (n *Node) Group() *auth.Group {
	return n.group
}

// SetAuth sets the authentication context.
func (n *Node) SetAuth(user *auth.User, group *auth.Group) {
	n.user = user
	n.group = group
}

// ConnectedAt returns the connection timestamp.
func (n *Node) ConnectedAt() time.Time {
	return n.connectedAt
}

// SetLatestBlock updates the latest block.
func (n *Node) SetLatestBlock(block *BlockStats) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.latestBlock = block
}

// LatestBlock returns the latest block.
func (n *Node) LatestBlock() *BlockStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.latestBlock
}

// SetLatestStats updates the latest stats.
func (n *Node) SetLatestStats(stats *NodeStats) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.latestStats = stats
}

// LatestStats returns the latest stats.
func (n *Node) LatestStats() *NodeStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.latestStats
}

// SetLatestPending updates the latest pending stats.
func (n *Node) SetLatestPending(pending *PendingStats) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.latestPending = pending
}

// LatestPending returns the latest pending stats.
func (n *Node) LatestPending() *PendingStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.latestPending
}

// SetLatestNewPayload updates the latest new payload stats.
func (n *Node) SetLatestNewPayload(payload *NewPayloadStats) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.latestNewPayload = payload
}

// LatestNewPayload returns the latest new payload stats.
func (n *Node) LatestNewPayload() *NewPayloadStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.latestNewPayload
}

// SetLatency updates the latency.
func (n *Node) SetLatency(latency string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.latency = latency
}

// Latency returns the latency.
func (n *Node) Latency() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.latency
}

// WriteJSON writes a JSON message to the websocket with proper locking and deadline.
func (n *Node) WriteJSON(v any) error {
	n.writeMu.Lock()
	defer n.writeMu.Unlock()

	if err := n.conn.SetWriteDeadline(time.Now().Add(n.writeWait)); err != nil {
		return err
	}

	return n.conn.WriteJSON(v)
}

// Close closes the node connection.
func (n *Node) Close() error {
	return n.conn.Close()
}
