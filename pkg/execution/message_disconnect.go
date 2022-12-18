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

func (c *Client) handleDisconnect(ctx context.Context, data []byte) *Disconnect {
	d := new(Disconnect)
	if err := rlp.DecodeBytes(data, &d); err != nil {
		c.log.WithError(err).Error("Error decoding disconnect")
	}

	return d
}
