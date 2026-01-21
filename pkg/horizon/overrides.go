package horizon

type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
	XatuOutputAuth struct {
		Enabled bool
		Value   string
	}
}
