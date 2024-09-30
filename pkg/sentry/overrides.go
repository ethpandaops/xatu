package sentry

type Override struct {
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
