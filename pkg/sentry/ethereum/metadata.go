package ethereum

import (
	"context"
	"errors"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/ethpandaops/xatu/pkg/wallclock"
	"github.com/samcm/beacon"
	"github.com/samcm/beacon/state"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	NetworkID uint64

	ConsensusImplementation string
	ConsensusVersion        string

	Genesis *v1.Genesis
	Spec    *state.Spec

	wallclock *wallclock.EthereumBeaconChain
}

func NewMetadataService(log logrus.FieldLogger, beacon beacon.Node) MetadataService {
	return MetadataService{
		beacon: beacon,
		log:    log.WithField("module", "sentry/ethereum/metadata"),
	}
}

func (m *MetadataService) Start(ctx context.Context) error {
	m.beacon.OnReady(ctx, func(ctx context.Context, event *beacon.ReadyEvent) error {
		m.log.WithField("event", event).Info("Beacon node is ready")

		return m.RefreshAll(ctx)
	})

	// TODO(sam.calder-mason): Add a ticker to refresh the metadata periodically.

	return nil
}

func (m *MetadataService) Ready() error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	if m.Spec == nil {
		return errors.New("spec is not available")
	}

	// TODO(sam.calder-mason): Check for version, network name/id.

	return nil
}

func (m *MetadataService) RefreshAll(ctx context.Context) error {
	if err := m.fetchSpec(ctx); err != nil {
		m.log.WithError(err).Error("Failed to fetch spec for refresh")

		return err
	}

	if err := m.fetchGenesis(ctx); err != nil {
		m.log.WithError(err).Error("Failed to fetch genesis for refresh")

		return err
	}

	m.wallclock = wallclock.NewEthereumBeaconChain(m.Genesis.GenesisTime, m.Spec.SecondsPerSlot.AsDuration(), uint64(m.Spec.SlotsPerEpoch))

	return nil
}

func (m *MetadataService) NetworkName() (string, error) {
	if m.Spec == nil {
		return "", errors.New("spec is not available")
	}

	return m.Spec.ConfigName, nil
}

func (m *MetadataService) Wallclock() *wallclock.EthereumBeaconChain {
	return m.wallclock
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
