package server

type Override struct {
	// MetricsAddr configures the metrics address.
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
	// EventIngesterBasicAuth configures basic authentication for the event ingester.
	EventIngesterBasicAuth struct {
		Username string
		Password string
	}
	// CoordinatorAuth configures the authentication for the coordinator.
	CoordinatorAuth struct {
		Enabled    bool
		AuthSecret string
	}
	// EventIngesterMetaNetworkName rewrites the client-asserted network name
	// on every ingested event.
	EventIngesterMetaNetworkName struct {
		Value  string
		Prefix string
		Suffix string
	}
}
