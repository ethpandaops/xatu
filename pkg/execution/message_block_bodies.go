// eth protocol block bodies https://github.com/ethereum/devp2p/blob/master/caps/eth.md#blockbodies-0x06
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	BlockBodiesCode = 0x16
)

type BlockBodies eth.BlockBodiesPacket66

func (msg *BlockBodies) Code() int { return BlockBodiesCode }

func (msg *BlockBodies) ReqID() uint64 { return msg.RequestId }

func (c *Client) handleBlockBodies(ctx context.Context, data []byte) (*BlockBodies, error) {
	s := new(BlockBodies)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding block bodies: %w", err)
	}

	return s, nil
}

func (c *Client) sendBlockBodies(ctx context.Context, bh *BlockBodies) error {
	c.log.WithFields(logrus.Fields{
		"code":         BlockBodiesCode,
		"request_id":   bh.RequestId,
		"bodies_count": len(bh.BlockBodiesPacket),
	}).Debug("sending BlockBodies")

	encodedData, err := rlp.EncodeToBytes(bh)
	if err != nil {
		return fmt.Errorf("error encoding block bodies: %w", err)
	}

	if _, err := c.rlpxConn.Write(BlockBodiesCode, encodedData); err != nil {
		return fmt.Errorf("error sending block bodies: %w", err)
	}

	return nil
}
