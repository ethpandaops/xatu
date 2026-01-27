package ethstats_test

import (
	"encoding/json"
	"testing"

	"github.com/ethpandaops/xatu/pkg/ethstats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_ParseHello(t *testing.T) {
	input := `{"emit":["hello",{"id":"node-1","secret":"mysecret","info":{"name":"geth/v1.0","net":"mainnet"}}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "hello", cmd)

	var auth ethstats.AuthMsg

	err = json.Unmarshal(msg.Emit[1], &auth)
	require.NoError(t, err)
	assert.Equal(t, "node-1", auth.ID)
	assert.Equal(t, "mysecret", auth.Secret)
	assert.Equal(t, "geth/v1.0", auth.Info.Name)
	assert.Equal(t, "mainnet", auth.Info.Network)
}

func TestMessage_ParseBlock(t *testing.T) {
	input := `{"emit":["block",{"id":"node-1","block":{"number":"12345","hash":"0xabc","parentHash":"0xdef","timestamp":"1234567890","miner":"0x123","gasUsed":21000,"gasLimit":30000000}}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "block", cmd)

	var report ethstats.BlockReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	require.NotNil(t, report.Block)
	assert.Equal(t, json.Number("12345"), report.Block.Number)
	assert.Equal(t, "0xabc", report.Block.Hash)
	assert.Equal(t, "0xdef", report.Block.ParentHash)
	assert.Equal(t, uint64(21000), report.Block.GasUsed)
	assert.Equal(t, uint64(30000000), report.Block.GasLimit)
}

func TestMessage_ParseStats(t *testing.T) {
	input := `{"emit":["stats",{"id":"node-1","stats":{"active":true,"syncing":false,"peers":25,"gasPrice":1000000000,"uptime":99}}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "stats", cmd)

	var report ethstats.StatsReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	require.NotNil(t, report.Stats)
	assert.True(t, report.Stats.Active)
	assert.False(t, report.Stats.Syncing)
	assert.Equal(t, 25, report.Stats.Peers)
	assert.Equal(t, 1000000000, report.Stats.GasPrice)
	assert.Equal(t, 99, report.Stats.Uptime)
}

func TestMessage_ParsePending(t *testing.T) {
	input := `{"emit":["pending",{"id":"node-1","stats":{"pending":150}}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "pending", cmd)

	var report ethstats.PendingReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	require.NotNil(t, report.Stats)
	assert.Equal(t, 150, report.Stats.Pending)
}

func TestMessage_ParseLatency(t *testing.T) {
	input := `{"emit":["latency",{"id":"node-1","latency":"50"}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "latency", cmd)

	var report ethstats.LatencyReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	assert.Equal(t, "50", report.Latency)
}

func TestMessage_ParseNodePing(t *testing.T) {
	input := `{"emit":["node-ping",{"id":"node-1","clientTime":"1234567890123"}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "node-ping", cmd)

	var report ethstats.PingReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	assert.Equal(t, "1234567890123", report.ClientTime)
}

func TestMessage_ParseBlockNewPayload(t *testing.T) {
	input := `{"emit":["block_new_payload",{"id":"node-1","block":{"number":"12345","hash":"0xabc","processingTime":5000000}}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "block_new_payload", cmd)

	var report ethstats.NewPayloadReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	require.NotNil(t, report.Block)
	assert.Equal(t, json.Number("12345"), report.Block.Number)
	assert.Equal(t, "0xabc", report.Block.Hash)
	assert.Equal(t, uint64(5000000), report.Block.ProcessingTime)
}

func TestMessage_ParseHistory(t *testing.T) {
	input := `{"emit":["history",{"id":"node-1","history":[{"number":"100","hash":"0xaaa"},{"number":"101","hash":"0xbbb"}]}]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	require.Len(t, msg.Emit, 2)

	var cmd string

	err = json.Unmarshal(msg.Emit[0], &cmd)
	require.NoError(t, err)
	assert.Equal(t, "history", cmd)

	var report ethstats.HistoryReport

	err = json.Unmarshal(msg.Emit[1], &report)
	require.NoError(t, err)
	assert.Equal(t, "node-1", report.ID)
	require.Len(t, report.History, 2)
	assert.Equal(t, json.Number("100"), report.History[0].Number)
	assert.Equal(t, "0xaaa", report.History[0].Hash)
	assert.Equal(t, json.Number("101"), report.History[1].Number)
	assert.Equal(t, "0xbbb", report.History[1].Hash)
}

func TestMessage_InvalidJSON(t *testing.T) {
	input := `{invalid json`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.Error(t, err)
}

func TestMessage_EmptyEmitArray(t *testing.T) {
	input := `{"emit":[]}`

	var msg ethstats.Message

	err := json.Unmarshal([]byte(input), &msg)
	require.NoError(t, err)
	assert.Empty(t, msg.Emit)
}

func TestNodeInfo_Parse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ethstats.NodeInfo
	}{
		{
			name:  "Full node info",
			input: `{"name":"geth/v1.13.0","node":"enode://abc@127.0.0.1:30303","port":30303,"net":"mainnet","protocol":"eth/68","api":"No","os":"linux","os_v":"ubuntu","client":"geth","canUpdateHistory":true}`,
			expected: ethstats.NodeInfo{
				Name:     "geth/v1.13.0",
				Node:     "enode://abc@127.0.0.1:30303",
				Port:     30303,
				Network:  "mainnet",
				Protocol: "eth/68",
				API:      "No",
				Os:       "linux",
				OsVer:    "ubuntu",
				Client:   "geth",
				History:  true,
			},
		},
		{
			name:  "Minimal node info",
			input: `{"name":"node-1","net":"sepolia"}`,
			expected: ethstats.NodeInfo{
				Name:    "node-1",
				Network: "sepolia",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info ethstats.NodeInfo

			err := json.Unmarshal([]byte(tt.input), &info)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, info)
		})
	}
}

func TestBlockStats_ParseFull(t *testing.T) {
	input := `{"number":"12345678","hash":"0xabc123","parentHash":"0xdef456","timestamp":"1700000000","miner":"0x742d35cc6634c0532925a3b844bc9e7595f9d123","gasUsed":15000000,"gasLimit":30000000,"difficulty":"0","totalDifficulty":"58750003716598352816469","transactions":[{"hash":"0xtx1"},{"hash":"0xtx2"}],"transactionsRoot":"0xroot","stateRoot":"0xstate","uncles":[]}`

	var block ethstats.BlockStats

	err := json.Unmarshal([]byte(input), &block)
	require.NoError(t, err)

	assert.Equal(t, json.Number("12345678"), block.Number)
	assert.Equal(t, "0xabc123", block.Hash)
	assert.Equal(t, "0xdef456", block.ParentHash)
	assert.Equal(t, json.Number("1700000000"), block.Timestamp)
	assert.Equal(t, "0x742d35cc6634c0532925a3b844bc9e7595f9d123", block.Miner)
	assert.Equal(t, uint64(15000000), block.GasUsed)
	assert.Equal(t, uint64(30000000), block.GasLimit)
	assert.Equal(t, "0", block.Diff)
	assert.Equal(t, "58750003716598352816469", block.TotalDiff)
	require.Len(t, block.Txs, 2)
	assert.Equal(t, "0xtx1", block.Txs[0].Hash)
	assert.Equal(t, "0xtx2", block.Txs[1].Hash)
	assert.Equal(t, "0xroot", block.TxHash)
	assert.Equal(t, "0xstate", block.Root)
	assert.Empty(t, block.Uncles)
}

func TestBlockStats_ParseEmpty(t *testing.T) {
	input := `{"number":"100","hash":"0xhash","transactions":[],"uncles":[]}`

	var block ethstats.BlockStats

	err := json.Unmarshal([]byte(input), &block)
	require.NoError(t, err)

	assert.Equal(t, json.Number("100"), block.Number)
	assert.Equal(t, "0xhash", block.Hash)
	assert.Empty(t, block.Txs)
}

func TestNodeStats_Parse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ethstats.NodeStats
	}{
		{
			name:  "Active syncing node",
			input: `{"active":true,"syncing":true,"peers":50,"gasPrice":20000000000,"uptime":100}`,
			expected: ethstats.NodeStats{
				Active:   true,
				Syncing:  true,
				Peers:    50,
				GasPrice: 20000000000,
				Uptime:   100,
			},
		},
		{
			name:  "Inactive node",
			input: `{"active":false,"syncing":false,"peers":0,"gasPrice":0,"uptime":0}`,
			expected: ethstats.NodeStats{
				Active:   false,
				Syncing:  false,
				Peers:    0,
				GasPrice: 0,
				Uptime:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stats ethstats.NodeStats

			err := json.Unmarshal([]byte(tt.input), &stats)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, stats)
		})
	}
}

func TestPrimusProtocol(t *testing.T) {
	// Test primus ping prefix
	pingInput := "primus::ping::1702000000000"
	assert.True(t, len(pingInput) > len(ethstats.PrimusPingPrefix))
	assert.Equal(t, ethstats.PrimusPingPrefix, pingInput[:len(ethstats.PrimusPingPrefix)])

	// Test primus pong prefix
	pongInput := "primus::pong::1702000000000"
	assert.True(t, len(pongInput) > len(ethstats.PrimusPongPrefix))
	assert.Equal(t, ethstats.PrimusPongPrefix, pongInput[:len(ethstats.PrimusPongPrefix)])
}

func TestCommandConstants(t *testing.T) {
	// Verify command constants match expected ethstats protocol values
	assert.Equal(t, "hello", ethstats.CommandHello)
	assert.Equal(t, "ready", ethstats.CommandReady)
	assert.Equal(t, "node-ping", ethstats.CommandNodePing)
	assert.Equal(t, "node-pong", ethstats.CommandNodePong)
	assert.Equal(t, "latency", ethstats.CommandLatency)
	assert.Equal(t, "block", ethstats.CommandBlock)
	assert.Equal(t, "block_new_payload", ethstats.CommandBlockNewPayload)
	assert.Equal(t, "pending", ethstats.CommandPending)
	assert.Equal(t, "stats", ethstats.CommandStats)
	assert.Equal(t, "history", ethstats.CommandHistory)

	// Verify primus prefixes
	assert.Equal(t, "primus::ping::", ethstats.PrimusPingPrefix)
	assert.Equal(t, "primus::pong::", ethstats.PrimusPongPrefix)
}
