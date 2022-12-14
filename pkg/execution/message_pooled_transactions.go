// eth protocol get get block headers https://github.com/ethereum/devp2p/blob/master/caps/eth.md#getblockheaders-0x03
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

const (
	PooledTransactionsCode = 0x1a
)

type PooledTransactions eth.PooledTransactionsPacket66

func (msg *PooledTransactions) Code() int { return PooledTransactionsCode }

func (msg *PooledTransactions) ReqID() uint64 { return msg.RequestId }

func (c *Client) handlePooledTransactions(ctx context.Context, data []byte) (*PooledTransactions, error) {
	s := new(PooledTransactions)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding get block headers: %w", err)
	}

	return s, nil
}

func (c *Client) sendPooledTransactions(ctx context.Context, pt *PooledTransactions) error {
	c.log.WithFields(logrus.Fields{
		"code":       PooledTransactionsCode,
		"request_id": pt.RequestId,
		"txs_count":  len(pt.PooledTransactionsPacket),
	}).Debug("sending PooledTransactions")

	encodedData, err := rlp.EncodeToBytes(pt)
	if err != nil {
		return fmt.Errorf("error encoding get block headers: %w", err)
	}

	if _, err := c.rlpxConn.Write(PooledTransactionsCode, encodedData); err != nil {
		return fmt.Errorf("error sending get block headers: %w", err)
	}

	return nil
}
