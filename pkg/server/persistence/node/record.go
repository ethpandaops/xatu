package node

import (
	"database/sql"
	"time"
)

type Record struct {
	// Enr is the enr of the node record.
	Enr string `json:"enr" db:"enr"`
	// Signature is the cryptographic signature of record contents
	Signature *[]byte `json:"signature" db:"signature" fieldopt:"omitempty"`
	// Seq is the sequence number, a 64-bit unsigned integer. Nodes should increase the number whenever the record changes and republish the record
	Seq *uint64 `json:"seq" db:"seq" fieldopt:"omitempty"`
	// CreateTime is the timestamp of when the node record was created.
	CreateTime time.Time `json:"createTime" db:"create_time" fieldopt:"omitempty"`
	// LastDialTime is the timestamp of when the node record was last dialed.
	LastDialTime sql.NullTime `json:"lastDialTime" db:"last_dial_time" fieldopt:"omitempty"`
	// ConsecutiveDialAttempts is the number of consecutive dial attempts.
	ConsecutiveDialAttempts int `json:"consecutiveDialAttempts" db:"consecutive_dial_attempts"`
	// LastConnectTime is the timestamp of when the node record was last connected.
	LastConnectTime sql.NullTime `json:"lastConnectTime" db:"last_connect_time" fieldopt:"omitempty"`
	// ID is the name of identity scheme, e.g. “v4”
	ID *string `json:"id" db:"id" fieldopt:"omitempty"`
	// Secp256k1 is the secp256k1 public key of the node record.
	Secp256k1 *[]byte `json:"secp256k1" db:"secp256k1" fieldopt:"omitempty"`
	// IP4 is the IPv4 address of the node record.
	IP4 *string `json:"ip4" db:"ip4" fieldopt:"omitempty"`
	// IP6 is the IPv6 address of the node record.
	IP6 *string `json:"ip6" db:"ip6" fieldopt:"omitempty"`
	// GeoCity is the city of the node record.
	GeoCity *string `json:"geoCity" db:"geo_city" fieldopt:"omitempty"`
	// GeoCountry is the country of the node record.
	GeoCountry *string `json:"geoCountry" db:"geo_country" fieldopt:"omitempty"`
	// GeoCountryCode is the country code of the node record.
	GeoCountryCode *string `json:"geoCountryCode" db:"geo_country_code" fieldopt:"omitempty"`
	// GeoContinentCode is the continent code of the node record.
	GeoContinentCode *string `json:"geoContinentCode" db:"geo_continent_code" fieldopt:"omitempty"`
	// GeoLongitude is the longitude of the node record.
	GeoLongitude *float64 `json:"geoLongitude" db:"geo_longitude" fieldopt:"omitempty"`
	// GeoLatitude is the latitude of the node record.
	GeoLatitude *float64 `json:"geoLatitude" db:"geo_latitude" fieldopt:"omitempty"`
	// GeoAutonomousSystemNumber is the autonomous system number of the node record.
	GeoAutonomousSystemNumber *uint32 `json:"geoAutonomousSystemNumber" db:"geo_autonomous_system_number" fieldopt:"omitempty"`
	// GeoAutonomousSystemOrganization is the autonomous system organization of the node record.
	GeoAutonomousSystemOrganization *string `json:"geoAutonomousSystemOrganization" db:"geo_autonomous_system_organization" fieldopt:"omitempty"`
	// TCP4 is the TCP port of the node record.
	TCP4 *uint32 `json:"tcp4" db:"tcp4" fieldopt:"omitempty"`
	// TCP6 is the TCP port of the node record.
	TCP6 *uint32 `json:"tcp6" db:"tcp6" fieldopt:"omitempty"`
	// UDP4 is the UDP port of the node record.
	UDP4 *uint32 `json:"udp4" db:"udp4" fieldopt:"omitempty"`
	// UDP6 is the UDP port of the node record.
	UDP6 *uint32 `json:"udp6" db:"udp6" fieldopt:"omitempty"`
	// Eth2 is the eth2 public key of the node record.
	ETH2 *[]byte `json:"eth2" db:"eth2" fieldopt:"omitempty"`
	// Attnets is the attestation subnet bitfield of the node record.
	Attnets *[]byte `json:"attnets" db:"attnets" fieldopt:"omitempty"`
	// Syncnets is the sync subnet bitfield of the node record.
	Syncnets *[]byte `json:"syncnets" db:"syncnets" fieldopt:"omitempty"`
	// NodeID is the node ID of the node record.
	NodeID *string `json:"nodeId" db:"node_id" fieldopt:"omitempty"`
	// PeerID is the peer ID of the node record.
	PeerID *string `json:"peerId" db:"peer_id" fieldopt:"omitempty"`
	// CGC is the custody group count of the node record.
	CGC *[]byte `json:"cgc" db:"cgc" fieldopt:"omitempty"`
	// NextForkDigest is the next fork digest of the next scheduled fork.
	NextForkDigest *[]byte `json:"nextForkDigest" db:"next_fork_digest" fieldopt:"omitempty"`
}
