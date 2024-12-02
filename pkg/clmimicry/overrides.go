package clmimicry

// Override is the set of overrides for the cl-mimicry command.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
}
