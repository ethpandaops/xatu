package cmd

import (
	"context"
	"time"
)

// shutdownObservability runs the observability shutdown with a bounded
// context so exporter flushing cannot block process exit.
func shutdownObservability(fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := fn(ctx); err != nil {
		log.WithError(err).Warn("observability shutdown reported errors")
	}
}
