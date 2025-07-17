package protocol

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Message is the wrapper for all ethstats protocol messages
type Message struct {
	Emit []interface{} `json:"emit"`
}

// HelloMessage is sent by the client on connection
type HelloMessage struct {
	ID     string   `json:"id"`
	Info   NodeInfo `json:"info"`
	Secret string   `json:"secret"`
}

// NodeInfo contains information about the connected node
type NodeInfo struct {
	Name             string `json:"name"`
	Node             string `json:"node"`
	Port             int    `json:"port"`
	Net              string `json:"net"`
	Protocol         string `json:"protocol"`
	API              string `json:"api"`
	OS               string `json:"os"`
	OSVersion        string `json:"osV"`
	Client           string `json:"client"`
	CanUpdateHistory bool   `json:"canUpdateHistory"`
	Contact          string `json:"contact,omitempty"`
}

// BlockReport is sent by the client to report new blocks
type BlockReport struct {
	ID    string `json:"id"`
	Block Block  `json:"block"`
}

// BlockNumber handles both hex string and number formats
type BlockNumber struct {
	value uint64
}

// UnmarshalJSON implements json.Unmarshaler
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as number first
	var num uint64
	if err := json.Unmarshal(data, &num); err == nil {
		bn.value = num

		return nil
	}

	// Try to unmarshal as string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("block number must be either number or string: %w", err)
	}

	// Handle hex string
	if strings.HasPrefix(str, "0x") {
		val, err := strconv.ParseUint(str[2:], 16, 64)
		if err != nil {
			return fmt.Errorf("invalid hex block number: %w", err)
		}

		bn.value = val

		return nil
	}

	// Handle decimal string
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid decimal block number: %w", err)
	}

	bn.value = val

	return nil
}

// Value returns the block number as uint64
func (bn BlockNumber) Value() uint64 {
	return bn.value
}

// MarshalJSON implements json.Marshaler
func (bn BlockNumber) MarshalJSON() ([]byte, error) {
	return json.Marshal(bn.value)
}

// Block represents a block in the blockchain
type Block struct {
	Number           BlockNumber `json:"number"`
	Hash             string      `json:"hash"`
	ParentHash       string      `json:"parentHash"`
	Timestamp        int64       `json:"timestamp"`
	Miner            string      `json:"miner"`
	GasUsed          int64       `json:"gasUsed"`
	GasLimit         int64       `json:"gasLimit"`
	Difficulty       string      `json:"difficulty"`
	TotalDifficulty  string      `json:"totalDifficulty"`
	Transactions     []TxHash    `json:"transactions"`
	TransactionsRoot string      `json:"transactionsRoot"`
	StateRoot        string      `json:"stateRoot"`
	Uncles           []string    `json:"uncles"`
}

// TxHash represents a transaction hash
type TxHash struct {
	Hash string `json:"hash"`
}

// StatsReport contains node statistics
type StatsReport struct {
	ID    string    `json:"id"`
	Stats NodeStats `json:"stats"`
}

// NodeStats represents the current statistics of a node
type NodeStats struct {
	Active   bool  `json:"active"`
	Syncing  bool  `json:"syncing"`
	Mining   bool  `json:"mining"`
	Hashrate int64 `json:"hashrate"`
	Peers    int   `json:"peers"`
	GasPrice int64 `json:"gasPrice"`
	Uptime   int   `json:"uptime"`
}

// PendingReport contains pending transaction count
type PendingReport struct {
	ID    string       `json:"id"`
	Stats PendingStats `json:"stats"`
}

// PendingStats contains the pending transaction count
type PendingStats struct {
	Pending int `json:"pending"`
}

// NodePing is sent by the client to measure latency
type NodePing struct {
	ID         string `json:"id"`
	ClientTime string `json:"clientTime"`
}

// NodePong is the server's response to a ping
type NodePong struct {
	ID         string `json:"id"`
	ClientTime string `json:"clientTime"`
	ServerTime string `json:"serverTime"`
}

// LatencyReport is sent by the client to report measured latency
type LatencyReport struct {
	ID      string `json:"id"`
	Latency int    `json:"latency"`
}

// HistoryRequest requests historical blocks
type HistoryRequest struct {
	Max int `json:"max"`
	Min int `json:"min"`
}

// HistoryReport contains historical blocks
type HistoryReport struct {
	ID      string  `json:"id"`
	History []Block `json:"history"`
}

// ParseMessage parses the initial message wrapper
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &msg, nil
}

// ParseHelloMessage parses a hello message from the emit array
func ParseHelloMessage(emit []interface{}) (*HelloMessage, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("hello message requires at least 2 elements")
	}

	if emit[0] != "hello" {
		return nil, fmt.Errorf("not a hello message")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hello data: %w", err)
	}

	var hello HelloMessage
	if err := json.Unmarshal(data, &hello); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hello message: %w", err)
	}

	return &hello, nil
}

// ParseBlockReport parses a block report from the emit array
func ParseBlockReport(emit []interface{}) (*BlockReport, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("block report requires at least 2 elements")
	}

	if emit[0] != "block" {
		return nil, fmt.Errorf("not a block report")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block data: %w", err)
	}

	var report BlockReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block report: %w", err)
	}

	return &report, nil
}

// ParseStatsReport parses a stats report from the emit array
func ParseStatsReport(emit []interface{}) (*StatsReport, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("stats report requires at least 2 elements")
	}

	if emit[0] != "stats" {
		return nil, fmt.Errorf("not a stats report")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stats data: %w", err)
	}

	var report StatsReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats report: %w", err)
	}

	return &report, nil
}

// ParsePendingReport parses a pending report from the emit array
func ParsePendingReport(emit []interface{}) (*PendingReport, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("pending report requires at least 2 elements")
	}

	if emit[0] != "pending" {
		return nil, fmt.Errorf("not a pending report")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pending data: %w", err)
	}

	var report PendingReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pending report: %w", err)
	}

	return &report, nil
}

// ParseNodePing parses a node ping from the emit array
func ParseNodePing(emit []interface{}) (*NodePing, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("node ping requires at least 2 elements")
	}

	if emit[0] != "node-ping" {
		return nil, fmt.Errorf("not a node ping")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ping data: %w", err)
	}

	var ping NodePing
	if err := json.Unmarshal(data, &ping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node ping: %w", err)
	}

	return &ping, nil
}

// ParseLatencyReport parses a latency report from the emit array
func ParseLatencyReport(emit []interface{}) (*LatencyReport, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("latency report requires at least 2 elements")
	}

	if emit[0] != "latency" {
		return nil, fmt.Errorf("not a latency report")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal latency data: %w", err)
	}

	var report LatencyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal latency report: %w", err)
	}

	return &report, nil
}

// ParseHistoryReport parses a history report from the emit array
func ParseHistoryReport(emit []interface{}) (*HistoryReport, error) {
	if len(emit) < 2 {
		return nil, fmt.Errorf("history report requires at least 2 elements")
	}

	if emit[0] != "history" {
		return nil, fmt.Errorf("not a history report")
	}

	data, err := json.Marshal(emit[1])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal history data: %w", err)
	}

	var report HistoryReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history report: %w", err)
	}

	return &report, nil
}

// FormatReadyMessage formats a ready message for the client
func FormatReadyMessage() []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{"ready"},
	}
	data, _ := json.Marshal(msg)

	return data
}

// FormatNodePong formats a node pong response
func FormatNodePong(ping *NodePing, serverTime string) []byte {
	pong := NodePong{
		ID:         ping.ID,
		ClientTime: ping.ClientTime,
		ServerTime: serverTime,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"node-pong", pong},
	}
	data, _ := json.Marshal(msg)

	return data
}

// FormatHistoryRequest formats a history request
func FormatHistoryRequest(maxVal, minVal int) []byte {
	req := HistoryRequest{
		Max: maxVal,
		Min: minVal,
	}

	msg := map[string]interface{}{
		"emit": []interface{}{"history", req},
	}
	data, _ := json.Marshal(msg)

	return data
}

// FormatPrimusPing formats a primus protocol ping
func FormatPrimusPing(timestamp int64) []byte {
	msg := map[string]interface{}{
		"emit": []interface{}{"primus::ping::", timestamp},
	}
	data, _ := json.Marshal(msg)

	return data
}
