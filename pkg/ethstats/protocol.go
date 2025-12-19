package ethstats

import "encoding/json"

// Message represents the basic ethstats protocol message format.
// All messages use the "emit" key with an array containing command and payload.
type Message struct {
	Emit []json.RawMessage `json:"emit"`
}

// AuthMsg is the authentication message sent by clients during hello.
type AuthMsg struct {
	ID     string   `json:"id"`
	Info   NodeInfo `json:"info"`
	Secret string   `json:"secret"`
}

// NodeInfo contains metadata about a connected node.
type NodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"` //nolint:tagliatelle // ethstats protocol uses os_v
	Client   string `json:"client"`
	History  bool   `json:"canUpdateHistory"`
}

// BlockStats contains information about a block.
type BlockStats struct {
	Number     json.Number `json:"number"`
	Hash       string      `json:"hash"`
	ParentHash string      `json:"parentHash"`
	Timestamp  json.Number `json:"timestamp"`
	Miner      string      `json:"miner"`
	GasUsed    uint64      `json:"gasUsed"`
	GasLimit   uint64      `json:"gasLimit"`
	Diff       string      `json:"difficulty"`
	TotalDiff  string      `json:"totalDifficulty"`
	Txs        []TxStats   `json:"transactions"`
	TxHash     string      `json:"transactionsRoot"`
	Root       string      `json:"stateRoot"`
	Uncles     []any       `json:"uncles"`
}

// TxStats contains information about a transaction.
type TxStats struct {
	Hash string `json:"hash"`
}

// BlockReport is the block message payload.
type BlockReport struct {
	ID    string      `json:"id"`
	Block *BlockStats `json:"block"`
}

// PendingStats contains pending transaction information.
type PendingStats struct {
	Pending int `json:"pending"`
}

// PendingReport is the pending transactions message payload.
type PendingReport struct {
	ID    string        `json:"id"`
	Stats *PendingStats `json:"stats"`
}

// NodeStats contains node status information.
type NodeStats struct {
	Active   bool `json:"active"`
	Syncing  bool `json:"syncing"`
	Peers    int  `json:"peers"`
	GasPrice int  `json:"gasPrice"`
	Uptime   int  `json:"uptime"`
}

// StatsReport is the stats message payload.
type StatsReport struct {
	ID    string     `json:"id"`
	Stats *NodeStats `json:"stats"`
}

// LatencyReport is the latency message payload.
type LatencyReport struct {
	ID      string `json:"id"`
	Latency string `json:"latency"`
}

// PingReport is the node-ping message payload.
type PingReport struct {
	ID         string `json:"id"`
	ClientTime string `json:"clientTime"`
}

// HistoryReport is the history message payload.
type HistoryReport struct {
	ID      string        `json:"id"`
	History []*BlockStats `json:"history"`
}

// NewPayloadStats contains information about a new payload event from engine API.
type NewPayloadStats struct {
	Number         json.Number `json:"number"`
	Hash           string      `json:"hash"`
	ProcessingTime uint64      `json:"processingTime"` // nanoseconds
}

// NewPayloadReport is the block_new_payload message payload.
type NewPayloadReport struct {
	ID    string           `json:"id"`
	Block *NewPayloadStats `json:"block"`
}

// Command types for ethstats protocol.
const (
	CommandHello           = "hello"
	CommandReady           = "ready"
	CommandNodePing        = "node-ping"
	CommandNodePong        = "node-pong"
	CommandLatency         = "latency"
	CommandBlock           = "block"
	CommandBlockNewPayload = "block_new_payload"
	CommandPending         = "pending"
	CommandStats           = "stats"
	CommandHistory         = "history"
)

// PrimusPingPrefix is the prefix for primus ping messages.
const PrimusPingPrefix = "primus::ping::"

// PrimusPongPrefix is the prefix for primus pong messages.
const PrimusPongPrefix = "primus::pong::"
