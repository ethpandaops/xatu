package connection

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/protocol"
	"github.com/gorilla/websocket"
)

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
	authenticated bool
	closeOnce     sync.Once
	done          chan struct{}
}

func NewClient(conn *websocket.Conn, ip string) *Client {
	return &Client{
		conn:        conn,
		ip:          ip,
		connectedAt: time.Now(),
		lastSeen:    time.Now(),
		done:        make(chan struct{}),
	}
}

func (c *Client) ID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.id
}

func (c *Client) SetAuthenticated(id, username, group string, nodeInfo *protocol.NodeInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.id = id
	c.username = username
	c.group = group
	c.nodeInfo = nodeInfo
	c.authenticated = true
}

func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.authenticated
}

func (c *Client) UpdateLastSeen() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSeen = time.Now()
}

func (c *Client) SendMessage(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *Client) Close() error {
	var err error

	c.closeOnce.Do(func() {
		close(c.done)
		err = c.conn.Close()
	})

	return err
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

// GetConn returns the underlying websocket connection
func (c *Client) GetConn() *websocket.Conn {
	return c.conn
}

// GetNodeInfo returns the node info if authenticated
func (c *Client) GetNodeInfo() *protocol.NodeInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.nodeInfo
}

// GetGroup returns the authentication group
func (c *Client) GetGroup() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.group
}

// GetUsername returns the authenticated username
func (c *Client) GetUsername() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.username
}

// GetIP returns the client IP address
func (c *Client) GetIP() string {
	return c.ip
}

// GetConnectedAt returns the connection time
func (c *Client) GetConnectedAt() time.Time {
	return c.connectedAt
}

// GetLastSeen returns the last seen time
func (c *Client) GetLastSeen() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastSeen
}
