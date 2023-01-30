// eth protocol get get block headers https://github.com/ethereum/devp2p/blob/master/caps/eth.md#getblockbodies-0x05
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	GetBlockBodiesCode = 0x15
)

type GetBlockBodies eth.GetBlockBodiesPacket66

func (msg *GetBlockBodies) Code() int { return GetBlockBodiesCode }

func (msg *GetBlockBodies) ReqID() uint64 { return msg.RequestId }

func (c *Client) receiveGetBlockBodies(ctx context.Context, data []byte) (*GetBlockBodies, error) {
	s := new(GetBlockBodies)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding get block bodies: %w", err)
	}

	return s, nil
}

func (c *Client) sendGetBlockBodies(ctx context.Context, bh *GetBlockBodies) error {
	c.log.WithFields(logrus.Fields{
		"code":       GetBlockBodiesCode,
		"request_id": bh.RequestId,
		"bodies":     bh.GetBlockBodiesPacket,
	}).Debug("sending GetBlockBodies")

	encodedData, err := rlp.EncodeToBytes(bh)
	if err != nil {
		return fmt.Errorf("error encoding get block bodies: %w", err)
	}

	if _, err := c.rlpxConn.Write(GetBlockBodiesCode, encodedData); err != nil {
		return fmt.Errorf("error sending get block bodies: %w", err)
	}

	return nil
}

func (c *Client) handleGetBlockBodies(ctx context.Context, code uint64, data []byte) error {
	c.log.WithField("code", code).Debug("received GetBlockBodies")

	blockBodies, err := c.receiveGetBlockBodies(ctx, data)
	if err != nil {
		return err
	}

	err = c.sendBlockBodies(ctx, &BlockBodies{
		RequestId:         blockBodies.RequestId,
		BlockBodiesPacket: []*eth.BlockBody{},
	})
	if err != nil {
		return err
	}

	return nil
}
