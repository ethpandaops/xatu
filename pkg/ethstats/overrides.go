package ethstats

// Override contains runtime overrides for ethstats configuration.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
	NetworkName struct {
		Enabled bool
		Value   string
	}
}
