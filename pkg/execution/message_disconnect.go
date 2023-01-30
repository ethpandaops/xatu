// RLPx disconnect https://github.com/ethereum/devp2p/blob/master/rlpx.md#disconnect-0x01
package execution

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	DisconnectCode = 0x01
)

type Disconnect struct {
	Reason p2p.DiscReason
}

func (h *Disconnect) Code() int { return DisconnectCode }

func (h *Disconnect) ReqID() uint64 { return 0 }

func (c *Client) receiveDisconnect(ctx context.Context, data []byte) *Disconnect {
	reason := data[0:1]
	// besu sends 2 byte disconnect message
	if len(data) > 1 {
		reason = data[1:2]
	}

	d := new(p2p.DiscReason)
	if err := rlp.DecodeBytes(reason, &d); err != nil {
		c.log.WithError(err).Debug("Error decoding disconnect")
	}

	return &Disconnect{Reason: *d}
}

func (c *Client) handleDisconnect(ctx context.Context, code uint64, data []byte) {
	c.log.WithField("code", code).Debug("received Disconnect")

	disconnect := c.receiveDisconnect(ctx, data)

	c.disconnect(ctx, disconnect)
}
