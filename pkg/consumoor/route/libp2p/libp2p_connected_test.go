package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_connected(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pConnectedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_CONNECTED,
			DateTime: testfixture.TS(),
			Id:       "conn-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
			Libp2PTraceConnected: &libp2ppb.Connected{
				RemotePeer:   wrapperspb.String(testPeerID),
				RemoteMaddrs: wrapperspb.String("/ip4/1.2.3.4/tcp/9000"),
				AgentVersion: wrapperspb.String("lighthouse/v4.5.6/linux"),
				Direction:    wrapperspb.String("inbound"),
			},
		},
	}, 1, map[string]any{
		"remote_protocol":             "ip4",
		"remote_ip":                   "1.2.3.4",
		"remote_transport_protocol":   "tcp",
		"remote_port":                 uint16(9000),
		"remote_agent_implementation": "lighthouse",
		"remote_agent_version":        "v4.5.6",
		"remote_agent_version_major":  "4",
		"remote_agent_version_minor":  "5",
		"remote_agent_version_patch":  "6",
		"remote_agent_platform":       "linux",
		"direction":                   "inbound",
		"remote_peer_id_unique_key":   expectedPeerIDKey,
	})
}
