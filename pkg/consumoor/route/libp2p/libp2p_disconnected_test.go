package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_disconnected(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pDisconnectedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DISCONNECTED,
			DateTime: testfixture.TS(),
			Id:       "disc-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceDisconnected{
			Libp2PTraceDisconnected: &libp2ppb.Disconnected{
				RemotePeer:   wrapperspb.String(testPeerID),
				RemoteMaddrs: wrapperspb.String("/ip4/5.6.7.8/udp/30303"),
				Direction:    wrapperspb.String("outbound"),
			},
		},
	}, 1, map[string]any{
		"remote_protocol":           "ip4",
		"remote_ip":                 "5.6.7.8",
		"remote_transport_protocol": "udp",
		"remote_port":               uint16(30303),
		"direction":                 "outbound",
		"remote_peer_id_unique_key": expectedPeerIDKey,
	})
}
