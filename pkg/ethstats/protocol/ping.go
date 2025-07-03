package protocol

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// PingManagerInterface defines the methods needed by ping manager
type PingManagerInterface interface {
	BroadcastMessage(msg []byte)
}

type PingManager struct {
	ticker   *time.Ticker
	done     chan struct{}
	manager  PingManagerInterface
	interval time.Duration
	log      logrus.FieldLogger
}

func NewPingManager(manager PingManagerInterface, interval time.Duration, log logrus.FieldLogger) *PingManager {
	return &PingManager{
		manager:  manager,
		interval: interval,
		log:      log,
		done:     make(chan struct{}),
	}
}

func (p *PingManager) Start(ctx context.Context) {
	p.ticker = time.NewTicker(p.interval)
	defer p.ticker.Stop()

	p.log.Info("Starting ping manager")

	for {
		select {
		case <-ctx.Done():
			p.log.Info("Stopping ping manager")

			return
		case <-p.done:
			return
		case <-p.ticker.C:
			p.sendPings()
		}
	}
}

func (p *PingManager) Stop() {
	close(p.done)
}

func (p *PingManager) sendPings() {
	// Send primus ping to all connected clients
	timestamp := time.Now().UnixMilli()
	pingMsg := FormatPrimusPing(timestamp)

	p.manager.BroadcastMessage(pingMsg)

	p.log.WithField("timestamp", timestamp).Debug("Sent primus ping")
}
