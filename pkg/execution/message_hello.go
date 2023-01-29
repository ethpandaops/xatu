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
	minETHProtocolVersion = uint(66)
	// TODO: implement eth/68
	maxETHProtocolVersion = uint(67)
	ETHCapName            = "eth"
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
		if cap.Name == ETHCapName {
			return &cap
		}
	}

	return nil
}

func (h *Hello) Validate() error {
	if h.Version < minP2PProtocolVersion {
		return fmt.Errorf("peer is using unsupported p2p protocol version: %d", h.Version)
	}

	supportsOurETHProtocolVersion := false
	highestETHProtocolVersion := uint(0)

	for _, cap := range h.Caps {
		if cap.Name == ETHCapName {
			if cap.Version == minETHProtocolVersion {
				supportsOurETHProtocolVersion = true
			}

			if cap.Version > highestETHProtocolVersion {
				highestETHProtocolVersion = cap.Version
			}
		}
	}

	if highestETHProtocolVersion == 0 {
		return fmt.Errorf("peer does not support eth protocol")
	}

	if !supportsOurETHProtocolVersion {
		return fmt.Errorf("peer is using unsupported eth protocol version: %d", minETHProtocolVersion)
	}

	return nil
}

func (h *Hello) ETHProtocolVersion() uint {
	highestETHProtocolVersion := uint(0)

	for _, cap := range h.Caps {
		if cap.Name == ETHCapName {
			if cap.Version > highestETHProtocolVersion && cap.Version <= maxETHProtocolVersion {
				highestETHProtocolVersion = cap.Version
			}
		}
	}

	return highestETHProtocolVersion
}

func SupportedEthCaps() []p2p.Cap {
	caps := []p2p.Cap{}
	for i := minETHProtocolVersion; i <= maxETHProtocolVersion; i++ {
		caps = append(caps, p2p.Cap{
			Name:    ETHCapName,
			Version: i,
		})
	}

	return caps
}

func (c *Client) receiveHello(ctx context.Context, data []byte) (*Hello, error) {
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

func (c *Client) sendHello(ctx context.Context) error {
	c.log.WithFields(logrus.Fields{
		"code": HelloCode,
	}).Debug("sending Hello")

	pub0 := crypto.FromECDSAPub(&c.privateKey.PublicKey)[1:]
	hello := &Hello{
		Version: P2PProtocolVersion,
		Caps:    SupportedEthCaps(),
		ID:      pub0,
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

func (c *Client) handleHello(ctx context.Context, code uint64, data []byte) error {
	c.log.WithField("code", code).Debug("received Hello")

	hello, err := c.receiveHello(ctx, data)
	if err != nil {
		return err
	}

	c.ethCapVersion = hello.ETHProtocolVersion()

	c.publishHello(ctx, hello)

	// always enable snappy to avoid jank
	c.rlpxConn.SetSnappy(true)

	return nil
}
