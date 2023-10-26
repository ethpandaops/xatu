package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/ethwallclock"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
)

type DutiesService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	beaconCommittees *ttlcache.Cache[phase0.Epoch, []*v1.BeaconCommittee]

	onBeaconCommitteeSubscriptions []func(phase0.Epoch, []*v1.BeaconCommittee) error

	mu sync.Mutex

	metadata *MetadataService

	bootstrapped bool

	onReadyCallbacks []func(context.Context) error

	lastSyncState bool
}

func NewDutiesService(log logrus.FieldLogger, sbeacon beacon.Node, metadata *MetadataService) DutiesService {
	return DutiesService{
		beacon: sbeacon,
		log:    log.WithField("module", "sentry/ethereum/duties"),
		beaconCommittees: ttlcache.New(
			ttlcache.WithTTL[phase0.Epoch, []*v1.BeaconCommittee](60 * time.Minute),
		),
		mu: sync.Mutex{},

		onBeaconCommitteeSubscriptions: []func(phase0.Epoch, []*v1.BeaconCommittee) error{},
		onReadyCallbacks:               []func(context.Context) error{},

		metadata: metadata,

		bootstrapped: false,

		lastSyncState: false,
	}
}

func (m *DutiesService) Start(ctx context.Context) error {
	go func() {
		operation := func() error {
			if err := m.fetchRequiredEpochDuties(ctx, false); err != nil {
				return err
			}

			//nolint:errcheck // We don't care about the error here
			m.fetchNiceToHaveEpochDuties(ctx)

			if err := m.Ready(ctx); err != nil {
				return err
			}

			return nil
		}

		if err := backoff.Retry(operation, backoff.NewExponentialBackOff()); err != nil {
			m.log.WithError(err).Warn("Failed to fetch epoch duties")
		}

		for _, fn := range m.onReadyCallbacks {
			if err := fn(ctx); err != nil {
				m.log.WithError(err).Error("Failed to fire on ready callback")
			}
		}
	}()

	m.metadata.Wallclock().OnEpochChanged(func(epoch ethwallclock.Epoch) {
		if err := m.fetchRequiredEpochDuties(ctx, true); err != nil {
			m.log.WithError(err).Warn("Failed to fetch required epoch duties")
		}

		//nolint:errcheck // We don't care about the error here
		m.fetchNiceToHaveEpochDuties(ctx)
	})

	m.beacon.OnChainReOrg(ctx, func(ctx context.Context, ev *v1.ChainReorgEvent) error {
		if err := m.fetchRequiredEpochDuties(ctx, true); err != nil {
			m.log.WithError(err).Warn("Failed to fetch required epoch duties")
		}

		return nil
	})

	m.beacon.OnSyncStatus(ctx, func(ctx context.Context, ev *beacon.SyncStatusEvent) error {
		if ev.State.IsSyncing != m.lastSyncState {
			if err := m.fetchRequiredEpochDuties(ctx, true); err != nil {
				m.log.
					WithError(err).
					WithField("is_syncing", ev.State.IsSyncing).
					Warn("Failed to fetch required epoch duties after a sync status change")
			}
		}

		m.lastSyncState = ev.State.IsSyncing

		return nil
	})

	go m.beaconCommittees.Start()

	return nil
}

func (m *DutiesService) Stop(ctx context.Context) error {
	m.beaconCommittees.Stop()

	return nil
}

func (m *DutiesService) OnBeaconCommittee(fn func(phase0.Epoch, []*v1.BeaconCommittee) error) {
	m.onBeaconCommitteeSubscriptions = append(m.onBeaconCommitteeSubscriptions, fn)
}

func (m *DutiesService) OnReady(ctx context.Context, fn func(context.Context) error) {
	m.onReadyCallbacks = append(m.onReadyCallbacks, fn)
}

func (m *DutiesService) Name() Name {
	return "duties"
}

func (m *DutiesService) RequiredEpochDuties(ctx context.Context) []phase0.Epoch {
	now := m.metadata.Wallclock().Epochs().Current()

	epochNumber := now.Number()

	epochs := []phase0.Epoch{
		phase0.Epoch(epochNumber),
		phase0.Epoch(epochNumber + 1),
	}

	return epochs
}

func (m *DutiesService) NiceToHaveEpochDuties(ctx context.Context) []phase0.Epoch {
	now := m.metadata.Wallclock().Epochs().Current()

	epochNumber := now.Number()

	epochs := []phase0.Epoch{
		phase0.Epoch(epochNumber - 1),
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

func (m *DutiesService) Ready(ctx context.Context) error {
	for _, epoch := range m.RequiredEpochDuties(ctx) {
		if duties := m.beaconCommittees.Get(epoch); duties == nil {
			return fmt.Errorf("duties for epoch %d are not ready", epoch)
		}
	}

	return nil
}

func (m *DutiesService) fetchRequiredEpochDuties(ctx context.Context, overrideCache ...bool) error {
	if m.metadata.Wallclock() == nil {
		return fmt.Errorf("metadata service is not ready")
	}

	for _, epoch := range m.RequiredEpochDuties(ctx) {
		if duties := m.beaconCommittees.Get(epoch); duties == nil || len(overrideCache) != 0 && overrideCache[0] {
			if err := m.fetchBeaconCommittee(ctx, epoch, overrideCache...); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *DutiesService) fetchNiceToHaveEpochDuties(ctx context.Context) error {
	if m.metadata.Wallclock() == nil {
		return fmt.Errorf("metadata service is not ready")
	}

	for _, epoch := range m.NiceToHaveEpochDuties(ctx) {
		if duties := m.beaconCommittees.Get(epoch); duties == nil {
			if err := m.fetchBeaconCommittee(ctx, epoch); err != nil {
				m.log.WithError(err).Debugf("Failed to fetch beacon committee for epoch %d", epoch)
			}
		}
	}

	return nil
}

func (m *DutiesService) fireOnBeaconCommitteeSubscriptions(epoch phase0.Epoch, committees []*v1.BeaconCommittee) {
	for _, fn := range m.onBeaconCommitteeSubscriptions {
		if err := fn(epoch, committees); err != nil {
			m.log.WithError(err).Error("Failed to fire on beacon committee subscription")
		}
	}
}

func (m *DutiesService) fetchBeaconCommittee(ctx context.Context, epoch phase0.Epoch, overrideCache ...bool) error {
	if len(overrideCache) != 0 && !overrideCache[0] {
		if duties := m.beaconCommittees.Get(epoch); duties != nil {
			return nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	committees, err := m.beacon.FetchBeaconCommittees(ctx, "head", epoch)
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch beacon committees")

		return err
	}

	m.beaconCommittees.Set(epoch, committees, time.Minute*60)

	m.fireOnBeaconCommitteeSubscriptions(epoch, committees)

	return nil
}

func (m *DutiesService) GetAttestationDuties(epoch phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	duties := m.beaconCommittees.Get(epoch)
	if duties == nil {
		return nil, fmt.Errorf("duties for epoch %d are not known", epoch)
	}

	return duties.Value(), nil
}

func (m *DutiesService) GetValidatorIndex(epoch phase0.Epoch, slot phase0.Slot, committeeIndex phase0.CommitteeIndex, position uint64) (phase0.ValidatorIndex, error) {
	duties := m.beaconCommittees.Get(epoch)
	if duties == nil {
		return 0, fmt.Errorf("duties for epoch %d are not known", epoch)
	}

	for _, committee := range duties.Value() {
		if committee.Slot != slot || committee.Index != committeeIndex {
			continue
		}

		if position < uint64(len(committee.Validators)) {
			return committee.Validators[position], nil
		} else {
			return 0, fmt.Errorf("position %d is out of range for slot %d in epoch %d in committee %d", position, slot, epoch, committeeIndex)
		}
	}

	return 0, fmt.Errorf("validator index not found")
}

func (m *DutiesService) GetLastCommitteeIndex(ctx context.Context, slot phase0.Slot) (*phase0.CommitteeIndex, error) {
	epoch := m.metadata.Wallclock().Epochs().FromSlot(uint64(slot))

	err := m.fetchBeaconCommittee(ctx, phase0.Epoch(epoch.Number()))
	if err != nil {
		return nil, fmt.Errorf("error fetching beacon committee for epoch %d: %w", epoch.Number(), err)
	}

	committees := m.beaconCommittees.Get(phase0.Epoch(epoch.Number()))
	if committees == nil {
		return nil, fmt.Errorf("error getting beacon committees from cache for epoch %d: %w", epoch.Number(), err)
	}

	var maxIndex *phase0.CommitteeIndex

	for _, committee := range committees.Value() {
		if committee.Slot == slot && (maxIndex == nil || committee.Index > *maxIndex) {
			maxIndex = &committee.Index
		}
	}

	if maxIndex == nil {
		return nil, fmt.Errorf("no committees found for slot %d", slot)
	}

	return maxIndex, nil
}
