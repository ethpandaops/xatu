package v2

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconBlockEventDeriver interface {
	Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error)
	Name() string
}
