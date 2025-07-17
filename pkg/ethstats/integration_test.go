package ethstats_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/auth"
	"github.com/ethpandaops/xatu/pkg/ethstats/connection"
	"github.com/ethpandaops/xatu/pkg/ethstats/protocol"
	"github.com/ethpandaops/xatu/pkg/ethstats/testutil"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEthstatsProtocol_Integration(t *testing.T) {
	// Create test logger
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create auth with test users
	authCfg := auth.Config{
		Enabled: true,
		Groups: map[string]auth.GroupConfig{
			"admin": {
				Users: map[string]auth.UserConfig{
					"testuser": {Password: "testpass"},
				},
			},
		},
	}
	authz, err := auth.NewAuthorization(log, authCfg)
	require.NoError(t, err)

	// Create metrics
	metrics := NewMockMetrics()

	// Create connection manager
	manager := connection.NewManager(nil, metrics, log)

	// Create protocol handler
	handler := protocol.NewHandler(log, metrics, authz, manager)

	// Create WebSocket test server
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)

			return
		}
		defer conn.Close()

		// Create client
		client := connection.NewClient(conn, r.RemoteAddr)
		if err := manager.AddClient(client); err != nil {
			t.Logf("Add client error: %v", err)

			return
		}
		defer manager.RemoveClient(client.ID())

		// Handle messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					t.Logf("Read error: %v", err)
				}

				break
			}

			if err := handler.HandleMessage(context.Background(), client, message); err != nil {
				t.Logf("Handle error: %v", err)
				// In real server, would close connection on auth failure
				if strings.Contains(err.Error(), "authentication failed") {
					break
				}
			}
		}
	}))
	defer server.Close()

	t.Run("successful authentication and communication", func(t *testing.T) {
		// Create test client
		client := testutil.NewTestClient(server.URL, "test-node", "testuser:testpass")

		// Connect to server
		err := client.Connect(context.Background())
		require.NoError(t, err)
		defer client.Close()

		// Send hello message
		err = client.SendHello()
		require.NoError(t, err)

		// Wait for ready message
		msg, err := client.WaitForMessage("ready", 2*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, msg)
		assert.True(t, client.IsAuthenticated())

		// Send block report
		block := testutil.CreateTestBlock(12345678, "0xparenthash")
		err = client.SendBlock(&block)
		assert.NoError(t, err)

		// Send stats
		stats := testutil.CreateTestStats(true, false, false, 25)
		err = client.SendStats(stats)
		assert.NoError(t, err)

		// Send pending count
		err = client.SendPending(150)
		assert.NoError(t, err)

		// Test ping-pong
		err = client.SendNodePing()
		assert.NoError(t, err)

		pongMsg, err := client.WaitForMessage("node-pong", time.Second)
		require.NoError(t, err)
		assert.NotNil(t, pongMsg)

		// Send latency report
		err = client.SendLatency(25)
		assert.NoError(t, err)

		// Verify metrics were called
		assert.True(t, metrics.WasCalled("IncAuthentication"))
		assert.True(t, metrics.WasCalled("IncConnectedClients"))
		assert.True(t, metrics.WasCalled("IncMessagesReceived"))
		assert.True(t, metrics.WasCalled("IncMessagesSent"))
	})

	t.Run("failed authentication", func(t *testing.T) {
		// Create client with wrong credentials
		client := testutil.NewTestClient(server.URL, "bad-node", "wronguser:wrongpass")

		// Connect to server
		err := client.Connect(context.Background())
		require.NoError(t, err)
		defer client.Close()

		// Send hello message
		err = client.SendHello()
		require.NoError(t, err)

		// Should not receive ready message
		_, err = client.WaitForMessage("ready", 500*time.Millisecond)
		assert.Error(t, err)
		assert.False(t, client.IsAuthenticated())

		// Verify auth failure was recorded
		assert.True(t, metrics.WasCalled("IncAuthentication"))
	})

	t.Run("messages before authentication", func(t *testing.T) {
		// Create test client
		client := testutil.NewTestClient(server.URL, "test-node2", "testuser:testpass")

		// Connect to server
		err := client.Connect(context.Background())
		require.NoError(t, err)
		defer client.Close()

		// Try to send block before authentication
		block := testutil.CreateTestBlock(12345679, "0xparenthash2")
		err = client.SendBlock(&block)
		assert.NoError(t, err) // Send succeeds at client level

		// Should not process the message
		time.Sleep(100 * time.Millisecond)

		// Now authenticate
		err = client.SendHello()
		require.NoError(t, err)

		// Should receive ready message
		_, err = client.WaitForMessage("ready", time.Second)
		require.NoError(t, err)
	})

	t.Run("primus protocol support", func(t *testing.T) {
		// Create test client
		client := testutil.NewTestClient(server.URL, "primus-node", "testuser:testpass")

		// Connect and authenticate
		err := client.Connect(context.Background())
		require.NoError(t, err)
		defer client.Close()

		err = client.SendHello()
		require.NoError(t, err)

		_, err = client.WaitForMessage("ready", 2*time.Second)
		require.NoError(t, err)

		// Send primus pong
		err = client.SendPrimusPong("1234567890")
		assert.NoError(t, err)

		// Verify it was received
		assert.True(t, metrics.WasCalled("IncMessagesReceived"))
	})

	t.Run("history report", func(t *testing.T) {
		// Create test client
		client := testutil.NewTestClient(server.URL, "history-node", "testuser:testpass")

		// Connect and authenticate
		err := client.Connect(context.Background())
		require.NoError(t, err)
		defer client.Close()

		err = client.SendHello()
		require.NoError(t, err)

		_, err = client.WaitForMessage("ready", 2*time.Second)
		require.NoError(t, err)

		// Send history report
		blocks := []protocol.Block{
			testutil.CreateTestBlock(12345676, "0xparenthash0"),
			testutil.CreateTestBlock(12345677, "0xparenthash1"),
			testutil.CreateTestBlock(12345678, "0xparenthash2"),
		}
		err = client.SendHistory(blocks)
		assert.NoError(t, err)

		// Verify it was received
		assert.True(t, metrics.WasCalled("IncMessagesReceived"))
	})
}

func TestEthstatsProtocol_DisabledAuth(t *testing.T) {
	// Create test logger
	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)

	// Create auth with disabled mode
	authCfg := auth.Config{
		Enabled: false,
	}
	authz, err := auth.NewAuthorization(log, authCfg)
	require.NoError(t, err)

	// Create other components
	metrics := NewMockMetrics()
	manager := connection.NewManager(nil, metrics, log)
	handler := protocol.NewHandler(log, metrics, authz, manager)

	// Create WebSocket test server
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, errUpgrade := upgrader.Upgrade(w, r, nil)
		if errUpgrade != nil {
			return
		}
		defer conn.Close()

		client := connection.NewClient(conn, r.RemoteAddr)
		if errAdd := manager.AddClient(client); errAdd != nil {
			return
		}
		defer manager.RemoveClient(client.ID())

		for {
			_, message, readErr := conn.ReadMessage()
			if readErr != nil {
				break
			}
			_ = handler.HandleMessage(context.Background(), client, message)
		}
	}))
	defer server.Close()

	// Create client with any credentials
	client := testutil.NewTestClient(server.URL, "any-node", "any:secret")

	// Connect to server
	err = client.Connect(context.Background())
	require.NoError(t, err)
	defer client.Close()

	// Send hello message
	err = client.SendHello()
	require.NoError(t, err)

	// Should receive ready message even with any credentials
	msg, err := client.WaitForMessage("ready", 2*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.True(t, client.IsAuthenticated())
}

// Mock metrics implementation for testing
type MockMetrics struct {
	mu    sync.Mutex
	calls map[string]int
}

func NewMockMetrics() *MockMetrics {
	return &MockMetrics{
		calls: make(map[string]int),
	}
}

func (m *MockMetrics) IncProtocolError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncProtocolError"]++
}

func (m *MockMetrics) IncAuthentication(status, group string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncAuthentication"]++
}

func (m *MockMetrics) IncConnectedClients(client, network string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncConnectedClients"]++
}

func (m *MockMetrics) IncMessagesSent(msgType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncMessagesSent"]++
}

func (m *MockMetrics) IncMessagesReceived(msgType, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncMessagesReceived"]++
}

func (m *MockMetrics) DecConnectedClients(nodeType, network string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["DecConnectedClients"]++
}

func (m *MockMetrics) IncIPRateLimitWarning(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["IncIPRateLimitWarning"]++
}

func (m *MockMetrics) ObserveConnectionDuration(duration float64, nodeType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ObserveConnectionDuration"]++
}

func (m *MockMetrics) WasCalled(method string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.calls[method] > 0
}

func (m *MockMetrics) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.calls[method]
}
