package flattener

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	libp2p "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRoutePipelineBuildValidation(t *testing.T) {
	t.Run("rejects duplicate stages", func(t *testing.T) {
		require.Panics(t, func() {
			RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
				CommonMetadata().
				CommonMetadata().
				Build()
		})
	})

	t.Run("rejects out-of-order stages", func(t *testing.T) {
		require.Panics(t, func() {
			RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
				EventData().
				CommonMetadata().
				Build()
		})
	})

	t.Run("rejects aliases without route alias stage", func(t *testing.T) {
		require.Panics(t, func() {
			RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
				EventData().
				Aliases(map[string]string{"remote_peer": "peer_id"}).
				Build()
		})
	})
}

func TestRoutePipelineStageSelection(t *testing.T) {
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_CONNECTED,
			Id:       "route-pipeline-test",
			DateTime: timestamppb.Now(),
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "peer-client",
			},
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
			Libp2PTraceConnected: &libp2p.Connected{
				RemotePeer: wrapperspb.String("16Uiu2peer"),
			},
		},
	}

	meta := metadata.Extract(event)

	t.Run("metadata stage only", func(t *testing.T) {
		route := RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
			CommonMetadata().
			Build()

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "peer-client", rows[0]["meta_client_name"])
		assert.NotContains(t, rows[0], "remote_peer")
	})

	t.Run("event stage only", func(t *testing.T) {
		route := RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
			EventData().
			Build()

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "16Uiu2peer", rows[0]["remote_peer"])
		assert.NotContains(t, rows[0], "meta_client_name")
	})

	t.Run("route aliases are explicit", func(t *testing.T) {
		route := RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
			EventData().
			RouteAliases().
			Aliases(map[string]string{"remote_peer": "peer_id"}).
			Build()

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "16Uiu2peer", rows[0]["peer_id"])
		assert.NotContains(t, rows[0], "remote_peer")
	})
}
