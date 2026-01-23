package extractors

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "beacon_blob",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
		ActivationFork: spec.DataVersionDeneb,
		Mode:           deriver.ProcessingModeEpoch,
		EpochProcessor: ProcessBeaconBlobs,
	})
}

// ProcessBeaconBlobs fetches and creates events for all blob sidecars in an epoch.
func ProcessBeaconBlobs(
	ctx context.Context,
	epoch phase0.Epoch,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	sp, err := beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	allEvents := []*xatu.DecoratedEvent{}

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		blobs, err := beacon.FetchBeaconBlockBlobs(ctx, xatuethv1.SlotAsString(slot))
		if err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) {
				switch apiErr.StatusCode {
				case 404:
					continue // No blobs for this slot
				case 503:
					return nil, errors.New("beacon node is syncing")
				}
			}

			return nil, errors.Wrapf(err, "failed to get beacon blob sidecars for slot %d", slot)
		}

		if blobs == nil {
			continue
		}

		for _, blob := range blobs {
			event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR)
			if err != nil {
				return nil, err
			}

			blockRoot, err := blob.SignedBlockHeader.Message.HashTreeRoot()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get block root")
			}

			event.Data = &xatu.DecoratedEvent_EthV1BeaconBlockBlobSidecar{
				EthV1BeaconBlockBlobSidecar: &xatuethv1.BlobSidecar{
					Slot:            &wrapperspb.UInt64Value{Value: uint64(blob.SignedBlockHeader.Message.Slot)},
					Blob:            fmt.Sprintf("0x%s", hex.EncodeToString(blob.Blob[:])),
					Index:           &wrapperspb.UInt64Value{Value: uint64(blob.Index)},
					BlockRoot:       fmt.Sprintf("0x%s", hex.EncodeToString(blockRoot[:])),
					BlockParentRoot: blob.SignedBlockHeader.Message.ParentRoot.String(),
					ProposerIndex:   &wrapperspb.UInt64Value{Value: uint64(blob.SignedBlockHeader.Message.ProposerIndex)},
					KzgCommitment:   blob.KZGCommitment.String(),
					KzgProof:        blob.KZGProof.String(),
				},
			}

			//nolint:gosec // blob sizes are bounded
			event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconBlobSidecar{
				EthV1BeaconBlobSidecar: &xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData{
					DataSize:      &wrapperspb.UInt64Value{Value: uint64(len(blob.Blob))},
					DataEmptySize: &wrapperspb.UInt64Value{Value: uint64(cldata.CountConsecutiveEmptyBytes(blob.Blob[:], 4))},
					VersionedHash: cldata.ConvertKzgCommitmentToVersionedHash(blob.KZGCommitment[:]).String(),
					Slot:          builder.BuildSlotV2(uint64(blob.SignedBlockHeader.Message.Slot)),
					Epoch:         builder.BuildEpochV2FromSlot(uint64(blob.SignedBlockHeader.Message.Slot)),
				},
			}

			allEvents = append(allEvents, event)
		}
	}

	return allEvents, nil
}
