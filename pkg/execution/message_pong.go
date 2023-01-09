// RLPx pong https://github.com/ethereum/devp2p/blob/master/rlpx.md#pong-0x03
package execution

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

const (
	PongCode = 0x03
)

type Pong struct{}

func (h *Pong) Code() int { return PongCode }

func (h *Pong) ReqID() uint64 { return 0 }

func (c *Client) sendPong(ctx context.Context) error {
	c.log.WithFields(logrus.Fields{
		"code": PongCode,
	}).Debug("sending Pong")

	if _, err := c.rlpxConn.Write(PongCode, []byte{}); err != nil {
		return fmt.Errorf("error sending pong: %w", err)
	}

	return nil
}
