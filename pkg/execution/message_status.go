// eth protocol status https://github.com/ethereum/devp2p/blob/master/caps/eth.md#status-0x00
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	StatusCode = 0x10
)

type Status eth.StatusPacket

func (msg *Status) Code() int { return StatusCode }

func (msg *Status) ReqID() uint64 { return 0 }

func (c *Client) receiveStatus(ctx context.Context, data []byte) (*Status, error) {
	s := new(Status)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding status: %w", err)
	}

	return s, nil
}

func (c *Client) sendStatus(ctx context.Context, status *Status) error {
	c.log.WithFields(logrus.Fields{
		"code":   StatusCode,
		"status": status,
	}).Debug("sending Status")

	encodedData, err := rlp.EncodeToBytes(status)
	if err != nil {
		return fmt.Errorf("error encoding status: %w", err)
	}

	if _, err := c.rlpxConn.Write(StatusCode, encodedData); err != nil {
		return fmt.Errorf("error sending status: %w", err)
	}

	return nil
}

func (c *Client) handleStatus(ctx context.Context, code uint64, data []byte) error {
	c.log.WithField("code", code).Debug("received Status")

	status, err := c.receiveStatus(ctx, data)
	if err != nil {
		return err
	}

	c.publishStatus(ctx, status)

	if err := c.sendStatus(ctx, status); err != nil {
		return err
	}

	return nil
}
