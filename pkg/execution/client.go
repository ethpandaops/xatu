package execution

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/sirupsen/logrus"
)

const (
	// TODO: handle eth68+ https://eips.ethereum.org/EIPS/eip-5793
	ETHProtocolVersion = 67
)

type Client struct {
	log logrus.FieldLogger

	nodeRecord *enode.Node
	broker     *emission.Emitter

	privateKey *ecdsa.PrivateKey

	peer    *p2p.Peer
	msgPipe *p2p.MsgPipeRW
	ethPeer *eth.Peer

	conn     net.Conn
	rlpxConn *rlpx.Conn

	pooledTransactionsMap map[uint64]chan *PooledTransactions
}

func parseNodeRecord(record string) (*enode.Node, error) {
	if strings.HasPrefix(record, "enode://") {
		return enode.ParseV4(record)
	}

	return enode.Parse(enode.ValidSchemes, record)
}

func New(ctx context.Context, log logrus.FieldLogger, record string) (*Client, error) {
	nodeRecord, err := parseNodeRecord(record)
	if err != nil {
		return nil, err
	}

	return &Client{
		log:                   log.WithField("node_record", nodeRecord.String()),
		nodeRecord:            nodeRecord,
		broker:                emission.NewEmitter(),
		pooledTransactionsMap: map[uint64]chan *PooledTransactions{},
	}, nil
}

func (c *Client) GetPooledTransactions(ctx context.Context, hashes []common.Hash) (*PooledTransactions, error) {
	//nolint:gosec // not a security issue
	requestID := uint64(rand.Uint32())<<32 + uint64(rand.Uint32())

	defer func() {
		if c.pooledTransactionsMap[requestID] == nil {
			close(c.pooledTransactionsMap[requestID])
			delete(c.pooledTransactionsMap, requestID)
		}
	}()

	c.pooledTransactionsMap[requestID] = make(chan *PooledTransactions)

	if err := c.sendGetPooledTransactions(ctx, &GetPooledTransactions{
		RequestId:                   requestID,
		GetPooledTransactionsPacket: hashes,
	}); err != nil {
		return nil, err
	}

	select {
	case res := <-c.pooledTransactionsMap[requestID]:
		return res, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout")
	}
}

func (c *Client) Start(ctx context.Context) error {
	c.log.Info("starting execution mimicry client")
	c.peer = p2p.NewPeer(c.nodeRecord.ID(), "xatu", []p2p.Cap{
		{
			Name:    "eth",
			Version: ETHProtocolVersion,
		},
	})

	_, msgPipe := p2p.MsgPipe()
	c.msgPipe = msgPipe

	c.ethPeer = eth.NewPeer(67, c.peer, c.msgPipe, nil)

	address := c.nodeRecord.IP().String() + ":" + strconv.Itoa(c.nodeRecord.TCP())
	c.log.WithField("address", address).Debug("dialing peer")

	var err error

	c.conn, err = net.Dial("tcp", address)
	if err != nil {
		c.log.WithField("address", address).WithError(err).Error("error dialing")

		return err
	}

	c.rlpxConn = rlpx.NewConn(c.conn, c.nodeRecord.Pubkey())
	c.privateKey, _ = crypto.GenerateKey()

	peerPublicKey, err := c.rlpxConn.Handshake(c.privateKey)
	if err != nil {
		c.log.WithField("address", address).WithError(err).Error("error handshaking")

		return err
	}

	c.log.WithFields(logrus.Fields{
		"peer_key": peerPublicKey,
		"address":  address,
	}).Debug("successfully handshaked")

	go func() {
		c.startSession(ctx)
	}()

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.rlpxConn != nil {
		if err := c.rlpxConn.Close(); err != nil {
			c.log.WithError(err).Error("error closing rlpx connection")
		}
	}

	return nil
}

func (c *Client) handleSessionError(ctx context.Context, err error) {
	c.publishDisconnect(ctx, nil)
	c.log.WithError(err).Error("error handling session")
}

func (c *Client) startSession(ctx context.Context) {
	for {
		code, data, _, err := c.rlpxConn.Read()
		if err != nil {
			c.handleSessionError(ctx, fmt.Errorf("error reading rlpx connection: %w", err))
			return
		}

		switch int(code) {
		case HelloCode:
			c.log.WithField("code", code).Debug("received Hello")

			hello, err := c.handleHello(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			c.publishHello(ctx, hello)

			if err := c.sendHello(ctx, hello.MaxETHProtocolVersion()); err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			// always enable snappy to avoid jank
			c.rlpxConn.SetSnappy(true)
		case DisconnectCode:
			c.log.WithField("code", code).Debug("received Disconnect")

			disconnect := c.handleDisconnect(ctx, data)

			c.publishDisconnect(ctx, disconnect)

			return
		case PingCode:
			c.log.WithField("code", code).Debug("received Ping")

			if err := c.sendPong(ctx); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case StatusCode:
			c.log.WithField("code", code).Debug("received Status")

			status, err := c.handleStatus(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			c.publishStatus(ctx, status)

			if err := c.sendStatus(ctx, status); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case TransactionsCode:
			c.log.WithField("code", code).Debug("received Transactions")

			txs, err := c.handleTransactions(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			c.publishTransactions(ctx, txs)
		case GetBlockHeadersCode:
			c.log.WithField("code", code).Debug("received GetBlockHeaders")

			blockHeaders, err := c.handleGetBlockHeaders(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			err = c.sendGetBlockHeaders(ctx, blockHeaders)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case BlockHeadersCode:
			c.log.WithField("code", code).Debug("received BlockHeaders")

			blockHeaders, err := c.handleBlockHeaders(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			err = c.sendBlockHeaders(ctx, blockHeaders)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case NewPooledTransactionHashesCode:
			c.log.WithField("code", code).Debug("received NewPooledTransactionHashes")

			hashes, err := c.handleNewPooledTransactionHashes(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			c.publishNewPooledTransactionHashes(ctx, hashes)
		case PooledTransactionsCode:
			c.log.WithField("code", code).Debug("received PooledTransactions")

			txs, err := c.handlePooledTransactions(ctx, data)
			if err != nil {
				c.handleSessionError(ctx, err)
				return
			}

			if c.pooledTransactionsMap[txs.ReqID()] != nil {
				c.pooledTransactionsMap[txs.ReqID()] <- txs
			}
		default:
			c.log.WithField("code", code).Debug("received unhandled message code")
		}
	}
}
