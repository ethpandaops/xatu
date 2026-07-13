package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestCreateNodeRecordEvent(t *testing.T) {
	p := &Peer{
		handlers: &handler.Peer{
			CreateNewClientMeta: func(_ context.Context) (*xatu.ClientMeta, error) {
				return &xatu.ClientMeta{Name: "test-mimicry"}, nil
			},
		},
		network: &networks.Network{Name: networks.NetworkNameMainnet, ID: 1},
		forkID:  &xatu.ForkID{Hash: "0xf0afd0e3", Next: "0"},
	}

	status := &xatu.ExecutionNodeStatus{
		NodeRecord:      "enr:-test",
		Name:            "Geth/v1.16.0-stable/linux-amd64/go1.24.0",
		NetworkId:       1,
		ProtocolVersion: 5,
		Capabilities: []*xatu.ExecutionNodeStatus_Capability{
			{Name: "eth", Version: 68},
			{Name: "snap", Version: 1},
		},
		Head:    []byte{0xab, 0xcd},
		Genesis: []byte{0x12, 0x34},
		ForkId: &xatu.ExecutionNodeStatus_ForkID{
			Hash: []byte{0xf0, 0xaf, 0xd0, 0xe3},
			Next: 0,
		},
	}

	event, err := p.createNodeRecordEvent(context.Background(), status)
	require.NoError(t, err)

	require.Equal(t, xatu.Event_NODE_RECORD_EXECUTION, event.GetEvent().GetName())
	require.NotEmpty(t, event.GetEvent().GetId())
	require.Equal(t, "test-mimicry", event.GetMeta().GetClient().GetName())
	require.Equal(t, "mainnet", event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName())

	data := event.GetNodeRecordExecution()
	require.NotNil(t, data)
	require.Equal(t, "enr:-test", data.GetEnr().GetValue())
	require.Equal(t, "Geth/v1.16.0-stable/linux-amd64/go1.24.0", data.GetName().GetValue())
	require.Equal(t, "eth/68,snap/1", data.GetCapabilities().GetValue())
	require.Equal(t, "5", data.GetProtocolVersion().GetValue())
	require.Equal(t, "0xabcd", data.GetHead().GetValue())
	require.Equal(t, "0x1234", data.GetGenesis().GetValue())
	require.Equal(t, "0xf0afd0e3", data.GetForkIdHash().GetValue())
	require.Equal(t, "0", data.GetForkIdNext().GetValue())
}
