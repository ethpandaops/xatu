package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconSyntheticBuilderPendingPaymentSettlementEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_SYNTHETIC_BUILDER_PENDING_PAYMENT_SETTLEMENT,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconSyntheticBuilderPendingPaymentSettlementTableName,
		beaconSyntheticBuilderPendingPaymentSettlementEventNames,
		func() route.ColumnarBatch { return newbeaconSyntheticBuilderPendingPaymentSettlementBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconSyntheticBuilderPendingPaymentSettlementBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetBeaconSyntheticBuilderPendingPaymentSettlement() == nil {
		return fmt.Errorf(
			"nil beacon_synthetic_builder_pending_payment_settlement payload: %w",
			route.ErrInvalidEvent,
		)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconSyntheticBuilderPendingPaymentSettlementBatch) appendRuntime(
	event *xatu.DecoratedEvent,
) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconSyntheticBuilderPendingPaymentSettlementBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetBeaconSyntheticBuilderPendingPaymentSettlement()

	if v := payload.GetBuilderIndex(); v != nil {
		b.BuilderIndex.Append(v.GetValue())
	} else {
		b.BuilderIndex.Append(0)
	}

	b.FeeRecipient.Append([]byte(payload.GetFeeRecipient()))

	if v := payload.GetAmount(); v != nil {
		b.Amount.Append(v.GetValue())
	} else {
		b.Amount.Append(0)
	}

	if v := payload.GetWeight(); v != nil {
		b.Weight.Append(v.GetValue())
	} else {
		b.Weight.Append(0)
	}

	if v := payload.GetQuorum(); v != nil {
		b.Quorum.Append(v.GetValue())
	} else {
		b.Quorum.Append(0)
	}

	b.Outcome.Append(builderPendingPaymentOutcomeEnumName(payload.GetOutcome()))
}

func (b *beaconSyntheticBuilderPendingPaymentSettlementBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	additional := event.GetMeta().GetClient().GetBeaconSyntheticBuilderPendingPaymentSettlement()
	if additional == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch values fit uint32
		} else {
			b.Epoch.Append(0)
		}

		if sdt := epoch.GetStartDateTime(); sdt != nil {
			b.EpochStartDateTime.Append(sdt.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}
}

// builderPendingPaymentOutcomeEnumName returns the human-readable name for the
// outcome enum used in the ClickHouse table (matches the migration's
// LowCardinality(String)).
func builderPendingPaymentOutcomeEnumName(o ethv1.BuilderPendingPaymentOutcome) string {
	switch o {
	case ethv1.BuilderPendingPaymentOutcome_BUILDER_PENDING_PAYMENT_OUTCOME_SETTLED:
		return "SETTLED"
	case ethv1.BuilderPendingPaymentOutcome_BUILDER_PENDING_PAYMENT_OUTCOME_DROPPED:
		return "DROPPED"
	default:
		return "UNKNOWN"
	}
}
