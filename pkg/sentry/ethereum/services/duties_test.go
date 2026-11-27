package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/go-eth2-client/api"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDutiesNode serves committees, or fails them with committeeErr.
type fakeDutiesNode struct {
	beacon.Node

	committeeErr   error
	committeeCalls atomic.Int32
}

func (f *fakeDutiesNode) FetchBeaconCommittees(_ context.Context, _ string, _ *phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	f.committeeCalls.Add(1)

	if f.committeeErr != nil {
		return nil, f.committeeErr
	}

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

			// A syncing node 503s validator endpoints.
			var proposerCalls atomic.Int32

			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				proposerCalls.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer api.Close()
			metadata := &MetadataService{
				wallclock: ethwallclock.NewEthereumBeaconChain(time.Now().Add(-time.Hour), 12*time.Second, 32),
			}

			log := logrus.New()
			log.SetLevel(logrus.PanicLevel)

			duties := NewDutiesService(log, node, api.URL, nil, metadata, true, tt.beaconCommitteesEnabled)

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

			assert.Equal(t, int32(1), proposerCalls.Load())

			if tt.beaconCommitteesEnabled {
				assert.Positive(t, node.committeeCalls.Load())
			}
		})
	}
}

func TestFetchProposerDutiesKeepsReportedKeys(t *testing.T) {
	const (
		xmssKey = "0xFCF7A1B2C3D4E5F60718293A4B5C6D7E8F90A1B2C3D4E5F60718293A4B5C4C59"
		blsKey  = "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
	)

	var gotPath, gotHeader string

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("X-Test")

		_, _ = w.Write([]byte(`{"dependent_root":"0x00","execution_optimistic":false,"data":[` +
			`{"pubkey":"` + xmssKey + `","validator_index":"7","slot":"229"},` +
			`{"pubkey":"` + blsKey + `","validator_index":"9001","slot":"230"}]}`))
	}))
	defer api.Close()

	duties, err := beaconAPI{address: api.URL + "/", headers: map[string]string{"X-Test": "yes"}}.fetchProposerDuties(context.Background(), 7)
	require.NoError(t, err)

	assert.Equal(t, "/eth/v1/validator/duties/proposer/7", gotPath)
	assert.Equal(t, "yes", gotHeader)
	assert.Equal(t, []*ProposerDuty{
		{Slot: 229, ValidatorIndex: 7, PubKey: strings.ToLower(xmssKey)},
		{Slot: 230, ValidatorIndex: 9001, PubKey: blsKey},
	}, duties)
}

func TestDutiesServiceReadyWithoutCommitteesEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantReady bool
	}{
		{name: "committees endpoint missing", err: &api.Error{Method: "GET", StatusCode: http.StatusNotFound}, wantReady: true},
		{name: "committees endpoint failing", err: &api.Error{Method: "GET", StatusCode: http.StatusServiceUnavailable}, wantReady: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			node := &fakeDutiesNode{committeeErr: tt.err}
			metadata := &MetadataService{
				wallclock: ethwallclock.NewEthereumBeaconChain(time.Now().Add(-time.Hour), 12*time.Second, 32),
			}

			log := logrus.New()
			log.SetLevel(logrus.PanicLevel)

			duties := NewDutiesService(log, node, "http://127.0.0.1:1", nil, metadata, false, true)

			ready := make(chan struct{})

			duties.OnReady(ctx, func(context.Context) error {
				close(ready)

				return nil
			})

			require.NoError(t, duties.Start(ctx))

			select {
			case <-ready:
				require.True(t, tt.wantReady, "became ready while committees were failing")
				require.NoError(t, duties.Ready(ctx))
			case <-time.After(2 * time.Second):
				require.False(t, tt.wantReady, "did not become ready without a committees endpoint")
				require.Error(t, duties.Ready(ctx))
			}
		})
	}
}
