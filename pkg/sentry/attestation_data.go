package sentry

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	v1 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v1"
)

func (s *Sentry) startAttestationDataSchedule(ctx context.Context) error {
	if !s.Config.AttestationData.Enabled {
		return nil
	}

	if s.Config.AttestationData.Interval.Enabled {
		logCtx := s.log.WithField("proccer", "interval").WithField("interval", s.Config.AttestationData.Interval.Every.String())

		if _, err := s.scheduler.Every(s.Config.AttestationData.Interval.Every.Duration).Do(func() {
			logCtx.Debug("Fetching validator attestation data")

			err := s.fetchDecoratedValidatorAttestationData(ctx)
			if err != nil {
				logCtx.WithError(err).Error("Failed to fetch validator attestation data")
			}
		}); err != nil {
			return err
		}
	}

	if s.Config.AttestationData.At.Enabled {
		for _, slotTime := range s.Config.AttestationData.At.SlotTimes {
			s.scheduleAttestationDataFetchingAtSlotTime(ctx, slotTime.Duration)
		}
	}

	return nil
}

func (s *Sentry) scheduleAttestationDataFetchingAtSlotTime(ctx context.Context, at time.Duration) {
	offset := at

	logCtx := s.log.
		WithField("proccer", "at_slot_time").
		WithField("slot_time", offset.String())

	logCtx.Debug("Scheduling validator attestation data fetching at slot time")

	s.beacon.Metadata().Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
		time.Sleep(offset)

		logCtx.WithField("slot", slot.Number()).Debug("Fetching validator attestation data")

		err := s.fetchDecoratedValidatorAttestationData(ctx)
		if err != nil {
			logCtx.WithField("slot_time", offset.String()).WithError(err).Error("Failed to fetch validator attestation data")
		}
	})
}

func (s *Sentry) fetchValidatorAttestationData(ctx context.Context) ([]*v1.ValidatorAttestationData, error) {
	startedAt := time.Now()

	snapshot := &v1.ValidatorAttestationDataSnapshot{
		RequestAt: startedAt,
	}

	slot := s.beacon.Metadata().Wallclock().Slots().Current()

	lastCommitteeIndex, err := s.beacon.Duties().GetLastCommitteeIndex(ctx, phase0.Slot(slot.Number()))
	if err != nil {
		return nil, err
	}

	targetCommitteeSize := uint64(*lastCommitteeIndex)

	attestationDataList := make([]*v1.ValidatorAttestationData, targetCommitteeSize)
	errChan := make(chan error, targetCommitteeSize)
	dataChan := make(chan *v1.ValidatorAttestationData, targetCommitteeSize)

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for i := uint64(0); i < targetCommitteeSize; i++ {
		go func(index phase0.CommitteeIndex) {
			data, err := s.beacon.Node().FetchAttestationData(ctx, phase0.Slot(slot.Number()), index)
			if err != nil {
				errChan <- err

				return
			}

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				errChan <- err

				return
			}

			snapshot.RequestDuration = time.Since(startedAt)
			snapshot.Event = data
			vad := v1.NewValidatorAttestationData(s.log, snapshot, s.beacon, meta)
			dataChan <- vad
		}(phase0.CommitteeIndex(i))
	}

	var errCount int

	for i := uint64(0); i < targetCommitteeSize; i++ {
		select {
		case err := <-errChan:
			errCount++
			if errCount == 1 { // only return the first error for simplicity
				return nil, err
			}
		case data := <-dataChan:
			attestationDataList[i] = data
		case <-timeoutCtx.Done():
			return nil, timeoutCtx.Err()
		}
	}

	return attestationDataList, nil
}

func (s *Sentry) fetchDecoratedValidatorAttestationData(ctx context.Context) error {
	dataList, err := s.fetchValidatorAttestationData(ctx)
	if err != nil {
		return err
	}

	for _, data := range dataList {
		ignore, err := data.ShouldIgnore(ctx)
		if err != nil {
			return err
		}

		if ignore {
			continue
		}

		decoratedEvent, err := data.Decorate(ctx)
		if err != nil {
			return err
		}

		err = s.handleNewDecoratedEvent(ctx, decoratedEvent)
		if err != nil {
			return err
		}
	}

	return nil
}
