package cannon

type Override struct {
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
}
