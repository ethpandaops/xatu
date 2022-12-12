package ethereum

import (
	"context"
	"errors"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/ethwallclock"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum/networks"
	"github.com/go-co-op/gocron"
	"github.com/samcm/beacon"
	"github.com/samcm/beacon/state"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	NetworkName networks.NetworkName

	Genesis *v1.Genesis
	Spec    *state.Spec

	wallclock *ethwallclock.EthereumBeaconChain
}

func NewMetadataService(log logrus.FieldLogger, sbeacon beacon.Node) MetadataService {
	return MetadataService{
		beacon:      sbeacon,
		log:         log.WithField("module", "sentry/ethereum/metadata"),
		NetworkName: networks.NetworkNameNone,
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

	if m.NetworkName == networks.NetworkNameNone {
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
		m.wallclock = ethwallclock.NewEthereumBeaconChain(m.Genesis.GenesisTime, m.Spec.SecondsPerSlot.AsDuration(), uint64(m.Spec.SlotsPerEpoch))
	}

	if err := m.DeriveNetworkName(ctx); err != nil {
		m.log.WithError(err).Error("Failed to derive network name for refresh")
	}

	return nil
}

func (m *MetadataService) Wallclock() *ethwallclock.EthereumBeaconChain {
	return m.wallclock
}

func (m *MetadataService) DeriveNetworkName(ctx context.Context) error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	network := networks.DeriveNetworkName(xatuethv1.RootAsString(m.Genesis.GenesisValidatorsRoot))

	if network != m.NetworkName {
		m.log.WithField("network", network).Info("Detected ethereum network")
	}

	m.NetworkName = network

	return nil
}

func (m *MetadataService) fetchSpec(ctx context.Context) error {
	spec, err := m.beacon.GetSpec(ctx)
	if err != nil {
		return err
	}

	m.Spec = spec

	return nil
}

func (m *MetadataService) fetchGenesis(ctx context.Context) error {
	genesis, err := m.beacon.GetGenesis(ctx)
	if err != nil {
		return err
	}

	m.Genesis = genesis

	return nil
}

func (m *MetadataService) NodeVersion(ctx context.Context) string {
	version, _ := m.beacon.GetNodeVersion(ctx)

	return version
}

func (m *MetadataService) Client(ctx context.Context) string {
	return string(ClientFromString(m.NodeVersion(ctx)))
}
