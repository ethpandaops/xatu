package relaymonitor

// Override is the set of overrides for the relay monitor command.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
}
