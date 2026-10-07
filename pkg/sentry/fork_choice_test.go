package sentry

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/stretchr/testify/assert"
)

func TestIsUnsupportedEndpoint(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "lighthouse rejects the version", err: &api.Error{StatusCode: http.StatusBadRequest}, want: true},
		{name: "nimbus has no route", err: &api.Error{StatusCode: http.StatusNotFound}, want: true},
		{name: "method not allowed", err: &api.Error{StatusCode: http.StatusMethodNotAllowed}, want: true},
		{name: "not implemented", err: &api.Error{StatusCode: http.StatusNotImplemented}, want: true},
		{name: "wrapped", err: fmt.Errorf("fetch: %w", &api.Error{StatusCode: http.StatusNotFound}), want: true},
		{name: "server error", err: &api.Error{StatusCode: http.StatusInternalServerError}, want: false},
		{name: "not an api error", err: errors.New("connection refused"), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isUnsupportedEndpoint(test.err))
		})
	}
}
