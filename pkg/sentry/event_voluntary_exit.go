package sentry

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (s *Sentry) handleVoluntaryExit(ctx context.Context, event *phase0.VoluntaryExit) error {
	s.log.Debug("VoluntaryExit received (not supported yet)")

	return nil
}
