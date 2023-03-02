package ethereum

import (
	"context"
	"errors"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/networks"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	Network *networks.Network

	Genesis *v1.Genesis
	Spec    *state.Spec

	wallclock *ethwallclock.EthereumBeaconChain

	mu sync.Mutex
}

func NewMetadataService(log logrus.FieldLogger, sbeacon beacon.Node) MetadataService {
	return MetadataService{
		beacon:  sbeacon,
		log:     log.WithField("module", "sentry/ethereum/metadata"),
		Network: &networks.Network{Name: networks.NetworkNameNone},
		mu:      sync.Mutex{},
	}
}

func (m *MetadataService) Start(ctx context.Context) error {
	m.beacon.OnReady(ctx, func(ctx context.Context, event *beacon.ReadyEvent) error {
		m.log.Info("Beacon node is ready")

		operation := func() error {
			return m.RefreshAll(ctx)
		}

		if err := backoff.Retry(operation, backoff.NewExponentialBackOff()); err != nil {
			m.log.WithError(err).Error("Failed to refresh metadata")

			return err
		}

		return nil
	})

	s := gocron.NewScheduler(time.Local)

	if _, err := s.Every("5m").Do(func() {
		_ = m.RefreshAll(ctx)
	}); err != nil {
		return err
	}

	s.StartAsync()

	return nil
}

func (m *MetadataService) Ready() error {
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

	return nil
}

func (m *MetadataService) RefreshAll(ctx context.Context) error {
	if err := m.fetchSpec(ctx); err != nil {
		m.log.WithError(err).Error("Failed to fetch spec for refresh")
	}

	if err := m.fetchGenesis(ctx); err != nil {
		m.log.WithError(err).Error("Failed to fetch genesis for refresh")
	}

	if m.Genesis != nil && m.Spec != nil {
		m.mu.Lock()

		if m.wallclock != nil {
			m.wallclock.Stop()
			m.wallclock = nil
		}

		m.wallclock = ethwallclock.NewEthereumBeaconChain(m.Genesis.GenesisTime, m.Spec.SecondsPerSlot.AsDuration(), uint64(m.Spec.SlotsPerEpoch))

		m.mu.Unlock()
	}

	if err := m.DeriveNetwork(ctx); err != nil {
		m.log.WithError(err).Error("Failed to derive network name for refresh")
	}

	return nil
}

func (m *MetadataService) Wallclock() *ethwallclock.EthereumBeaconChain {
	return m.wallclock
}

func (m *MetadataService) DeriveNetwork(ctx context.Context) error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	network := networks.DeriveFromGenesisRoot(xatuethv1.RootAsString(m.Genesis.GenesisValidatorsRoot))

	if network.Name != m.Network.Name {
		m.log.WithFields(logrus.Fields{
			"name": network.Name,
			"id":   network.ID,
		}).Info("Detected ethereum network")
	}

	m.Network = network

	return nil
}

func (m *MetadataService) fetchSpec(ctx context.Context) error {
	spec, err := m.beacon.Spec()
	if err != nil {
		return err
	}

	m.Spec = spec

	return nil
}

func (m *MetadataService) fetchGenesis(ctx context.Context) error {
	genesis, err := m.beacon.Genesis()
	if err != nil {
		return err
	}

	m.Genesis = genesis

	return nil
}

func (m *MetadataService) NodeVersion(ctx context.Context) string {
	version, _ := m.beacon.NodeVersion()

	return version
}

func (m *MetadataService) Client(ctx context.Context) string {
	return string(ClientFromString(m.NodeVersion(ctx)))
}
