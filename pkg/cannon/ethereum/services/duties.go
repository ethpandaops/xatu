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
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
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
			ttlcache.
				WithTTL[phase0.Epoch, []*v1.BeaconCommittee](90*time.Minute),
			ttlcache.
				WithCapacity[phase0.Epoch, []*v1.BeaconCommittee](256),
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

func (m *DutiesService) Ready(ctx context.Context) error {
	return nil
}

func (m *DutiesService) fireOnBeaconCommitteeSubscriptions(epoch phase0.Epoch, committees []*v1.BeaconCommittee) {
	for _, fn := range m.onBeaconCommitteeSubscriptions {
		if err := fn(epoch, committees); err != nil {
			m.log.WithError(err).Error("Failed to fire on beacon committee subscription")
		}
	}
}

func (m *DutiesService) FetchBeaconCommittee(ctx context.Context, stateID string, epoch phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	if duties := m.beaconCommittees.Get(epoch); duties != nil {
		return duties.Value(), nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.WithField("epoch", epoch).Debug("Fetching beacon committee")

	committees, err := m.beacon.FetchBeaconCommittees(ctx, xatuethv1.StateIDFinalized, &epoch)
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch beacon committees")

		return nil, err
	}

	m.beaconCommittees.Set(epoch, committees, time.Minute*90)

	m.fireOnBeaconCommitteeSubscriptions(epoch, committees)

	return committees, nil
}

func (m *DutiesService) GetValidatorIndex(epoch phase0.Epoch, slot phase0.Slot, committeeIndex phase0.CommitteeIndex, position uint64) (phase0.ValidatorIndex, error) {
	if _, err := m.FetchBeaconCommittee(context.Background(), "head", epoch); err != nil {
		return 0, fmt.Errorf("error fetching beacon committee for epoch %d: %w", epoch, err)
	}

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

	_, err := m.FetchBeaconCommittee(ctx, "head", phase0.Epoch(epoch.Number()))
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
