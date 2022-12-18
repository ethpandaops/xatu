// RLPx hello https://github.com/ethereum/devp2p/blob/master/rlpx.md#hello-0x00
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	HelloCode             = 0x00
	P2PProtocolVersion    = 5
	minP2PProtocolVersion = 5
	minETHProtocolVersion = 66
)

// https://github.com/ethereum/go-ethereum/blob/master/cmd/devp2p/internal/ethtest/types.go
type Hello struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte // secp256k1 public key

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

func (h *Hello) Code() int { return HelloCode }

func (h *Hello) ReqID() uint64 { return 0 }

func (h *Hello) ETHCap() *p2p.Cap {
	for _, cap := range h.Caps {
		if cap.Name == "eth" {
			return &cap
		}
	}

	return nil
}

func (h *Hello) MaxETHProtocolVersion() uint {
	ethCap := h.ETHCap()
	if ethCap == nil {
		return ETHProtocolVersion
	}

	if ethCap.Version > ETHProtocolVersion {
		return ETHProtocolVersion
	}

	return ethCap.Version
}

func (h *Hello) Validate() error {
	if h.Version < minP2PProtocolVersion {
		return fmt.Errorf("peer is using unsupported p2p protocol version: %d", h.Version)
	}

	ethCap := h.ETHCap()
	if ethCap == nil {
		return fmt.Errorf("peer does not support eth protocol")
	}

	if ethCap.Version < minETHProtocolVersion {
		return fmt.Errorf("peer is using unsupported eth protocol version: %d", ethCap.Version)
	}

	return nil
}

func (c *Client) handleHello(ctx context.Context, data []byte) (*Hello, error) {
	h := new(Hello)
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, fmt.Errorf("error decoding hello: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"version": h.Version,
		"caps":    h.Caps,
		"id":      h.ID,
	}).Debug("received hello message")

	if err := h.Validate(); err != nil {
		return nil, err
	}

	return h, nil
}

func (c *Client) sendHello(ctx context.Context, ethProtocolVersion uint) error {
	c.log.WithFields(logrus.Fields{
		"code":                 HelloCode,
		"eth_protocol_version": ethProtocolVersion,
	}).Debug("sending NewPooledTransactionHashes")

	pub0 := crypto.FromECDSAPub(&c.privateKey.PublicKey)[1:]
	hello := &Hello{
		Version: P2PProtocolVersion,
		Caps: []p2p.Cap{
			{
				Name:    "eth",
				Version: ethProtocolVersion,
			},
		},
		ID: pub0,
	}

	encodedData, err := rlp.EncodeToBytes(hello)
	if err != nil {
		return fmt.Errorf("error encoding hello: %w", err)
	}

	if _, err := c.rlpxConn.Write(HelloCode, encodedData); err != nil {
		return fmt.Errorf("error sending hello: %w", err)
	}

	return nil
}
