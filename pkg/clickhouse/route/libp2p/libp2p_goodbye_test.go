package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_goodbye(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
	)

	testfixture.AssertSnapshot(t, newlibp2pGoodbyeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOODBYE,
			DateTime: testfixture.TS(),
			Id:       "goodbye-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceGoodbye{
			Libp2PTraceGoodbye: &libp2ppb.Goodbye{
				RemotePeer:   wrapperspb.String(testPeerID),
				Code:         wrapperspb.UInt64(129),
				Reason:       wrapperspb.String("client has too many peers"),
				AgentVersion: wrapperspb.String("Lighthouse/v8.0.1-e3ee7fe/x86_64-linux"),
			},
		},
	}, 1, map[string]any{
		"remote_peer_id_unique_key":   route.SeaHashInt64(testPeerID + testNetwork),
		"code":                        uint64(129),
		"reason":                      "client has too many peers",
		"remote_agent_implementation": "lighthouse",
		"remote_agent_version":        "v8.0.1-e3ee7fe",
		"remote_agent_version_major":  "8",
		"remote_agent_version_minor":  "0",
		"remote_agent_version_patch":  "1",
		"remote_agent_platform":       "x86_64-linux",
		"meta_client_name":            "test-client",
	})
}
