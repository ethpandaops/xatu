package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Mock implementations
type MockMetrics struct {
	mu    sync.Mutex
	calls map[string][]interface{}
}

func NewMockMetrics() *MockMetrics {
	return &MockMetrics{
		calls: make(map[string][]interface{}),
	}
}

func (m *MockMetrics) IncProtocolError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncProtocolError"] = append(m.calls["IncProtocolError"], errorType)
}

func (m *MockMetrics) IncAuthentication(status, group string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncAuthentication"] = append(m.calls["IncAuthentication"], []any{status, group})
}

func (m *MockMetrics) IncConnectedClients(client, network string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncConnectedClients"] = append(m.calls["IncConnectedClients"], []any{client, network})
}

func (m *MockMetrics) IncMessagesSent(msgType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncMessagesSent"] = append(m.calls["IncMessagesSent"], msgType)
}

func (m *MockMetrics) IncMessagesReceived(msgType, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncMessagesReceived"] = append(m.calls["IncMessagesReceived"], []any{msgType, clientID})
}

func (m *MockMetrics) AssertCalled(t *testing.T, method string, times int) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	assert.Equal(t, times, len(m.calls[method]), "Expected %s to be called %d times, but was called %d times", method, times, len(m.calls[method]))
}

type MockAuth struct {
	authorizeFunc func(secret string) (username, group string, err error)
}

func (m *MockAuth) AuthorizeSecret(secret string) (username, group string, err error) {
	if m.authorizeFunc != nil {
		return m.authorizeFunc(secret)
	}

	return "", "", errors.New("not implemented")
}

type MockConnectionManager struct{}

func (m *MockConnectionManager) BroadcastMessage(msg []byte) {
	// Mock implementation
}

func (m *MockConnectionManager) AddClient(client ClientInterface) {}
func (m *MockConnectionManager) RemoveClient(id string)           {}
func (m *MockConnectionManager) GetClient(id string) (ClientInterface, bool) {
	return nil, false
}

type MockClient struct {
	authenticated bool
	id            string
	nodeInfo      *NodeInfo
	username      string
	group         string
	sentMessages  [][]byte
	sendError     error
}

func (m *MockClient) ID() string {
	return m.id
}

func (m *MockClient) IsAuthenticated() bool {
	return m.authenticated
}

func (m *MockClient) SetAuthenticated(id, username, group string, nodeInfo *NodeInfo) {
	m.authenticated = true
	m.id = id
	m.username = username
	m.group = group
	m.nodeInfo = nodeInfo
}

func (m *MockClient) SendMessage(msg []byte) error {
	if m.sendError != nil {
		return m.sendError
	}
	m.sentMessages = append(m.sentMessages, msg)

	return nil
}

func TestNewHandler(t *testing.T) {
	log := logrus.New()
	metrics := NewMockMetrics()
	auth := &MockAuth{}
	manager := &MockConnectionManager{}

	handler := NewHandler(log, metrics, auth, manager)

	assert.NotNil(t, handler)
	assert.Equal(t, log, handler.log)
}

func TestHandler_HandleMessage_InvalidJSON(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{}

	err := handler.HandleMessage(context.Background(), client, []byte("invalid json"))

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed to parse message")
	}
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncProtocolError", 1)
}

func TestHandler_HandleMessage_EmptyEmit(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{}

	msg := `{"emit":[]}`
	err := handler.HandleMessage(context.Background(), client, []byte(msg))

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "empty emit array")
	}
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncProtocolError", 1)
}

func TestHandler_HandleMessage_InvalidMessageType(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{id: "test-client"}

	msg := `{"emit":[123, {}]}`
	err := handler.HandleMessage(context.Background(), client, []byte(msg))

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid message type")
	}
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncProtocolError", 1)
}

func TestHandler_HandleMessage_Hello_Success(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{}

	// Setup auth mock
	handler.auth.(*MockAuth).authorizeFunc = func(secret string) (string, string, error) {
		if secret == "test-secret" {
			return "testuser", "admin", nil
		}

		return "", "", errors.New("invalid secret")
	}

	helloMsg := createHelloMessage("test-node", "test-secret")

	err := handler.HandleMessage(context.Background(), client, helloMsg)

	assert.NoError(t, err)
	assert.True(t, client.authenticated)
	assert.Equal(t, "test-node", client.id)
	assert.Equal(t, "testuser", client.username)
	assert.Equal(t, "admin", client.group)
	assert.Len(t, client.sentMessages, 1) // Ready message

	handler.metrics.(*MockMetrics).AssertCalled(t, "IncAuthentication", 1)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncConnectedClients", 1)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesSent", 1)
}

func TestHandler_HandleMessage_Hello_AuthFailure(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{}

	// Setup auth mock to fail
	handler.auth.(*MockAuth).authorizeFunc = func(secret string) (string, string, error) {
		return "", "", errors.New("invalid credentials")
	}

	helloMsg := createHelloMessage("test-node", "wrong-secret")

	err := handler.HandleMessage(context.Background(), client, helloMsg)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "authentication failed")
	}
	assert.False(t, client.authenticated)

	handler.metrics.(*MockMetrics).AssertCalled(t, "IncAuthentication", 1)
}

func TestHandler_HandleMessage_Block_Authenticated(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	blockMsg := createBlockMessage("test-node", 12345678)

	err := handler.HandleMessage(context.Background(), client, blockMsg)

	assert.NoError(t, err)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
}

func TestHandler_HandleMessage_Block_NotAuthenticated(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: false, id: "test-node"}

	blockMsg := createBlockMessage("test-node", 12345678)

	err := handler.HandleMessage(context.Background(), client, blockMsg)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "client not authenticated")
	}
}

func TestHandler_HandleMessage_Stats(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	statsMsg := createStatsMessage("test-node", true, false, 25)

	err := handler.HandleMessage(context.Background(), client, statsMsg)

	assert.NoError(t, err)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
}

func TestHandler_HandleMessage_Pending(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	pendingMsg := createPendingMessage("test-node", 150)

	err := handler.HandleMessage(context.Background(), client, pendingMsg)

	assert.NoError(t, err)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
}

func TestHandler_HandleMessage_NodePing(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	pingMsg := createNodePingMessage("test-node", "1234567890")

	err := handler.HandleMessage(context.Background(), client, pingMsg)

	assert.NoError(t, err)
	assert.Len(t, client.sentMessages, 1) // Pong response

	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesSent", 1)
}

func TestHandler_HandleMessage_Latency(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	latencyMsg := createLatencyMessage("test-node", 25)

	err := handler.HandleMessage(context.Background(), client, latencyMsg)

	assert.NoError(t, err)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
}

func TestHandler_HandleMessage_History(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	historyMsg := createHistoryMessage("test-node", 1)

	err := handler.HandleMessage(context.Background(), client, historyMsg)

	assert.NoError(t, err)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
}

func TestHandler_HandleMessage_PrimusPong(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{authenticated: true, id: "test-node"}

	primusPongMsg := createPrimusPongMessage("1234567890")

	err := handler.HandleMessage(context.Background(), client, primusPongMsg)

	assert.NoError(t, err)
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncMessagesReceived", 1)
}

func TestHandler_HandleMessage_UnknownType(t *testing.T) {
	handler := createTestHandler()
	client := &MockClient{id: "test-node"}

	msg := `{"emit":["unknown-type", {}]}`

	err := handler.HandleMessage(context.Background(), client, []byte(msg))

	assert.NoError(t, err) // Unknown messages don't return error
	handler.metrics.(*MockMetrics).AssertCalled(t, "IncProtocolError", 1)
}

// Helper functions
func createTestHandler() *Handler {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	return &Handler{
		log:     log,
		metrics: NewMockMetrics(),
		auth:    &MockAuth{},
		manager: &MockConnectionManager{},
	}
}

func createHelloMessage(id, secret string) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{
			"hello",
			map[string]interface{}{
				"id": id,
				"info": map[string]interface{}{
					"name":             id,
					"node":             "Geth/v1.10.0",
					"port":             30303,
					"net":              "1",
					"protocol":         "eth/66",
					"api":              "no",
					"os":               "linux",
					"os_v":             "x64",
					"client":           "test-client",
					"canUpdateHistory": true,
				},
				"secret": secret,
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createBlockMessage(id string, blockNumber uint64) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{
			"block",
			map[string]interface{}{
				"id": id,
				"block": map[string]interface{}{
					"number":           blockNumber,
					"hash":             "0x1234",
					"parentHash":       "0x5678",
					"timestamp":        time.Now().Unix(),
					"miner":            "0xabc",
					"gasUsed":          15000000,
					"gasLimit":         30000000,
					"difficulty":       "1234567890",
					"totalDifficulty":  "12345678901234567890",
					"transactions":     []interface{}{},
					"transactionsRoot": "0xtxroot",
					"stateRoot":        "0xstateroot",
					"uncles":           []interface{}{},
				},
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createStatsMessage(id string, active, syncing bool, peers int) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{
			"stats",
			map[string]interface{}{
				"id": id,
				"stats": map[string]interface{}{
					"active":   active,
					"syncing":  syncing,
					"mining":   false,
					"hashrate": 0,
					"peers":    peers,
					"gasPrice": 30000000000,
					"uptime":   100,
				},
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createPendingMessage(id string, pending int) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{
			"pending",
			map[string]interface{}{
				"id": id,
				"stats": map[string]interface{}{
					"pending": pending,
				},
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createNodePingMessage(id, clientTime string) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{
			"node-ping",
			map[string]interface{}{
				"id":         id,
				"clientTime": clientTime,
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createLatencyMessage(id string, latency int) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{
			"latency",
			map[string]interface{}{
				"id":      id,
				"latency": latency,
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createHistoryMessage(id string, blockCount int) []byte {
	blocks := make([]interface{}, blockCount)
	for i := 0; i < blockCount; i++ {
		blocks[i] = map[string]interface{}{
			"number":           12345677 + i,
			"hash":             "0x1234",
			"parentHash":       "0x5678",
			"timestamp":        time.Now().Unix(),
			"miner":            "0xabc",
			"gasUsed":          15000000,
			"gasLimit":         30000000,
			"difficulty":       "1234567890",
			"totalDifficulty":  "12345678901234567890",
			"transactions":     []interface{}{},
			"transactionsRoot": "0xtxroot",
			"stateRoot":        "0xstateroot",
			"uncles":           []interface{}{},
		}
	}

	msg := map[string]interface{}{
		"emit": []interface{}{
			"history",
			map[string]interface{}{
				"id":      id,
				"history": blocks,
			},
		},
	}
	data, _ := json.Marshal(msg)

	return data
}

func createPrimusPongMessage(timestamp string) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{"primus::pong::", timestamp},
	}
	data, _ := json.Marshal(msg)

	return data
}
