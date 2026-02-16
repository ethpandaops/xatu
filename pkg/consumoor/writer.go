package consumoor

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Writer writes flattened rows to ClickHouse.
type Writer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Write(table string, rows []map[string]any)
}

// NewWriter creates a ClickHouse writer implementation based on config.
func NewWriter(
	log logrus.FieldLogger,
	config *ClickHouseConfig,
	metrics *Metrics,
) (Writer, error) {
	switch config.Backend {
	case "ch-go":
		return NewChGoWriter(log, config, metrics)
	default:
		return NewClickHouseWriter(log, config, metrics)
	}
}
