package connection

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/protocol"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	// Create a test WebSocket server
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer conn.Close()

	client := NewClient(conn, "127.0.0.1")

	assert.NotNil(t, client)
	assert.Equal(t, conn, client.conn)
	assert.Equal(t, "127.0.0.1", client.ip)
	assert.False(t, client.authenticated)
	assert.Empty(t, client.id)
	assert.NotNil(t, client.done)
	assert.WithinDuration(t, time.Now(), client.connectedAt, time.Second)
	assert.WithinDuration(t, time.Now(), client.lastSeen, time.Second)
}

func TestClient_SetAuthenticated(t *testing.T) {
	client := &Client{
		mu: sync.RWMutex{},
	}

	nodeInfo := &protocol.NodeInfo{
		Name:     "test-node",
		Node:     "Geth/v1.0.0",
		Port:     30303,
		Net:      "1",
		Protocol: "eth/66",
	}

	client.SetAuthenticated("test-id", "testuser", "admin", nodeInfo)

	assert.Equal(t, "test-id", client.ID())
	assert.True(t, client.IsAuthenticated())
	assert.Equal(t, "testuser", client.GetUsername())
	assert.Equal(t, "admin", client.GetGroup())
	assert.Equal(t, nodeInfo, client.GetNodeInfo())
}

func TestClient_UpdateLastSeen(t *testing.T) {
	client := &Client{
		mu:       sync.RWMutex{},
		lastSeen: time.Now().Add(-time.Hour),
	}

	oldLastSeen := client.GetLastSeen()
	time.Sleep(10 * time.Millisecond)

	client.UpdateLastSeen()
	newLastSeen := client.GetLastSeen()

	assert.True(t, newLastSeen.After(oldLastSeen))
	assert.WithinDuration(t, time.Now(), newLastSeen, time.Second)
}

func TestClient_SendMessage(t *testing.T) {
	// Create a test WebSocket server that echoes messages
	upgrader := websocket.Upgrader{}
	receivedMsg := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err == nil {
			receivedMsg <- msg
		}
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer conn.Close()

	client := NewClient(conn, "127.0.0.1")

	testMsg := []byte(`{"test": "message"}`)
	err = client.SendMessage(testMsg)
	assert.NoError(t, err)

	// Wait for message to be received
	select {
	case msg := <-receivedMsg:
		assert.Equal(t, testMsg, msg)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestClient_Close(t *testing.T) {
	// Create a test WebSocket server
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Keep connection open
		<-make(chan struct{})
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	client := NewClient(conn, "127.0.0.1")

	// First close should succeed
	err = client.Close()
	assert.NoError(t, err)

	// Verify done channel is closed
	select {
	case <-client.Done():
		// Expected
	default:
		t.Fatal("done channel should be closed")
	}

	// Second close should not panic (due to closeOnce)
	err = client.Close()
	assert.NoError(t, err)
}

func TestClient_Getters(t *testing.T) {
	connectedAt := time.Now().Add(-time.Hour)
	lastSeen := time.Now().Add(-time.Minute)

	nodeInfo := &protocol.NodeInfo{
		Name: "test-node",
	}

	client := &Client{
		mu:            sync.RWMutex{},
		id:            "test-id",
		username:      "testuser",
		group:         "admin",
		ip:            "192.168.1.1",
		connectedAt:   connectedAt,
		lastSeen:      lastSeen,
		authenticated: true,
		nodeInfo:      nodeInfo,
	}

	assert.Equal(t, "test-id", client.ID())
	assert.Equal(t, "testuser", client.GetUsername())
	assert.Equal(t, "admin", client.GetGroup())
	assert.Equal(t, "192.168.1.1", client.GetIP())
	assert.Equal(t, connectedAt, client.GetConnectedAt())
	assert.Equal(t, lastSeen, client.GetLastSeen())
	assert.True(t, client.IsAuthenticated())
	assert.Equal(t, nodeInfo, client.GetNodeInfo())
}

func TestClient_ConcurrentAccess(t *testing.T) {
	client := &Client{
		mu:       sync.RWMutex{},
		lastSeen: time.Now(),
	}

	// Test concurrent reads and writes
	var wg sync.WaitGroup
	iterations := 100

	// Reader goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = client.ID()
				_ = client.IsAuthenticated()
				_ = client.GetLastSeen()
			}
		}()
	}

	// Writer goroutines
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				client.UpdateLastSeen()
				if j%10 == 0 {
					client.SetAuthenticated(
						"id",
						"user",
						"group",
						&protocol.NodeInfo{Name: "node"},
					)
				}
			}
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, the test passes
}
