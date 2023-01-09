// eth protocol block receipts https://github.com/ethereum/devp2p/blob/master/caps/eth.md#blockreceipts-0x06
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	ReceiptsCode = 0x20
)

type Receipts eth.ReceiptsPacket66

func (msg *Receipts) Code() int { return ReceiptsCode }

func (msg *Receipts) ReqID() uint64 { return msg.RequestId }

func (c *Client) handleReceipts(ctx context.Context, data []byte) (*Receipts, error) {
	s := new(Receipts)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding block receipts: %w", err)
	}

	return s, nil
}

func (c *Client) sendReceipts(ctx context.Context, bh *Receipts) error {
	c.log.WithFields(logrus.Fields{
		"code":           ReceiptsCode,
		"request_id":     bh.RequestId,
		"receipts_count": len(bh.ReceiptsPacket),
	}).Debug("sending Receipts")

	encodedData, err := rlp.EncodeToBytes(bh)
	if err != nil {
		return fmt.Errorf("error encoding block receipts: %w", err)
	}

	if _, err := c.rlpxConn.Write(ReceiptsCode, encodedData); err != nil {
		return fmt.Errorf("error sending block receipts: %w", err)
	}

	return nil
}
