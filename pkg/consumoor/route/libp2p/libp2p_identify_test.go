package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_identify(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pIdentifyBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_IDENTIFY,
			DateTime: testfixture.TS(),
			Id:       "ident-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceIdentify{
			Libp2PTraceIdentify: &libp2ppb.Identify{
				RemotePeer:      wrapperspb.String(testPeerID),
				Success:         wrapperspb.Bool(true),
				AgentVersion:    wrapperspb.String("lighthouse/v4.5.6/linux"),
				ProtocolVersion: wrapperspb.String("/eth2/beacon_chain/req/status/1/ssz_snappy"),
				Protocols:       []string{"/meshsub/1.1.0", "/eth2/beacon_chain/req/status/1"},
				ListenAddrs:     []string{"/ip4/0.0.0.0/tcp/9000"},
				ObservedAddr:    wrapperspb.String("/ip4/1.2.3.4/tcp/9000"),
				Transport:       wrapperspb.String("tcp"),
				Security:        wrapperspb.String("noise"),
				Muxer:           wrapperspb.String("mplex"),
				Direction:       wrapperspb.String("inbound"),
				RemoteMultiaddr: wrapperspb.String("/ip4/5.6.7.8/tcp/30303"),
			},
		},
	}, 1, map[string]any{
		"success":                     true,
		"remote_protocol":             "ip4",
		"remote_ip":                   "5.6.7.8",
		"remote_transport_protocol":   "tcp",
		"remote_port":                 uint16(30303),
		"remote_agent_implementation": "lighthouse",
		"remote_agent_version":        "v4.5.6",
		"remote_agent_version_major":  "4",
		"remote_agent_version_minor":  "5",
		"remote_agent_version_patch":  "6",
		"remote_agent_platform":       "linux",
		"protocol_version":            "/eth2/beacon_chain/req/status/1/ssz_snappy",
		"observed_addr":               "/ip4/1.2.3.4/tcp/9000",
		"transport":                   "tcp",
		"security":                    "noise",
		"muxer":                       "mplex",
		"direction":                   "inbound",
		"remote_multiaddr":            "/ip4/5.6.7.8/tcp/30303",
		"remote_peer_id_unique_key":   expectedPeerIDKey,
	})
}
