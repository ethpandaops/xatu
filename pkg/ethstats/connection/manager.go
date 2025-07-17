package connection

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Manager struct {
	mu          sync.RWMutex
	clients     map[string]*Client
	clientsByIP map[string]map[string]*Client
	rateLimiter *RateLimiter
	metrics     MetricsInterface
	log         logrus.FieldLogger
}

func NewManager(rateLimiter *RateLimiter, metrics MetricsInterface, log logrus.FieldLogger) *Manager {
	return &Manager{
		clients:     make(map[string]*Client),
		clientsByIP: make(map[string]map[string]*Client),
		rateLimiter: rateLimiter,
		metrics:     metrics,
		log:         log,
	}
}

func (m *Manager) AddClient(client *Client) error {
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}

	// Check rate limit before adding
	if m.rateLimiter != nil && !m.rateLimiter.AddConnection(client.ip) {
		m.metrics.IncIPRateLimitWarning(client.ip)

		return fmt.Errorf("rate limit exceeded for IP %s", client.ip)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize IP map if needed
	if _, exists := m.clientsByIP[client.ip]; !exists {
		m.clientsByIP[client.ip] = make(map[string]*Client)
	}

	// Add to temporary ID map initially (will be moved after auth)
	tempID := fmt.Sprintf("temp_%p", client)
	m.clients[tempID] = client
	m.clientsByIP[client.ip][tempID] = client

	m.log.WithFields(logrus.Fields{
		"ip":      client.ip,
		"temp_id": tempID,
	}).Debug("Client connected")

	return nil
}

func (m *Manager) RemoveClient(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// First check using the provided ID
	client, exists := m.clients[id]
	if !exists {
		// If not found by ID, it might be a temporary ID
		// Search through all clients to find one that might have been closed
		for tempID, c := range m.clients {
			if c.ID() == id || tempID == id {
				client = c
				id = tempID
				exists = true
				break
			}
		}

		if !exists {
			return
		}
	}

	// Remove from clients map
	delete(m.clients, id)

	// Remove from IP map - check all possible IDs
	if ipClients, exists := m.clientsByIP[client.ip]; exists {
		// Remove by the temp ID
		delete(ipClients, id)

		// Also remove by the actual client ID if different
		actualID := client.ID()
		if actualID != "" && actualID != id {
			delete(ipClients, actualID)
		}

		if len(ipClients) == 0 {
			delete(m.clientsByIP, client.ip)
		}
	}

	// Update rate limiter
	if m.rateLimiter != nil {
		m.rateLimiter.RemoveConnection(client.ip)
	}

	// Update metrics if authenticated
	if client.IsAuthenticated() && client.nodeInfo != nil {
		m.metrics.DecConnectedClients(client.nodeInfo.Client, client.nodeInfo.Net)

		// Calculate connection duration
		duration := float64(client.lastSeen.Sub(client.connectedAt).Seconds())
		m.metrics.ObserveConnectionDuration(duration, client.nodeInfo.Client)
	}

	m.log.WithFields(logrus.Fields{
		"id":        id,
		"client_id": client.ID(),
		"ip":        client.ip,
		"username":  client.username,
		"group":     client.group,
	}).Info("Client disconnected")
}

func (m *Manager) GetClient(id string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[id]

	return client, exists
}

func (m *Manager) GetClientsByIP(ip string) []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*Client, 0)

	if ipClients, exists := m.clientsByIP[ip]; exists {
		for _, client := range ipClients {
			clients = append(clients, client)
		}
	}

	return clients
}

func (m *Manager) BroadcastMessage(msg []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		if client.IsAuthenticated() {
			if err := client.SendMessage(msg); err != nil {
				m.log.WithError(err).WithField("id", client.ID()).Error("Failed to send message to client")
			}
		}
	}
}

func (m *Manager) GetStats() (total, authenticated int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.clients)

	for _, client := range m.clients {
		if client.IsAuthenticated() {
			authenticated++
		}
	}

	return total, authenticated
}
