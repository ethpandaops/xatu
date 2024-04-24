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

	proposerDuties *ttlcache.Cache[phase0.Epoch, []*v1.ProposerDuty]

	onBeaconCommitteeSubscriptions []func(phase0.Epoch, []*v1.BeaconCommittee) error
	onProposerDutiesSubscriptions  []func(phase0.Epoch, []*v1.ProposerDuty) error

	committeeMu sync.Mutex
	proposerMu  sync.Mutex

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
		proposerDuties: ttlcache.New(
			ttlcache.WithTTL[phase0.Epoch, []*v1.ProposerDuty](60 * time.Minute),
		),
		committeeMu: sync.Mutex{},
		proposerMu:  sync.Mutex{},

		onBeaconCommitteeSubscriptions: []func(phase0.Epoch, []*v1.BeaconCommittee) error{},
		onProposerDutiesSubscriptions:  []func(phase0.Epoch, []*v1.ProposerDuty) error{},
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

			epoch := m.metadata.Wallclock().Epochs().Current()

			if err := m.fetchProposerDuties(ctx, phase0.Epoch(epoch.Number())); err != nil {
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

	// Fetch beacon committees
	m.metadata.Wallclock().OnEpochChanged(func(epoch ethwallclock.Epoch) {
		// Sleep for a bit to give the beacon node a chance to run its epoch transition.
		// We don't really care about nice-to-have duties so the sleep here is fine.
		// "Required" duties (aka the current epoch) will be refetched the moment that epoch
		// starts.
		time.Sleep(500 * time.Millisecond)

		if err := m.fetchRequiredEpochDuties(ctx, true); err != nil {
			m.log.WithError(err).Warn("Failed to fetch required epoch duties after an epoch change")
		}

		time.Sleep(15 * time.Second)

		//nolint:errcheck // We don't care about the error here
		m.fetchNiceToHaveEpochDuties(ctx)
	})

	// Fetch proposer duties
	m.metadata.Wallclock().OnEpochChanged(func(epoch ethwallclock.Epoch) {
		// Sleep for a bit to give the beacon node a chance to run its epoch transition.
		time.Sleep(100 * time.Millisecond)

		if err := m.fetchProposerDuties(ctx, phase0.Epoch(epoch.Number())); err != nil {
			m.log.WithError(err).Warn("Failed to fetch proposer duties")
		}
	})

	// Anticipate the next epoch and fetch the next epoch's beacon committees.
	m.metadata.Wallclock().OnEpochChanged(func(epoch ethwallclock.Epoch) {
		// Sleep until just before the start of the next epoch to fetch the next epoch's duties.
		time.Sleep(epoch.TimeWindow().EndsIn() - 2*time.Second)

		m.log.
			WithField("current_epoch", epoch.Number()).
			WithField("next_epoch", epoch.Number()+1).
			Debug("Fetching beacon committees for next epoch")

		if err := m.fetchBeaconCommittee(ctx, phase0.Epoch(epoch.Number()+1), true); err != nil {
			m.log.WithError(err).Warn("Failed to fetch required epoch duties in anticipation of an epoch change")
		}

		//nolint:errcheck // We don't care about the error here
		m.fetchNiceToHaveEpochDuties(ctx)
	})

	m.beacon.OnChainReOrg(ctx, func(ctx context.Context, ev *v1.ChainReorgEvent) error {
		m.log.Info("Chain reorg detected - refetching beacon committees")

		if err := m.fetchRequiredEpochDuties(ctx, true); err != nil {
			m.log.WithError(err).Warn("Failed to fetch required epoch duties")
		}

		return nil
	})

	m.beacon.OnSyncStatus(ctx, func(ctx context.Context, ev *beacon.SyncStatusEvent) error {
		if ev.State.IsSyncing != m.lastSyncState {
			m.log.WithFields(logrus.Fields{
				"is_syncing": ev.State.IsSyncing,
			}).Info("Sync status changed - refetching beacon committees")

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
	go m.proposerDuties.Start()

	return nil
}

func (m *DutiesService) Stop(ctx context.Context) error {
	m.beaconCommittees.Stop()

	return nil
}

func (m *DutiesService) OnBeaconCommittee(fn func(phase0.Epoch, []*v1.BeaconCommittee) error) {
	m.onBeaconCommitteeSubscriptions = append(m.onBeaconCommitteeSubscriptions, fn)
}

func (m *DutiesService) OnProposerDuties(fn func(phase0.Epoch, []*v1.ProposerDuty) error) {
	m.onProposerDutiesSubscriptions = append(m.onProposerDutiesSubscriptions, fn)
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

func (m *DutiesService) fireOnProposerDutiesSubscriptions(epoch phase0.Epoch, duties []*v1.ProposerDuty) {
	for _, fn := range m.onProposerDutiesSubscriptions {
		if err := fn(epoch, duties); err != nil {
			m.log.WithError(err).Error("Failed to fire on proposer duties subscription")
		}
	}
}

func (m *DutiesService) fetchBeaconCommittee(ctx context.Context, epoch phase0.Epoch, overrideCache ...bool) error {
	if len(overrideCache) != 0 && !overrideCache[0] {
		if duties := m.beaconCommittees.Get(epoch); duties != nil {
			return nil
		}
	}

	m.committeeMu.Lock()
	defer m.committeeMu.Unlock()

	wallclockEpoch := m.metadata.Wallclock().Epochs().Current()

	m.log.
		WithField("epoch", epoch).
		WithField("override_cache", overrideCache).
		WithField("wallclock_epoch", wallclockEpoch.Number()).
		Debug("Fetching beacon committee")

	committees, err := m.beacon.FetchBeaconCommittees(ctx, "head", epoch)
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch beacon committees")

		return err
	}

	m.beaconCommittees.Set(epoch, committees, time.Minute*60)

	m.fireOnBeaconCommitteeSubscriptions(epoch, committees)

	return nil
}

func (m *DutiesService) fetchProposerDuties(ctx context.Context, epoch phase0.Epoch) error {
	if duties := m.proposerDuties.Get(epoch); duties != nil {
		return nil
	}

	m.proposerMu.Lock()
	defer m.proposerMu.Unlock()

	wallclockEpoch := m.metadata.Wallclock().Epochs().Current()

	m.log.
		WithField("epoch", epoch).
		WithField("wallclock_epoch", wallclockEpoch.Number()).
		Debug("Fetching proposer duties")

	duties, err := m.beacon.FetchProposerDuties(ctx, epoch)
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch proposer duties")

		return err
	}

	m.proposerDuties.Set(epoch, duties, time.Minute*60)

	m.fireOnProposerDutiesSubscriptions(epoch, duties)

	return nil
}

func (m *DutiesService) GetAttestationDuties(epoch phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	duties := m.beaconCommittees.Get(epoch)
	if duties == nil {
		return nil, fmt.Errorf("duties for epoch %d are not known", epoch)
	}

	return duties.Value(), nil
}

func (m *DutiesService) GetProposerDuties(epoch phase0.Epoch) ([]*v1.ProposerDuty, error) {
	duties := m.proposerDuties.Get(epoch)
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
