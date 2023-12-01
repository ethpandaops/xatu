package armiarma

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/mitchellh/hashstructure/v2"
)

// TimedEthereumAttestation contains the data for an Ethereum Attestation that was received
// along with extra data such as when it arrived and who sent it
type TimedEthereumAttestation struct {
	Attestation *phase0.Attestation `json:"attestation"`
	//nolint:tagliatelle // Defined by API.
	AttestationExtraData *AttestationExtraData `json:"attestation_extra_data"`
	//nolint:tagliatelle // Defined by API.
	PeerInfo *PeerInfo `json:"peer_info"`
}

// PeerInfo contains information about a peer
type PeerInfo struct {
	ID   string `json:"id"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	//nolint:tagliatelle // Defined by API.
	UserAgent string        `json:"user_agent"`
	Latency   time.Duration `json:"latency"`
	Protocols []string      `json:"protocols"`
	//nolint:tagliatelle // Defined by API.
	ProtocolVersion string `json:"protocol_version"`
}

// AttestationExtraData contains extra data for an attestation
type AttestationExtraData struct {
	//nolint:tagliatelle // Defined by API.
	ArrivedAt time.Time `json:"arrived_at"`
	//nolint:tagliatelle // Defined by API.
	P2PMsgID string `json:"peer_msg_id"`
	Subnet   int    `json:"subnet"`
	//nolint:tagliatelle // Defined by API.
	TimeInSlot time.Duration `json:"time_in_slot"`
}

func (a *TimedEthereumAttestation) AttestationHash() (string, error) {
	hash, err := hashstructure.Hash(a.Attestation, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(hash), nil
}
