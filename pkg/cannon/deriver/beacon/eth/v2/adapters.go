package v2

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/cldata"
	cldataiterator "github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// BeaconClientAdapter wraps the Cannon's BeaconNode to implement cldata.BeaconClient.
type BeaconClientAdapter struct {
	beacon *ethereum.BeaconNode
}

// NewBeaconClientAdapter creates a new BeaconClientAdapter.
func NewBeaconClientAdapter(beaconNode *ethereum.BeaconNode) *BeaconClientAdapter {
	return &BeaconClientAdapter{beacon: beaconNode}
}

// GetBeaconBlock retrieves a beacon block by its identifier.
func (a *BeaconClientAdapter) GetBeaconBlock(ctx context.Context, identifier string) (*spec.VersionedSignedBeaconBlock, error) {
	return a.beacon.GetBeaconBlock(ctx, identifier)
}

// LazyLoadBeaconBlock queues a block for background preloading.
func (a *BeaconClientAdapter) LazyLoadBeaconBlock(identifier string) {
	a.beacon.LazyLoadBeaconBlock(identifier)
}

// Synced checks if the beacon node is synced.
func (a *BeaconClientAdapter) Synced(ctx context.Context) error {
	return a.beacon.Synced(ctx)
}

// Node returns the underlying beacon node.
func (a *BeaconClientAdapter) Node() beacon.Node {
	return a.beacon.Node()
}

// FetchBeaconBlockBlobs retrieves blob sidecars for a given block identifier.
func (a *BeaconClientAdapter) FetchBeaconBlockBlobs(ctx context.Context, identifier string) ([]*deneb.BlobSidecar, error) {
	return a.beacon.Node().FetchBeaconBlockBlobs(ctx, identifier)
}

// FetchBeaconCommittee retrieves the beacon committees for a given epoch.
func (a *BeaconClientAdapter) FetchBeaconCommittee(ctx context.Context, epoch phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	return a.beacon.Duties().FetchBeaconCommittee(ctx, epoch)
}

// GetValidatorIndex looks up a validator index from the committee for a given position.
func (a *BeaconClientAdapter) GetValidatorIndex(
	ctx context.Context,
	epoch phase0.Epoch,
	slot phase0.Slot,
	committeeIndex phase0.CommitteeIndex,
	position uint64,
) (phase0.ValidatorIndex, error) {
	return a.beacon.Duties().GetValidatorIndex(ctx, epoch, slot, committeeIndex, position)
}

// Verify BeaconClientAdapter implements cldata.BeaconClient.
var _ cldata.BeaconClient = (*BeaconClientAdapter)(nil)

// IteratorAdapter wraps the Cannon's BackfillingCheckpoint to implement cldata/iterator.Iterator.
type IteratorAdapter struct {
	iter *iterator.BackfillingCheckpoint
}

// NewIteratorAdapter creates a new IteratorAdapter.
func NewIteratorAdapter(iter *iterator.BackfillingCheckpoint) *IteratorAdapter {
	return &IteratorAdapter{iter: iter}
}

// Start initializes the iterator.
func (a *IteratorAdapter) Start(ctx context.Context, activationFork spec.DataVersion) error {
	return a.iter.Start(ctx, activationFork)
}

// Next returns the next position to process.
func (a *IteratorAdapter) Next(ctx context.Context) (*cldataiterator.Position, error) {
	resp, err := a.iter.Next(ctx)
	if err != nil {
		return nil, err
	}

	// Convert BackfillingCheckpoint response to shared Position
	direction := cldataiterator.DirectionForward
	if resp.Direction == iterator.BackfillingCheckpointDirectionBackfill {
		direction = cldataiterator.DirectionBackward
	}

	return &cldataiterator.Position{
		Epoch:           resp.Next,
		LookAheadEpochs: resp.LookAheads,
		Direction:       direction,
	}, nil
}

// UpdateLocation persists the current position.
func (a *IteratorAdapter) UpdateLocation(ctx context.Context, position *cldataiterator.Position) error {
	// Convert shared Direction to BackfillingCheckpoint direction
	direction := iterator.BackfillingCheckpointDirectionHead
	if position.Direction == cldataiterator.DirectionBackward {
		direction = iterator.BackfillingCheckpointDirectionBackfill
	}

	return a.iter.UpdateLocation(ctx, position.Epoch, direction)
}

// Verify IteratorAdapter implements cldataiterator.Iterator.
var _ cldataiterator.Iterator = (*IteratorAdapter)(nil)

// ContextProviderAdapter wraps Cannon's metadata creation to implement cldata.ContextProvider.
type ContextProviderAdapter struct {
	clientMeta     *xatu.ClientMeta
	networkName    string
	networkID      uint64
	wallclock      *ethwallclock.EthereumBeaconChain
	depositChainID uint64
}

// NewContextProviderAdapter creates a new ContextProviderAdapter.
func NewContextProviderAdapter(
	clientMeta *xatu.ClientMeta,
	networkName string,
	networkID uint64,
	wallclock *ethwallclock.EthereumBeaconChain,
	depositChainID uint64,
) *ContextProviderAdapter {
	return &ContextProviderAdapter{
		clientMeta:     clientMeta,
		networkName:    networkName,
		networkID:      networkID,
		wallclock:      wallclock,
		depositChainID: depositChainID,
	}
}

// CreateClientMeta returns the client metadata.
func (a *ContextProviderAdapter) CreateClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	return a.clientMeta, nil
}

// NetworkName returns the network name.
func (a *ContextProviderAdapter) NetworkName() string {
	return a.networkName
}

// NetworkID returns the network ID.
func (a *ContextProviderAdapter) NetworkID() uint64 {
	return a.networkID
}

// Wallclock returns the Ethereum wallclock.
func (a *ContextProviderAdapter) Wallclock() *ethwallclock.EthereumBeaconChain {
	return a.wallclock
}

// DepositChainID returns the execution layer chain ID.
func (a *ContextProviderAdapter) DepositChainID() uint64 {
	return a.depositChainID
}

// Verify ContextProviderAdapter implements cldata.ContextProvider.
var _ cldata.ContextProvider = (*ContextProviderAdapter)(nil)
