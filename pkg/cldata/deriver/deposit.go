package deriver

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	DepositDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT
)

// DepositDeriverConfig holds the configuration for the DepositDeriver.
type DepositDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// DepositDeriver derives deposit events from the consensus layer.
// It processes epochs of blocks and emits decorated events for each deposit.
type DepositDeriver struct {
	log               logrus.FieldLogger
	cfg               *DepositDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewDepositDeriver creates a new DepositDeriver instance.
func NewDepositDeriver(
	log logrus.FieldLogger,
	config *DepositDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *DepositDeriver {
	return &DepositDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/deposit",
			"type":   DepositDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (d *DepositDeriver) CannonType() xatu.CannonType {
	return DepositDeriverName
}

func (d *DepositDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (d *DepositDeriver) Name() string {
	return DepositDeriverName.String()
}

func (d *DepositDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

func (d *DepositDeriver) Start(ctx context.Context) error {
	if !d.cfg.Enabled {
		d.log.Info("Deposit deriver disabled")

		return nil
	}

	d.log.Info("Deposit deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	d.run(ctx)

	return nil
}

func (d *DepositDeriver) Stop(ctx context.Context) error {
	return nil
}

func (d *DepositDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", d.Name()),
					trace.WithAttributes(
						attribute.String("network", d.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := d.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := d.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				d.lookAhead(ctx, position.LookAheadEpochs)

				// Process the epoch
				events, err := d.processEpoch(ctx, position.Epoch)
				if err != nil {
					d.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range d.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := d.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Location updated. Done.")

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					d.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				d.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (d *DepositDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"DepositDeriver.lookAhead",
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		d.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			d.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (d *DepositDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"DepositDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := d.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (d *DepositDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"DepositDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := d.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, d.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := make([]*xatu.DecoratedEvent, 0)

	deposits, err := d.getDeposits(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deposits for slot %d", slot)
	}

	for _, deposit := range deposits {
		event, err := d.createEvent(ctx, deposit, blockIdentifier)
		if err != nil {
			d.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for deposit %s", deposit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (d *DepositDeriver) getDeposits(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*xatuethv1.DepositV2, error) {
	deposits := make([]*xatuethv1.DepositV2, 0)

	dps, err := block.Deposits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain deposits")
	}

	for _, deposit := range dps {
		proof := make([]string, 0, len(deposit.Proof))
		for _, p := range deposit.Proof {
			proof = append(proof, fmt.Sprintf("0x%x", p))
		}

		deposits = append(deposits, &xatuethv1.DepositV2{
			Proof: proof,
			Data: &xatuethv1.DepositV2_Data{
				Pubkey:                deposit.Data.PublicKey.String(),
				WithdrawalCredentials: fmt.Sprintf("0x%x", deposit.Data.WithdrawalCredentials),
				Amount:                wrapperspb.UInt64(uint64(deposit.Data.Amount)),
				Signature:             deposit.Data.Signature.String(),
			},
		})
	}

	return deposits, nil
}

func (d *DepositDeriver) createEvent(
	ctx context.Context,
	deposit *xatuethv1.DepositV2,
	identifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := d.ctx.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: deposit,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockDeposit{
		EthV2BeaconBlockDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockDepositData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
