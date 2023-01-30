// RLPx ping https://github.com/ethereum/devp2p/blob/master/rlpx.md#ping-0x02
package execution

import "context"

const (
	PingCode = 0x02
)

type Ping struct{}

func (h *Ping) Code() int { return PingCode }

func (h *Ping) ReqID() uint64 { return 0 }

func (c *Client) handlePing(ctx context.Context, code uint64, data []byte) error {
	c.log.WithField("code", code).Debug("received Ping")

	if err := c.sendPong(ctx); err != nil {
		return err
	}

	return nil
}
