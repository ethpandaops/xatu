package http

import (
	"net/http"
	"testing"
)

func TestNewTransportSizesPoolToWorkers(t *testing.T) {
	cases := []struct {
		name    string
		workers int
		want    int
	}{
		{name: "default", workers: 1, want: 1},
		{name: "moderate", workers: 64, want: 64},
		{name: "high", workers: 2048, want: 2048},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := newTransport(&Config{Workers: tc.workers})

			if got.MaxConnsPerHost != tc.want {
				t.Errorf("MaxConnsPerHost = %d, want %d", got.MaxConnsPerHost, tc.want)
			}

			if got.MaxIdleConnsPerHost != tc.want {
				t.Errorf("MaxIdleConnsPerHost = %d, want %d", got.MaxIdleConnsPerHost, tc.want)
			}

			if got.MaxIdleConns < tc.want {
				t.Errorf("MaxIdleConns = %d, want >= %d", got.MaxIdleConns, tc.want)
			}
		})
	}
}

func TestNewTransportNoWorkersKeepsDefaults(t *testing.T) {
	defaults, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatal("http.DefaultTransport is not *http.Transport")
	}

	got := newTransport(&Config{Workers: 0})

	if got.MaxConnsPerHost != defaults.MaxConnsPerHost {
		t.Errorf("MaxConnsPerHost = %d, want default %d", got.MaxConnsPerHost, defaults.MaxConnsPerHost)
	}

	if got.MaxIdleConnsPerHost != defaults.MaxIdleConnsPerHost {
		t.Errorf("MaxIdleConnsPerHost = %d, want default %d", got.MaxIdleConnsPerHost, defaults.MaxIdleConnsPerHost)
	}
}
