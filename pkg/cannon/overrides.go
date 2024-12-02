package cannon

type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
	BeaconNodeURL struct {
		Enabled bool
		Value   string
	}
	BeaconNodeAuthorizationHeader struct {
		Enabled bool
		Value   string
	}
	XatuOutputAuth struct {
		Enabled bool
		Value   string
	}
	XatuCoordinatorAuth struct {
		Enabled bool
		Value   string
	}
	NetworkName struct {
		Enabled bool
		Value   string
	}
}
