package sentry

type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
	BeaconNodeURL struct {
		Enabled bool
		Value   string
	}
	XatuOutputAuth struct {
		Enabled bool
		Value   string
	}
	Preset struct {
		Enabled bool
		Value   string
	}
}
