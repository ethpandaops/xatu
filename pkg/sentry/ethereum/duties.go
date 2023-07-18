package ethereum

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/go-co-op/gocron"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
)

type DutiesService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	attestationDuties *ttlcache.Cache[phase0.Epoch, []*v1.BeaconCommittee]

	mu sync.Mutex

	metadata *MetadataService

	bootstrapped bool
}

func NewDutiesService(log logrus.FieldLogger, sbeacon beacon.Node, metadata *MetadataService) DutiesService {
	return DutiesService{
		beacon: sbeacon,
		log:    log.WithField("module", "sentry/ethereum/duties"),
		attestationDuties: ttlcache.New(
			ttlcache.WithTTL[phase0.Epoch, []*v1.BeaconCommittee](60 * time.Minute),
		),
		mu: sync.Mutex{},

		metadata: metadata,

		bootstrapped: false,
	}
}

func (m *DutiesService) Start(ctx context.Context) error {
	m.beacon.OnReady(ctx, func(ctx context.Context, event *beacon.ReadyEvent) error {
		m.bootstrapped = true
		m.log.Info("Beacon node is ready")

		operation := func() error {
			return m.backFillEpochDuties(ctx)
		}

		if err := backoff.Retry(operation, backoff.NewExponentialBackOff()); err != nil {
			m.log.WithError(err).Warn("Failed to fetch epoch duties")

			return err
		}

		return nil
	})

	s := gocron.NewScheduler(time.Local)

	if _, err := s.Every("1m").Do(func() {
		_ = m.backFillEpochDuties(ctx)
	}); err != nil {
		return err
	}

	s.StartAsync()

	m.attestationDuties.Start()

	return nil
}

func (m *DutiesService) RequiredEpochDuties() []phase0.Epoch {
	now := m.beacon.Wallclock().Epochs().Current()

	epochNumber := now.Number()

	epochs := []phase0.Epoch{
		phase0.Epoch(epochNumber - 3),
		phase0.Epoch(epochNumber - 2),
		phase0.Epoch(epochNumber - 1),
		phase0.Epoch(epochNumber),
		phase0.Epoch(epochNumber + 1),
	}

	final := map[phase0.Epoch]struct{}{}

	// Deduplicate in case the current epoch is below epoch 3.
	for _, epoch := range epochs {
		final[epoch] = struct{}{}
	}

	epochs = make([]phase0.Epoch, 0, len(final))
	for epoch := range final {
		epochs = append(epochs, epoch)
	}

	return epochs
}

func (m *DutiesService) Ready() error {
	if !m.bootstrapped {
		return fmt.Errorf("beacon node is not ready")
	}

	if err := m.metadata.Ready(); err != nil {
		return fmt.Errorf("metadata service is not ready")
	}

	for _, epoch := range m.RequiredEpochDuties() {
		if duties := m.attestationDuties.Get(epoch); duties == nil {
			return fmt.Errorf("duties for epoch %d are not ready", epoch)
		}
	}

	return nil
}

func (m *DutiesService) backFillEpochDuties(ctx context.Context) error {
	if m.metadata.Wallclock() == nil {
		return fmt.Errorf("metadata service is not ready")
	}

	for _, epoch := range m.RequiredEpochDuties() {
		if duties := m.attestationDuties.Get(epoch); duties == nil {
			if err := m.fetchBeaconCommittee(ctx, epoch); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *DutiesService) fetchBeaconCommittee(ctx context.Context, epoch phase0.Epoch) error {
	if duties := m.attestationDuties.Get(epoch); duties != nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	committees, err := m.beacon.FetchBeaconCommittees(ctx, "head", epoch)
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch beacon committees")

		return err
	}

	m.attestationDuties.Set(epoch, committees, time.Minute*60)

	return nil
}

func (m *DutiesService) GetAttestationDuties(epoch phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	duties := m.attestationDuties.Get(epoch)
	if duties == nil {
		return nil, fmt.Errorf("duties for epoch %d are not known", epoch)
	}

	return duties.Value(), nil
}

func (m *DutiesService) GetValidatorIndex(epoch phase0.Epoch, slot phase0.Slot, committeeIndex phase0.CommitteeIndex, position uint64) (phase0.ValidatorIndex, error) {
	duties := m.attestationDuties.Get(epoch)
	if duties == nil {
		return 0, fmt.Errorf("duties for epoch %d are not known", epoch)
	}

	for _, committee := range duties.Value() {
		if committee.Slot != slot || committee.Index != committeeIndex {
			continue
		}

		return committee.Validators[position], nil
	}

	return 0, fmt.Errorf("validator index not found")
}
