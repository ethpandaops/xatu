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
	GetBlockHeadersCode = 0x13
)

type GetBlockHeaders eth.GetBlockHeadersPacket66

func (msg *GetBlockHeaders) Code() int { return GetBlockHeadersCode }

func (msg *GetBlockHeaders) ReqID() uint64 { return msg.RequestId }

func (c *Client) handleGetBlockHeaders(ctx context.Context, data []byte) (*GetBlockHeaders, error) {
	s := new(GetBlockHeaders)
	if err := rlp.DecodeBytes(data, &s); err != nil {
		return nil, fmt.Errorf("error decoding get block headers: %w", err)
	}

	return s, nil
}

func (c *Client) sendGetBlockHeaders(ctx context.Context, bh *GetBlockHeaders) error {
	c.log.WithFields(logrus.Fields{
		"code":       GetBlockHeadersCode,
		"request_id": bh.RequestId,
		"headers":    bh.GetBlockHeadersPacket,
	}).Debug("sending GetBlockHeaders")

	encodedData, err := rlp.EncodeToBytes(bh)
	if err != nil {
		return fmt.Errorf("error encoding get block headers: %w", err)
	}

	if _, err := c.rlpxConn.Write(GetBlockHeadersCode, encodedData); err != nil {
		return fmt.Errorf("error sending get block headers: %w", err)
	}

	return nil
}
