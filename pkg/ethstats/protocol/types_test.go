package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockNumber_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
		wantErr  bool
	}{
		{
			name:     "number format",
			input:    "12345678",
			expected: 12345678,
		},
		{
			name:     "hex string format",
			input:    `"0xbc614e"`,
			expected: 12345678,
		},
		{
			name:     "decimal string format",
			input:    `"12345678"`,
			expected: 12345678,
		},
		{
			name:    "invalid hex",
			input:   `"0xzzzz"`,
			wantErr: true,
		},
		{
			name:    "invalid decimal string",
			input:   `"abc123"`,
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   `true`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bn BlockNumber
			err := json.Unmarshal([]byte(tt.input), &bn)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, bn.Value())
			}
		})
	}
}

func TestBlockNumber_MarshalJSON(t *testing.T) {
	bn := &BlockNumber{}
	data, _ := json.Marshal(uint64(12345678))
	_ = bn.UnmarshalJSON(data)

	marshaled, err := json.Marshal(bn)
	require.NoError(t, err)
	assert.Equal(t, "12345678", string(marshaled))
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "valid message",
			input: `{"emit":["hello",{"id":"test"}]}`,
		},
		{
			name:  "empty emit array",
			input: `{"emit":[]}`,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "missing emit field",
			input:   `{"data":"test"}`,
			wantErr: false, // Will parse but emit will be nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, msg)
			}
		})
	}
}

func TestParseHelloMessage(t *testing.T) {
	validHello := []interface{}{
		"hello",
		map[string]interface{}{
			"id": "test-node",
			"info": map[string]interface{}{
				"name":             "test-node",
				"node":             "Geth/v1.10.0",
				"port":             30303,
				"net":              "1",
				"protocol":         "eth/66",
				"api":              "no",
				"os":               "linux",
				"os_v":             "x64",
				"client":           "0.1.1",
				"canUpdateHistory": true,
			},
			"secret": "test-secret",
		},
	}

	hello, err := ParseHelloMessage(validHello)
	require.NoError(t, err)
	assert.Equal(t, "test-node", hello.ID)
	assert.Equal(t, "test-secret", hello.Secret)
	assert.Equal(t, "test-node", hello.Info.Name)
	assert.Equal(t, "Geth/v1.10.0", hello.Info.Node)
	assert.Equal(t, 30303, hello.Info.Port)
	assert.Equal(t, "1", hello.Info.Net)
	assert.Equal(t, "eth/66", hello.Info.Protocol)
	assert.True(t, hello.Info.CanUpdateHistory)

	// Test invalid cases
	_, err = ParseHelloMessage([]interface{}{"hello"})
	assert.Error(t, err, "should fail with too few elements")

	_, err = ParseHelloMessage([]interface{}{"not-hello", map[string]interface{}{}})
	assert.Error(t, err, "should fail with wrong message type")
}

func TestParseBlockReport(t *testing.T) {
	validBlock := []interface{}{
		"block",
		map[string]interface{}{
			"id": "test-node",
			"block": map[string]interface{}{
				"number":           12345678,
				"hash":             "0x1234",
				"parentHash":       "0x5678",
				"timestamp":        1234567890,
				"miner":            "0xabc",
				"gasUsed":          15000000,
				"gasLimit":         30000000,
				"difficulty":       "1234567890",
				"totalDifficulty":  "12345678901234567890",
				"transactions":     []interface{}{map[string]interface{}{"hash": "0xtx1"}},
				"transactionsRoot": "0xtxroot",
				"stateRoot":        "0xstateroot",
				"uncles":           []interface{}{},
			},
		},
	}

	report, err := ParseBlockReport(validBlock)
	require.NoError(t, err)
	assert.Equal(t, "test-node", report.ID)
	assert.Equal(t, uint64(12345678), report.Block.Number.Value())
	assert.Equal(t, "0x1234", report.Block.Hash)
	assert.Equal(t, "0x5678", report.Block.ParentHash)
	assert.Equal(t, int64(1234567890), report.Block.Timestamp)
	assert.Equal(t, "0xabc", report.Block.Miner)
	assert.Equal(t, 1, len(report.Block.Transactions))
	assert.Equal(t, 0, len(report.Block.Uncles))
}

func TestParseStatsReport(t *testing.T) {
	validStats := []interface{}{
		"stats",
		map[string]interface{}{
			"id": "test-node",
			"stats": map[string]interface{}{
				"active":   true,
				"syncing":  false,
				"mining":   false,
				"hashrate": 0,
				"peers":    25,
				"gasPrice": 30000000000,
				"uptime":   100,
			},
		},
	}

	report, err := ParseStatsReport(validStats)
	require.NoError(t, err)
	assert.Equal(t, "test-node", report.ID)
	assert.True(t, report.Stats.Active)
	assert.False(t, report.Stats.Syncing)
	assert.False(t, report.Stats.Mining)
	assert.Equal(t, 25, report.Stats.Peers)
	assert.Equal(t, int64(30000000000), report.Stats.GasPrice)
	assert.Equal(t, 100, report.Stats.Uptime)
}

func TestParsePendingReport(t *testing.T) {
	validPending := []interface{}{
		"pending",
		map[string]interface{}{
			"id": "test-node",
			"stats": map[string]interface{}{
				"pending": 150,
			},
		},
	}

	report, err := ParsePendingReport(validPending)
	require.NoError(t, err)
	assert.Equal(t, "test-node", report.ID)
	assert.Equal(t, 150, report.Stats.Pending)
}

func TestParseNodePing(t *testing.T) {
	validPing := []interface{}{
		"node-ping",
		map[string]interface{}{
			"id":         "test-node",
			"clientTime": "1234567890",
		},
	}

	ping, err := ParseNodePing(validPing)
	require.NoError(t, err)
	assert.Equal(t, "test-node", ping.ID)
	assert.Equal(t, "1234567890", ping.ClientTime)
}

func TestParseLatencyReport(t *testing.T) {
	validLatency := []interface{}{
		"latency",
		map[string]interface{}{
			"id":      "test-node",
			"latency": 25,
		},
	}

	report, err := ParseLatencyReport(validLatency)
	require.NoError(t, err)
	assert.Equal(t, "test-node", report.ID)
	assert.Equal(t, 25, report.Latency)
}

func TestParseHistoryReport(t *testing.T) {
	validHistory := []interface{}{
		"history",
		map[string]interface{}{
			"id": "test-node",
			"history": []interface{}{
				map[string]interface{}{
					"number":           12345677,
					"hash":             "0x1233",
					"parentHash":       "0x5677",
					"timestamp":        1234567889,
					"miner":            "0xabc",
					"gasUsed":          14000000,
					"gasLimit":         30000000,
					"difficulty":       "1234567889",
					"totalDifficulty":  "12345678901234567889",
					"transactions":     []interface{}{},
					"transactionsRoot": "0xtxroot",
					"stateRoot":        "0xstateroot",
					"uncles":           []interface{}{},
				},
			},
		},
	}

	report, err := ParseHistoryReport(validHistory)
	require.NoError(t, err)
	assert.Equal(t, "test-node", report.ID)
	assert.Equal(t, 1, len(report.History))
	assert.Equal(t, uint64(12345677), report.History[0].Number.Value())
}

func TestFormatReadyMessage(t *testing.T) {
	msg := FormatReadyMessage()

	var parsed map[string]interface{}
	err := json.Unmarshal(msg, &parsed)
	require.NoError(t, err)

	emit, ok := parsed["emit"].([]interface{})
	require.True(t, ok)
	require.Equal(t, 1, len(emit))
	assert.Equal(t, "ready", emit[0])
}

func TestFormatNodePong(t *testing.T) {
	ping := &NodePing{
		ID:         "test-node",
		ClientTime: "1234567890",
	}

	msg := FormatNodePong(ping, "1234567891")

	var parsed map[string]interface{}
	err := json.Unmarshal(msg, &parsed)
	require.NoError(t, err)

	emit, ok := parsed["emit"].([]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(emit))
	assert.Equal(t, "node-pong", emit[0])

	pongData, ok := emit[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-node", pongData["id"])
	assert.Equal(t, "1234567890", pongData["clientTime"])
	assert.Equal(t, "1234567891", pongData["serverTime"])
}

func TestFormatHistoryRequest(t *testing.T) {
	msg := FormatHistoryRequest(50, 1)

	var parsed map[string]interface{}
	err := json.Unmarshal(msg, &parsed)
	require.NoError(t, err)

	emit, ok := parsed["emit"].([]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(emit))
	assert.Equal(t, "history", emit[0])

	reqData, ok := emit[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(50), reqData["max"])
	assert.Equal(t, float64(1), reqData["min"])
}

func TestFormatPrimusPing(t *testing.T) {
	msg := FormatPrimusPing(1234567890)

	var parsed map[string]interface{}
	err := json.Unmarshal(msg, &parsed)
	require.NoError(t, err)

	emit, ok := parsed["emit"].([]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(emit))
	assert.Equal(t, "primus::ping::", emit[0])
	assert.Equal(t, float64(1234567890), emit[1])
}
