package node

type Record struct {
	// Enr is the enr of the node record.
	Enr string `json:"enr"`
	// Signature is the cryptographic signature of record contents
	Signature *[]byte `json:"signature"`
	// Seq is the sequence number, a 64-bit unsigned integer. Nodes should increase the number whenever the record changes and republish the record
	Seq *uint64 `json:"seq"`
	// CreatedTimestamp is the timestamp of when the node record was created.
	CreatedTimestamp *uint64 `json:"created_timestamp"`
	// ID is the name of identity scheme, e.g. “v4”
	ID *string `json:"id"`
	// Secp256k1 is the secp256k1 public key of the node record.
	Secp256k1 *[]byte `json:"secp256k1"`
	// IP4 is the IPv4 address of the node record.
	IP4 *string `json:"ip4"`
	// IP6 is the IPv6 address of the node record.
	IP6 *string `json:"ip6"`
	// TCP4 is the TCP port of the node record.
	TCP4 *uint32 `json:"tcp4"`
	// TCP6 is the TCP port of the node record.
	TCP6 *uint32 `json:"tcp6"`
	// UDP4 is the UDP port of the node record.
	UDP4 *uint32 `json:"udp4"`
	// UDP6 is the UDP port of the node record.
	UDP6 *uint32 `json:"udp6"`
	// Eth2 is the eth2 public key of the node record.
	ETH2 *[]byte `json:"eth2"`
	// Attnets is the attestation subnet bitfield of the node record.
	Attnets *[]byte `json:"attnets"`
	// Syncnets is the sync subnet bitfield of the node record.
	Syncnets *[]byte `json:"syncnets"`
	// NodeID is the node ID of the node record.
	NodeID *string `json:"node_id"`
	// PeerID is the peer ID of the node record.
	PeerID *string `json:"peer_id"`
}
