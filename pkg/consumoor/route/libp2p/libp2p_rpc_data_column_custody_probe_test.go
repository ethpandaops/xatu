package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_rpc_data_column_custody_probe(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pRpcDataColumnCustodyProbeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE,
			DateTime: testfixture.TS(),
			Id:       "probe-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceRpcDataColumnCustodyProbe{
				Libp2PTraceRpcDataColumnCustodyProbe: &xatu.ClientMeta_AdditionalLibP2PTraceRpcDataColumnCustodyProbeData{},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceRpcDataColumnCustodyProbe{
			Libp2PTraceRpcDataColumnCustodyProbe: &libp2ppb.DataColumnCustodyProbe{
				PeerId: wrapperspb.String(testPeerID),
			},
		},
	}, 1, map[string]any{
		"peer_id_unique_key": expectedPeerIDKey,
	})
}
