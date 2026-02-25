package consumoor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWriter implements source.Writer for testing health checks.
type mockWriter struct {
	pingErr error
}

func (m *mockWriter) Start(_ context.Context) error                   { return nil }
func (m *mockWriter) Stop(_ context.Context) error                    { return nil }
func (m *mockWriter) Write(_ string, _ *xatu.DecoratedEvent)          {}
func (m *mockWriter) FlushAll(_ context.Context) error                { return nil }
func (m *mockWriter) FlushTables(_ context.Context, _ []string) error { return nil }
func (m *mockWriter) Ping(_ context.Context) error                    { return m.pingErr }

func TestHandleHealthz(t *testing.T) {
	c := &Consumoor{
		log: logrus.NewEntry(logrus.New()),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	c.handleHealthz(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp healthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
	assert.Empty(t, resp.Error)
}

func TestHandleReadyz(t *testing.T) {
	tests := []struct {
		name           string
		pingErr        error
		wantStatus     int
		wantRespStatus string
		wantError      string
	}{
		{
			name:           "healthy writer",
			pingErr:        nil,
			wantStatus:     http.StatusOK,
			wantRespStatus: "ok",
		},
		{
			name:           "unhealthy writer",
			pingErr:        errors.New("connection refused"),
			wantStatus:     http.StatusServiceUnavailable,
			wantRespStatus: "not ready",
			wantError:      "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Consumoor{
				log:    logrus.NewEntry(logrus.New()),
				writer: &mockWriter{pingErr: tt.pingErr},
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

			c.handleReadyz(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var resp healthResponse
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, tt.wantRespStatus, resp.Status)
			assert.Equal(t, tt.wantError, resp.Error)
		})
	}
}
