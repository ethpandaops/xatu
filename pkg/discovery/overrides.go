package discovery

// Override is the set of overrides for the discovery command.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
}
