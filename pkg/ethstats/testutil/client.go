package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/protocol"
	"github.com/gorilla/websocket"
)

// TestClient is a mock ethstats client for testing
type TestClient struct {
	mu               sync.RWMutex
	conn             *websocket.Conn
	url              string
	nodeInfo         protocol.NodeInfo
	secret           string
	receivedMessages []json.RawMessage
	connected        bool
	authenticated    bool
	done             chan struct{}
	doneOnce         sync.Once
	errors           []error
}

// NewTestClient creates a new test client
func NewTestClient(serverURL, nodeID, secret string) *TestClient {
	return &TestClient{
		url:    serverURL,
		secret: secret,
		nodeInfo: protocol.NodeInfo{
			Name:             nodeID,
			Node:             "TestClient/v1.0.0",
			Port:             30303,
			Net:              "1",
			Protocol:         "eth/66,eth/67",
			API:              "no",
			OS:               "linux",
			OSVersion:        "x64",
			Client:           "test-client",
			CanUpdateHistory: true,
		},
		done: make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the server
func (c *TestClient) Connect(ctx context.Context) error {
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	// Ethstats server handles WebSocket connections at root path
	u.Path = "/"

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	// Start reading messages
	go c.readLoop()

	return nil
}

// SendHello sends the initial hello message
func (c *TestClient) SendHello() error {
	hello := protocol.HelloMessage{
		ID:     c.nodeInfo.Name,
		Info:   c.nodeInfo,
		Secret: c.secret,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"hello", hello},
	}

	return c.sendMessage(msg)
}

// SendBlock sends a block report
func (c *TestClient) SendBlock(block *protocol.Block) error {
	report := protocol.BlockReport{
		ID:    c.nodeInfo.Name,
		Block: *block,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"block", report},
	}

	return c.sendMessage(msg)
}

// SendStats sends a stats report
func (c *TestClient) SendStats(stats protocol.NodeStats) error {
	report := protocol.StatsReport{
		ID:    c.nodeInfo.Name,
		Stats: stats,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"stats", report},
	}

	return c.sendMessage(msg)
}

// SendPending sends a pending transactions report
func (c *TestClient) SendPending(pending int) error {
	report := protocol.PendingReport{
		ID: c.nodeInfo.Name,
		Stats: protocol.PendingStats{
			Pending: pending,
		},
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"pending", report},
	}

	return c.sendMessage(msg)
}

// SendNodePing sends a node ping message
func (c *TestClient) SendNodePing() error {
	ping := protocol.NodePing{
		ID:         c.nodeInfo.Name,
		ClientTime: fmt.Sprintf("%d", time.Now().UnixMilli()),
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"node-ping", ping},
	}

	return c.sendMessage(msg)
}

// SendLatency sends a latency report
func (c *TestClient) SendLatency(latency int) error {
	report := protocol.LatencyReport{
		ID:      c.nodeInfo.Name,
		Latency: latency,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"latency", report},
	}

	return c.sendMessage(msg)
}

// SendHistory sends a history report
func (c *TestClient) SendHistory(blocks []protocol.Block) error {
	report := protocol.HistoryReport{
		ID:      c.nodeInfo.Name,
		History: blocks,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"history", report},
	}

	return c.sendMessage(msg)
}

// SendPrimusPong sends a primus pong message
func (c *TestClient) SendPrimusPong(timestamp string) error {
	msg := map[string]interface{}{
		"emit": []interface{}{"primus::pong::", timestamp},
	}

	return c.sendMessage(msg)
}

// WaitForMessage waits for a specific message type with timeout
func (c *TestClient) WaitForMessage(msgType string, timeout time.Duration) (json.RawMessage, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c.mu.RLock()
		messages := c.receivedMessages
		c.mu.RUnlock()

		for _, msg := range messages {
			var parsed map[string]interface{}
			if err := json.Unmarshal(msg, &parsed); err != nil {
				continue
			}

			if emit, ok := parsed["emit"].([]interface{}); ok && len(emit) > 0 {
				if emit[0] == msgType {
					return msg, nil
				}
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for message type: %s", msgType)
}

// IsAuthenticated returns whether the client received a ready message
func (c *TestClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.authenticated
}

// GetReceivedMessages returns all received messages
func (c *TestClient) GetReceivedMessages() []json.RawMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	messages := make([]json.RawMessage, len(c.receivedMessages))
	copy(messages, c.receivedMessages)

	return messages
}

// GetErrors returns any errors encountered
func (c *TestClient) GetErrors() []error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	errors := make([]error, len(c.errors))
	copy(errors, c.errors)

	return errors
}

// Close closes the client connection
func (c *TestClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error

	if c.conn != nil {
		c.doneOnce.Do(func() {
			close(c.done)
		})

		err = c.conn.Close()
	}

	return err
}

func (c *TestClient) sendMessage(msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *TestClient) readLoop() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				c.mu.Lock()
				c.errors = append(c.errors, err)
				c.mu.Unlock()

				return
			}

			c.mu.Lock()
			c.receivedMessages = append(c.receivedMessages, message)

			// Check if it's a ready message
			var parsed map[string]interface{}
			if err := json.Unmarshal(message, &parsed); err == nil {
				if emit, ok := parsed["emit"].([]interface{}); ok && len(emit) > 0 {
					if emit[0] == "ready" {
						c.authenticated = true
					}
				}
			}
			c.mu.Unlock()
		}
	}
}

// CreateTestBlock creates a test block with sample data
func CreateTestBlock(number uint64, parentHash string) protocol.Block {
	// Create BlockNumber by marshaling and unmarshaling
	blockNum := &protocol.BlockNumber{}
	data, _ := json.Marshal(number)
	_ = blockNum.UnmarshalJSON(data)

	return protocol.Block{
		Number:          *blockNum,
		Hash:            fmt.Sprintf("0x%064x", number),
		ParentHash:      parentHash,
		Timestamp:       time.Now().Unix(),
		Miner:           "0x0000000000000000000000000000000000000000",
		GasUsed:         15000000,
		GasLimit:        30000000,
		Difficulty:      "1",
		TotalDifficulty: "1000000",
		Transactions: []protocol.TxHash{
			{Hash: "0x1234567890abcdef"},
			{Hash: "0xfedcba0987654321"},
		},
		TransactionsRoot: "0xabcdef",
		StateRoot:        "0x123456",
		Uncles:           []string{},
	}
}

// CreateTestStats creates test node statistics
func CreateTestStats(active, syncing, mining bool, peers int) protocol.NodeStats {
	return protocol.NodeStats{
		Active:   active,
		Syncing:  syncing,
		Mining:   mining,
		Hashrate: 0,
		Peers:    peers,
		GasPrice: 30000000000,
		Uptime:   100,
	}
}
