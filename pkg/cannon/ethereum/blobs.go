package ethereum

import (
	"context"

	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

// GetBlobSidecars returns the blob sidecars of a block. From Gloas the block
// body no longer carries KZG commitments, so beacon nodes cannot serve blob
// sidecars for it. Those sidecars are rebuilt from the blobs endpoint and the
// commitments in the block's bid, and carry no KZG or inclusion proof.
func (b *BeaconNode) GetBlobSidecars(ctx context.Context, block *spec.VersionedSignedBeaconBlock, blockID string) ([]*deneb.BlobSidecar, error) {
	if block.Version < spec.DataVersionGloas {
		return b.beacon.FetchBeaconBlockBlobs(ctx, blockID)
	}

	bid, err := block.SignedExecutionPayloadBid()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read execution payload bid")
	}

	commitments, err := bid.BlobKZGCommitments()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read blob kzg commitments from bid")
	}

	if len(commitments) == 0 {
		return nil, nil
	}

	// A withheld payload takes its blobs with it.
	envelope, err := b.GetExecutionPayloadEnvelope(ctx, blockID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get execution payload envelope")
	}

	if envelope == nil {
		return nil, nil
	}

	provider, isProvider := b.beacon.Service().(client.BlobsProvider)
	if !isProvider {
		return nil, errors.New("blobs provider not available")
	}

	resp, err := provider.Blobs(ctx, &api.BlobsOpts{Block: blockID})
	if err != nil {
		return nil, err
	}

	if resp == nil || len(resp.Data) != len(commitments) {
		got := 0
		if resp != nil {
			got = len(resp.Data)
		}

		return nil, errors.Errorf("beacon node returned %d blobs for %d commitments", got, len(commitments))
	}

	header, err := blockHeader(block)
	if err != nil {
		return nil, err
	}

	sidecars := make([]*deneb.BlobSidecar, len(commitments))
	for i, blob := range resp.Data {
		sidecars[i] = &deneb.BlobSidecar{
			Index:             deneb.BlobIndex(i),
			Blob:              *blob,
			KZGCommitment:     commitments[i],
			SignedBlockHeader: &phase0.SignedBeaconBlockHeader{Message: header},
		}
	}

	return sidecars, nil
}

func blockHeader(block *spec.VersionedSignedBeaconBlock) (*phase0.BeaconBlockHeader, error) {
	slot, err := block.Slot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read block slot")
	}

	proposerIndex, err := block.ProposerIndex()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read block proposer index")
	}

	parentRoot, err := block.ParentRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read block parent root")
	}

	stateRoot, err := block.StateRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read block state root")
	}

	bodyRoot, err := block.BodyRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read block body root")
	}

	return &phase0.BeaconBlockHeader{
		Slot:          slot,
		ProposerIndex: proposerIndex,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bodyRoot,
	}, nil
}
