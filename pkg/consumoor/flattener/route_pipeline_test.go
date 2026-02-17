package flattener

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	libp2p "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRoutePipelineBuildValidation(t *testing.T) {
	t.Run("from to builder", func(t *testing.T) {
		require.NotPanics(t, func() {
			From(xatu.Event_LIBP2P_TRACE_CONNECTED).
				To(TableName("libp2p_connected")).
				Apply(AddCommonMetadataFields).
				Build()
		})
	})

	t.Run("allows duplicate steps", func(t *testing.T) {
		require.NotPanics(t, func() {
			RouteTo(TableName("libp2p_connected"), xatu.Event_LIBP2P_TRACE_CONNECTED).
				Apply(AddCommonMetadataFields).
				Apply(AddCommonMetadataFields).
				Build()
		})
	})
}

func TestRoutePipelineStepSelection(t *testing.T) {
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

	t.Run("metadata step only", func(t *testing.T) {
		route := RouteTo(TableName("libp2p_connected"), xatu.Event_LIBP2P_TRACE_CONNECTED).
			Apply(AddCommonMetadataFields).
			Build()

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "peer-client", rows[0]["meta_client_name"])
		assert.NotContains(t, rows[0], "remote_peer")
	})

	t.Run("event step only", func(t *testing.T) {
		route := RouteTo(TableName("libp2p_connected"), xatu.Event_LIBP2P_TRACE_CONNECTED).
			Apply(FlattenEventDataFields).
			Build()

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "16Uiu2peer", rows[0]["remote_peer"])
		assert.NotContains(t, rows[0], "meta_client_name")
	})

	t.Run("route aliases are explicit", func(t *testing.T) {
		route := RouteTo(TableName("libp2p_connected"), xatu.Event_LIBP2P_TRACE_CONNECTED).
			Apply(FlattenEventDataFields).
			Apply(ApplyExplicitAliases(map[string]string{"remote_peer": "peer_id"})).
			Build()

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "16Uiu2peer", rows[0]["peer_id"])
		assert.NotContains(t, rows[0], "remote_peer")
	})

	t.Run("if condition is readable", func(t *testing.T) {
		route := From(xatu.Event_LIBP2P_TRACE_CONNECTED).
			To(TableName("libp2p_connected")).
			Apply(FlattenEventDataFields).
			If(func(event *xatu.DecoratedEvent) bool {
				return event.GetMeta().GetClient().GetName() == "peer-client"
			}).
			Build()

		assert.True(t, route.ShouldProcess(event))

		otherMsg := proto.Clone(event)
		other, ok := otherMsg.(*xatu.DecoratedEvent)
		require.True(t, ok)

		other.Meta = &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "other-client",
			},
		}
		assert.False(t, route.ShouldProcess(other))

		rows, err := route.Flatten(event, meta)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "16Uiu2peer", rows[0]["remote_peer"])
	})
}
