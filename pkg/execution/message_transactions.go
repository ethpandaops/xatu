// eth protocol transactions https://github.com/ethereum/devp2p/blob/master/caps/eth.md#transactions-0x02
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	TransactionsCode = 0x12
)

type Transactions eth.TransactionsPacket

func (msg *Transactions) Code() int { return TransactionsCode }

func (msg *Transactions) ReqID() uint64 { return 0 }

func (c *Client) handleTransactions(ctx context.Context, data []byte) (*Transactions, error) {
	s := new(Transactions)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding transactions: %w", err)
	}

	return s, nil
}
