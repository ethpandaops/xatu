package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/ethwallclock"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDutiesNode serves committees and fails proposer duties, like a syncing node that 503s validator endpoints.
type fakeDutiesNode struct {
	beacon.Node

	proposerCalls  atomic.Int32
	committeeCalls atomic.Int32
}

func (f *fakeDutiesNode) FetchProposerDuties(_ context.Context, _ phase0.Epoch) ([]*v1.ProposerDuty, error) {
	f.proposerCalls.Add(1)

	return nil, errors.New("GET failed with status 503")
}

func (f *fakeDutiesNode) FetchBeaconCommittees(_ context.Context, _ string, _ *phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	f.committeeCalls.Add(1)

	return []*v1.BeaconCommittee{{}}, nil
}

func (f *fakeDutiesNode) OnChainReOrg(_ context.Context, _ func(ctx context.Context, ev *v1.ChainReorgEvent) error) {
}

func (f *fakeDutiesNode) OnSyncStatus(_ context.Context, _ func(ctx context.Context, ev *beacon.SyncStatusEvent) error) {
}

func TestDutiesServiceReadyWhenProposerDutiesFail(t *testing.T) {
	tests := []struct {
		name                    string
		beaconCommitteesEnabled bool
	}{
		{name: "proposer duties only", beaconCommitteesEnabled: false},
		{name: "proposer duties and committees", beaconCommitteesEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			node := &fakeDutiesNode{}
			metadata := &MetadataService{
				wallclock: ethwallclock.NewEthereumBeaconChain(time.Now().Add(-time.Hour), 12*time.Second, 32),
			}

			log := logrus.New()
			log.SetLevel(logrus.PanicLevel)

			duties := NewDutiesService(log, node, metadata, true, tt.beaconCommitteesEnabled)

			ready := make(chan struct{})

			duties.OnReady(ctx, func(context.Context) error {
				close(ready)

				return nil
			})

			require.NoError(t, duties.Start(ctx))

			select {
			case <-ready:
			case <-time.After(5 * time.Second):
				t.Fatal("duties service did not become ready while proposer duties were failing")
			}

			assert.Equal(t, int32(1), node.proposerCalls.Load())

			if tt.beaconCommitteesEnabled {
				assert.Positive(t, node.committeeCalls.Load())
			}
		})
	}
}
