package connection

// MetricsInterface defines the methods needed by the connection manager
type MetricsInterface interface {
	IncConnectedClients(nodeType, network string)
	DecConnectedClients(nodeType, network string)
	IncIPRateLimitWarning(ip string)
	ObserveConnectionDuration(duration float64, nodeType string)
}
