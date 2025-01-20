package mimicry

// Override is the set of overrides for the mimicry command.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
}
