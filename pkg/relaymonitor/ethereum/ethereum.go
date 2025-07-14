package ethereum

import (
	"context"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/ethwallclock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BeaconNode struct {
	log       logrus.FieldLogger
	config    *Config
	node      beacon.Node
	wallclock *ethwallclock.EthereumBeaconChain
}

func NewBeaconNode(name string, log logrus.FieldLogger, config *Config) (*BeaconNode, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	opts := *beacon.
		DefaultOptions().
		DisableEmptySlotDetection().
		DisablePrometheusMetrics()

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	opts.BeaconSubscription.Enabled = false

	node := beacon.NewNode(log, &beacon.Config{
		Name:    name,
		Addr:    config.BeaconNodeURL,
		Headers: config.BeaconNodeHeaders,
	}, "xatu_relaymonitor", opts)

	return &BeaconNode{
		log:    log.WithField("component", "relaymonitor/ethereum/beacon"),
		config: config,
		node:   node,
	}, nil
}

func (b *BeaconNode) Wallclock() *ethwallclock.EthereumBeaconChain {
	return b.wallclock
}

func (b *BeaconNode) Start(ctx context.Context) error {
	// Start a 5min deadline timer to wait for the beacon node to be ready
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ready := make(chan struct{})

	b.node.OnReady(ctx, func(ctx context.Context, event *beacon.ReadyEvent) error {
		close(ready)

		return nil
	})

	b.node.StartAsync(ctx)

	select {
	case <-ready:
		// Beacon node is ready
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "beacon node failed to start within timeout")
	}

	spec, err := b.node.Spec()
	if err != nil {
		return errors.Wrap(err, "failed to get spec")
	}

	genesis, err := b.node.FetchGenesis(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch genesis")
	}

	// Create a new EthereumBeaconChain with the fetched genesis time and network config
	b.wallclock = ethwallclock.NewEthereumBeaconChain(
		genesis.GenesisTime,
		time.Duration(spec.SecondsPerSlot),
		uint64(spec.SlotsPerEpoch),
	)

	slot, epoch, err := b.wallclock.Now()
	if err != nil {
		return errors.Wrap(err, "failed to get current slot")
	}

	b.log.WithFields(logrus.Fields{
		"current_slot":  slot.Number(),
		"current_epoch": epoch.Number(),
	}).Info("Beacon chain wallclock initialized")

	return nil
}

func (b *BeaconNode) GetActiveValidators(ctx context.Context) (map[phase0.ValidatorIndex]*apiv1.Validator, error) {
	provider, isProvider := b.node.Service().(eth2client.ValidatorsProvider)
	if !isProvider {
		return nil, errors.New("node service is not a validators provider")
	}

	resp, err := provider.Validators(ctx, &api.ValidatorsOpts{
		ValidatorStates: []apiv1.ValidatorState{apiv1.ValidatorStateActiveOngoing},
		State:           "head",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active validators")
	}

	validators := make(map[phase0.ValidatorIndex]*apiv1.Validator, 0)
	for _, v := range resp.Data {
		validators[v.Index] = v
	}

	return validators, nil
}
