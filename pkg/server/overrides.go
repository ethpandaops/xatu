package server

type Override struct {
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
}
