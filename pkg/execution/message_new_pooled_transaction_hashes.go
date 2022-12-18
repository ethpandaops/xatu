// eth protocol new pooled transaction hashes https://github.com/ethereum/devp2p/blob/master/caps/eth.md#newpooledtransactionhashes-0x08
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	NewPooledTransactionHashesCode = 0x18
)

type NewPooledTransactionHashes eth.NewPooledTransactionHashesPacket

func (msg *NewPooledTransactionHashes) Code() int { return NewPooledTransactionHashesCode }

func (msg *NewPooledTransactionHashes) ReqID() uint64 { return 0 }

func (c *Client) handleNewPooledTransactionHashes(ctx context.Context, data []byte) (*NewPooledTransactionHashes, error) {
	s := new(NewPooledTransactionHashes)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding new pooled transaction hashes: %w", err)
	}

	return s, nil
}

func (c *Client) sendNewPooledTransactionHashes(ctx context.Context, pth *NewPooledTransactionHashes) error {
	c.log.WithFields(logrus.Fields{
		"code":         NewPooledTransactionHashesCode,
		"hashes_count": len(*pth),
	}).Debug("sending NewPooledTransactionHashes")

	encodedData, err := rlp.EncodeToBytes(pth)
	if err != nil {
		return fmt.Errorf("error encoding new pooled transaction hashes: %w", err)
	}

	if _, err := c.rlpxConn.Write(NewPooledTransactionHashesCode, encodedData); err != nil {
		return fmt.Errorf("error sending new pooled transaction hashes: %w", err)
	}

	return nil
}
