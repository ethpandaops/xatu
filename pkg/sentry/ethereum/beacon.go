package ethereum

import (
	"context"
	"time"

	"github.com/samcm/beacon"
	"github.com/sirupsen/logrus"
)

type BeaconNode struct {
	config *Config
	log    logrus.FieldLogger

	beacon beacon.Node
}

func NewBeaconNode(name string, config *Config, log logrus.FieldLogger) (*BeaconNode, error) {
	opts := *beacon.DefaultOptions().EnableDefaultBeaconSubscription()
	opts.BeaconSubscription.Enabled = true

	opts.HealthCheck.Interval.Duration = time.Second * 5
	opts.HealthCheck.SuccessfulResponses = 1

	return &BeaconNode{
		config: config,
		log:    log,
		beacon: beacon.NewNode(log, &beacon.Config{
			Name: name,
			Addr: config.BeaconNodeAddress,
		}, "xatu", opts),
	}, nil
}

func (b *BeaconNode) Start(ctx context.Context) error {
	return b.beacon.Start(ctx)
}

func (b *BeaconNode) Node() beacon.Node {
	return b.beacon
}
