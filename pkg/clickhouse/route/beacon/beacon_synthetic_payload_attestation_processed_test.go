package beacon

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TestSnapshot_beacon_synthetic_payload_attestation_processed verifies the
// per-PTC-vote enrichment event flattens to ClickHouse columns. Asserts the
// processing-latency, validator-identity, and PTC vote payload bits round-trip
// correctly. EIP-7732 ePBS.
func TestSnapshot_beacon_synthetic_payload_attestation_processed(t *testing.T) {
	if len(beaconSyntheticPayloadAttestationProcessedEventNames) == 0 {
		t.Skip("no event names registered for beacon_synthetic_payload_attestation_processed")
	}

	const (
		colBeaconBlockRoot      = "beacon_block_root"
		colValidatorIndex       = "validator_index"
		colPayloadPresent       = "payload_present"
		colBlobDataAvailable    = "blob_data_available"
		colPeerID               = "peer_id"
		colProcessingDurationMs = "processing_duration_ms"
	)

	receivedAt := time.Unix(1_700_000_000, 0).UTC()
	processedAt := receivedAt.Add(17 * time.Millisecond)

	payload := &ethv1.PayloadAttestationProcessed{
		Slot:                 wrapperspb.UInt64(12345),
		BeaconBlockRoot:      blockRoot64A,
		ValidatorIndex:       wrapperspb.UInt64(987654),
		PayloadPresent:       true,
		BlobDataAvailable:    true,
		PeerId:               "16Uiu2HAmExamplePeerIDForSnapshotTest1234567890",
		ProcessingDurationMs: wrapperspb.UInt64(17),
		ReceivedAt:           timestamppb.New(receivedAt),
		ProcessedAt:          timestamppb.New(processedAt),
	}

	testfixture.AssertSnapshot(t, newbeaconSyntheticPayloadAttestationProcessedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconSyntheticPayloadAttestationProcessedEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_BeaconSyntheticPayloadAttestationProcessed{
			BeaconSyntheticPayloadAttestationProcessed: payload,
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colSlot:                       uint32(12345),
		colBeaconBlockRoot:            blockRoot64A,
		colValidatorIndex:             uint32(987654),
		colPayloadPresent:             true,
		colBlobDataAvailable:          true,
		colPeerID:                     "16Uiu2HAmExamplePeerIDForSnapshotTest1234567890",
		colProcessingDurationMs:       uint64(17),
	})
}
