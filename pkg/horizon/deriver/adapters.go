package deriver

import (
	"context"
	"runtime"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
)

// BeaconClientAdapter wraps the Horizon's BeaconNodePool to implement cldata.BeaconClient.
type BeaconClientAdapter struct {
	pool *ethereum.BeaconNodePool
}

// NewBeaconClientAdapter creates a new BeaconClientAdapter.
func NewBeaconClientAdapter(pool *ethereum.BeaconNodePool) *BeaconClientAdapter {
	return &BeaconClientAdapter{pool: pool}
}

// GetBeaconBlock retrieves a beacon block by its identifier.
func (a *BeaconClientAdapter) GetBeaconBlock(ctx context.Context, identifier string) (*spec.VersionedSignedBeaconBlock, error) {
	return a.pool.GetBeaconBlock(ctx, identifier)
}

// LazyLoadBeaconBlock queues a block for background preloading.
func (a *BeaconClientAdapter) LazyLoadBeaconBlock(identifier string) {
	a.pool.LazyLoadBeaconBlock(identifier)
}

// Synced checks if the beacon node pool has at least one synced node.
func (a *BeaconClientAdapter) Synced(ctx context.Context) error {
	return a.pool.Synced(ctx)
}

// Node returns the underlying beacon node (uses first healthy node).
func (a *BeaconClientAdapter) Node() beacon.Node {
	wrapper, err := a.pool.GetHealthyNode()
	if err != nil {
		return nil
	}

	return wrapper.Node()
}

// FetchBeaconBlockBlobs retrieves blob sidecars for a given block identifier.
func (a *BeaconClientAdapter) FetchBeaconBlockBlobs(ctx context.Context, identifier string) ([]*deneb.BlobSidecar, error) {
	wrapper, err := a.pool.GetHealthyNode()
	if err != nil {
		return nil, err
	}

	return wrapper.Node().FetchBeaconBlockBlobs(ctx, identifier)
}

// FetchBeaconCommittee retrieves the beacon committees for a given epoch.
func (a *BeaconClientAdapter) FetchBeaconCommittee(ctx context.Context, epoch phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	return a.pool.Duties().FetchBeaconCommittee(ctx, epoch)
}

// GetValidatorIndex looks up a validator index from the committee for a given position.
func (a *BeaconClientAdapter) GetValidatorIndex(
	ctx context.Context,
	epoch phase0.Epoch,
	slot phase0.Slot,
	committeeIndex phase0.CommitteeIndex,
	position uint64,
) (phase0.ValidatorIndex, error) {
	return a.pool.Duties().GetValidatorIndex(ctx, epoch, slot, committeeIndex, position)
}

// FetchProposerDuties retrieves the proposer duties for a given epoch.
func (a *BeaconClientAdapter) FetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*v1.ProposerDuty, error) {
	wrapper, err := a.pool.GetHealthyNode()
	if err != nil {
		return nil, err
	}

	return wrapper.Node().FetchProposerDuties(ctx, epoch)
}

// GetValidators retrieves validators for a given state identifier.
// Note: Horizon doesn't cache validators like Cannon does. This is a direct fetch.
func (a *BeaconClientAdapter) GetValidators(ctx context.Context, identifier string) (map[phase0.ValidatorIndex]*v1.Validator, error) {
	wrapper, err := a.pool.GetHealthyNode()
	if err != nil {
		return nil, err
	}

	// Pass nil for validatorIndices and pubkeys to fetch all validators.
	return wrapper.Node().FetchValidators(ctx, identifier, nil, nil)
}

// LazyLoadValidators is a no-op for Horizon (no validator caching).
func (a *BeaconClientAdapter) LazyLoadValidators(_ string) {
	// Horizon doesn't cache validators - blocks are already cached and validators
	// are fetched on-demand.
}

// DeleteValidatorsFromCache is a no-op for Horizon (no validator caching).
func (a *BeaconClientAdapter) DeleteValidatorsFromCache(_ string) {
	// Horizon doesn't cache validators.
}

// Verify BeaconClientAdapter implements cldata.BeaconClient.
var _ cldata.BeaconClient = (*BeaconClientAdapter)(nil)

// ContextProviderAdapter wraps Horizon's metadata creation to implement cldata.ContextProvider.
type ContextProviderAdapter struct {
	id             uuid.UUID
	name           string
	networkName    string
	networkID      uint64
	wallclock      *ethwallclock.EthereumBeaconChain
	depositChainID uint64
	labels         map[string]string
}

// NewContextProviderAdapter creates a new ContextProviderAdapter.
func NewContextProviderAdapter(
	id uuid.UUID,
	name string,
	networkName string,
	networkID uint64,
	wallclock *ethwallclock.EthereumBeaconChain,
	depositChainID uint64,
	labels map[string]string,
) *ContextProviderAdapter {
	return &ContextProviderAdapter{
		id:             id,
		name:           name,
		networkName:    networkName,
		networkID:      networkID,
		wallclock:      wallclock,
		depositChainID: depositChainID,
		labels:         labels,
	}
}

// CreateClientMeta creates the client metadata for events.
// Unlike Cannon which pre-builds metadata, Horizon creates it fresh for each call
// to ensure accurate timestamps.
func (a *ContextProviderAdapter) CreateClientMeta(_ context.Context) (*xatu.ClientMeta, error) {
	return &xatu.ClientMeta{
		Name:           a.name,
		Version:        xatu.Short(),
		Id:             a.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ModuleName:     xatu.ModuleName_HORIZON,
		ClockDrift:     0, // Horizon doesn't track clock drift currently
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Name: a.networkName,
				Id:   a.networkID,
			},
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{},
		},
		Labels: a.labels,
	}, nil
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
