package relaymonitor

import (
	"errors"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Type set but Data absent used to silently leave l.Value as an empty
// string, since the old code only marshaled when data was present. Marshal
// should now reject the request instead.
func TestMarshal_TypeSetDataAbsent_ReturnsError(t *testing.T) {
	cases := []struct {
		name string
		typ  xatu.RelayMonitorType
	}{
		{"RELAY_MONITOR_BID_TRACE", xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE},
		{"RELAY_MONITOR_PAYLOAD_DELIVERED", xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := &Location{}

			err := l.Marshal(&xatu.RelayMonitorLocation{
				MetaNetworkName: "mainnet",
				RelayName:       "flashbots",
				Type:            tc.typ,
				// Data: intentionally left nil
			})
			if err == nil {
				t.Fatalf("expected an error, got nil (l.Value = %q)", l.Value)
			}

			if !errors.Is(err, ErrLocationDataRequired) {
				t.Fatalf("expected ErrLocationDataRequired, got: %v", err)
			}

			if l.Value != "" {
				t.Fatalf("l.Value should be untouched on rejection, got %q", l.Value)
			}
		})
	}
}

// Sanity check the happy path still round-trips correctly.
func TestMarshal_WithData_RoundTrips(t *testing.T) {
	l := &Location{}

	msg := &xatu.RelayMonitorLocation{
		MetaNetworkName: "mainnet",
		RelayName:       "flashbots",
		Type:            xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE,
		Data: &xatu.RelayMonitorLocation_BidTrace{
			BidTrace: &xatu.RelayMonitorLocationBidTrace{},
		},
	}

	if err := l.Marshal(msg); err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if l.Value == "" {
		t.Fatal("expected a non-empty marshaled value")
	}

	got, err := l.Unmarshal()
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.GetBidTrace() == nil {
		t.Fatal("expected BidTrace data to round-trip")
	}
}
