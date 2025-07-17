package protocol

// MetricsInterface defines the methods needed by the protocol handler
type MetricsInterface interface {
	IncAuthentication(status, group string)
	IncMessagesReceived(msgType, nodeID string)
	IncMessagesSent(msgType string)
	IncProtocolError(errorType string)
	IncConnectedClients(nodeType, network string)
}

// AuthInterface defines the methods needed for authorization
type AuthInterface interface {
	AuthorizeSecret(secret string) (username, group string, err error)
}

// ConnectionManagerInterface defines the methods needed for connection management
type ConnectionManagerInterface interface {
	BroadcastMessage(msg []byte)
}

// ClientInterface defines the methods needed for client interaction
type ClientInterface interface {
	ID() string
	SetAuthenticated(id, username, group string, nodeInfo *NodeInfo)
	IsAuthenticated() bool
	SendMessage(msg []byte) error
}
