package ethstats

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	connectedClients      *prometheus.GaugeVec
	authenticationTotal   *prometheus.CounterVec
	messagesReceivedTotal *prometheus.CounterVec
	messagesSentTotal     *prometheus.CounterVec
	protocolErrors        *prometheus.CounterVec
	connectionDuration    *prometheus.HistogramVec
	ipRateLimitWarnings   *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		connectedClients: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "connected_clients",
				Help:      "Number of currently connected clients",
			},
			[]string{"node_type", "network"},
		),
		authenticationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "authentication_total",
				Help:      "Total number of authentication attempts",
			},
			[]string{"status", "group"},
		),
		messagesReceivedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_received_total",
				Help:      "Total number of messages received from clients",
			},
			[]string{"type", "node_id"},
		),
		messagesSentTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_sent_total",
				Help:      "Total number of messages sent to clients",
			},
			[]string{"type"},
		),
		protocolErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "protocol_errors_total",
				Help:      "Total number of protocol errors",
			},
			[]string{"error_type"},
		),
		connectionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "connection_duration_seconds",
				Help:      "Duration of client connections in seconds",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
			},
			[]string{"node_type"},
		),
		ipRateLimitWarnings: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "ip_rate_limit_warnings_total",
				Help:      "Total number of IP rate limit warnings",
			},
			[]string{"ip"},
		),
	}

	prometheus.MustRegister(
		m.connectedClients,
		m.authenticationTotal,
		m.messagesReceivedTotal,
		m.messagesSentTotal,
		m.protocolErrors,
		m.connectionDuration,
		m.ipRateLimitWarnings,
	)

	return m
}

func (m *Metrics) IncConnectedClients(nodeType, network string) {
	m.connectedClients.WithLabelValues(nodeType, network).Inc()
}

func (m *Metrics) DecConnectedClients(nodeType, network string) {
	m.connectedClients.WithLabelValues(nodeType, network).Dec()
}

func (m *Metrics) IncAuthentication(status, group string) {
	m.authenticationTotal.WithLabelValues(status, group).Inc()
}

func (m *Metrics) IncMessagesReceived(msgType, nodeID string) {
	m.messagesReceivedTotal.WithLabelValues(msgType, nodeID).Inc()
}

func (m *Metrics) IncMessagesSent(msgType string) {
	m.messagesSentTotal.WithLabelValues(msgType).Inc()
}

func (m *Metrics) IncProtocolError(errorType string) {
	m.protocolErrors.WithLabelValues(errorType).Inc()
}

func (m *Metrics) ObserveConnectionDuration(duration float64, nodeType string) {
	m.connectionDuration.WithLabelValues(nodeType).Observe(duration)
}

func (m *Metrics) IncIPRateLimitWarning(ip string) {
	m.ipRateLimitWarnings.WithLabelValues(ip).Inc()
}
