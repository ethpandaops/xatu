package services

import (
	"context"
	"errors"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/ethereum"
	"github.com/ethpandaops/xatu/pkg/networks"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	Network *networks.Network

	Genesis *v1.Genesis
	Spec    *state.Spec

	wallclock *ethwallclock.EthereumBeaconChain

	onReadyCallbacks []func(context.Context) error

	overrideNetworkName string

	mu sync.Mutex
}

func NewMetadataService(log logrus.FieldLogger, sbeacon beacon.Node, overrideNetworkName string) MetadataService {
	return MetadataService{
		beacon:              sbeacon,
		log:                 log.WithField("module", "sentry/ethereum/metadata"),
		Network:             &networks.Network{Name: networks.NetworkNameNone},
		onReadyCallbacks:    []func(context.Context) error{},
		mu:                  sync.Mutex{},
		overrideNetworkName: overrideNetworkName,
	}
}

func (m *MetadataService) Start(ctx context.Context) error {
	go func() {
		operation := func() (string, error) {
			if err := m.RefreshAll(ctx); err != nil {
				return "", err
			}

			if err := m.Ready(ctx); err != nil {
				return "", err
			}

			return "", nil
		}

		retryOpts := []backoff.RetryOption{
			backoff.WithBackOff(backoff.NewExponentialBackOff()),
			backoff.WithNotify(func(err error, timer time.Duration) {
				m.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to refresh metadata")
			}),
		}

		if _, err := backoff.Retry(ctx, operation, retryOpts...); err != nil {
			m.log.WithError(err).Warn("Failed to refresh metadata")
		}

		for _, cb := range m.onReadyCallbacks {
			if err := cb(ctx); err != nil {
				m.log.WithError(err).Warn("Failed to execute onReady callback")
			}
		}
	}()

	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	if _, err := s.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func() {
				_ = m.RefreshAll(ctx)
			},
			ctx,
		),
	); err != nil {
		return err
	}

	s.Start()

	return nil
}

func (m *MetadataService) Name() Name {
	return "metadata"
}

func (m *MetadataService) Stop(ctx context.Context) error {
	return nil
}

func (m *MetadataService) OnReady(ctx context.Context, cb func(context.Context) error) {
	m.onReadyCallbacks = append(m.onReadyCallbacks, cb)
}

func (m *MetadataService) Ready(ctx context.Context) error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	if m.Spec == nil {
		return errors.New("spec is not available")
	}

	if m.NodeVersion(context.Background()) == "" {
		return errors.New("node version is not available")
	}

	if m.Network.Name == networks.NetworkNameNone {
		return errors.New("network name is not available")
	}

	if m.wallclock == nil {
		return errors.New("wallclock is not available")
	}

	return nil
}

func (m *MetadataService) RefreshAll(ctx context.Context) error {
	if err := m.fetchSpec(ctx); err != nil {
		m.log.WithError(err).Warn("Failed to fetch spec for refresh")
	}

	if err := m.fetchGenesis(ctx); err != nil {
		m.log.WithError(err).Warn("Failed to fetch genesis for refresh")
	}

	if m.Genesis != nil && m.Spec != nil && m.wallclock == nil {
		if newWallclock := ethwallclock.NewEthereumBeaconChain(m.Genesis.GenesisTime, m.Spec.SecondsPerSlot.AsDuration(), uint64(m.Spec.SlotsPerEpoch)); newWallclock != nil {
			m.mu.Lock()

			m.wallclock = newWallclock

			m.mu.Unlock()
		}
	}

	if err := m.DeriveNetwork(ctx); err != nil {
		m.log.WithError(err).Warn("Failed to derive network name for refresh")
	}

	return nil
}

func (m *MetadataService) Wallclock() *ethwallclock.EthereumBeaconChain {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.wallclock
}

func (m *MetadataService) DeriveNetwork(_ context.Context) error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	network := networks.DeriveFromGenesisRoot(xatuethv1.RootAsString(m.Genesis.GenesisValidatorsRoot))

	if m.overrideNetworkName != "" {
		network.Name = networks.NetworkName(m.overrideNetworkName)

		network.ID = m.Spec.DepositChainID
	}

	if network.Name != m.Network.Name {
		m.log.WithFields(logrus.Fields{
			"name": network.Name,
			"id":   network.ID,
		}).Info("Detected ethereum network")
	}

	m.Network = network

	return nil
}

func (m *MetadataService) fetchSpec(_ context.Context) error {
	spec, err := m.beacon.Spec()
	if err != nil {
		return err
	}

	m.Spec = spec

	return nil
}

func (m *MetadataService) fetchGenesis(_ context.Context) error {
	genesis, err := m.beacon.Genesis()
	if err != nil {
		return err
	}

	m.Genesis = genesis

	return nil
}

func (m *MetadataService) NodeVersion(_ context.Context) string {
	version, _ := m.beacon.NodeVersion()

	return version
}

func (m *MetadataService) Client(ctx context.Context) string {
	return string(ethereum.ClientFromString(m.NodeVersion(ctx)))
}
