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

	ethCapVersion uint
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
	c.log.Debug("starting execution mimicry client")
	c.peer = p2p.NewPeer(c.nodeRecord.ID(), "xatu", SupportedEthCaps())

	_, msgPipe := p2p.MsgPipe()
	c.msgPipe = msgPipe

	c.ethPeer = eth.NewPeer(maxETHProtocolVersion, c.peer, c.msgPipe, nil)

	address := c.nodeRecord.IP().String() + ":" + strconv.Itoa(c.nodeRecord.TCP())
	c.log.WithField("address", address).Debug("dialing peer")

	var err error

	// set a deadline for dialing
	d := net.Dialer{Deadline: time.Now().Add(5 * time.Second)}

	c.conn, err = d.Dial("tcp", address)
	if err != nil {
		return err
	}

	c.rlpxConn = rlpx.NewConn(c.conn, c.nodeRecord.Pubkey())

	// update deadline for handshake
	err = c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return err
	}

	c.privateKey, _ = crypto.GenerateKey()

	peerPublicKey, err := c.rlpxConn.Handshake(c.privateKey)
	if err != nil {
		return err
	}

	// clear deadline
	err = c.conn.SetDeadline(time.Time{})
	if err != nil {
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
	if c.peer != nil {
		c.peer.Disconnect(p2p.DiscQuitting)
	}

	if c.ethPeer != nil {
		c.ethPeer.Close()
	}

	if c.rlpxConn != nil {
		if err := c.rlpxConn.Close(); err != nil {
			c.log.WithError(err).Error("error closing rlpx connection")
		}
	}

	if c.msgPipe != nil {
		if err := c.msgPipe.Close(); err != nil {
			c.log.WithError(err).Error("error closing msg pipe")
		}
	}

	return nil
}

func (c *Client) handleSessionError(ctx context.Context, err error) {
	c.log.WithError(err).Debug("error handling session")
	c.disconnect(ctx, nil)
}

func (c *Client) disconnect(ctx context.Context, reason *Disconnect) {
	c.log.Debug("disconnecting from peer")
	c.publishDisconnect(ctx, reason)
}

func (c *Client) startSession(ctx context.Context) {
	if err := c.sendHello(ctx); err != nil {
		c.handleSessionError(ctx, err)
		return
	}

	for {
		// set read deadline
		err := c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			c.handleSessionError(ctx, fmt.Errorf("error setting rlpx read deadline: %w", err))
			return
		}

		code, data, _, err := c.rlpxConn.Read()
		if err != nil {
			c.handleSessionError(ctx, fmt.Errorf("error reading rlpx connection: %w", err))
			return
		}

		switch int(code) {
		case HelloCode:
			if err := c.handleHello(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case DisconnectCode:
			c.handleDisconnect(ctx, code, data)
			return
		case PingCode:
			if err := c.handlePing(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case StatusCode:
			if err := c.handleStatus(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case TransactionsCode:
			if err := c.handleTransactions(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case GetBlockHeadersCode:
			if err := c.handleGetBlockHeaders(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case BlockHeadersCode:
			if err := c.handleBlockHeaders(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case GetBlockBodiesCode:
			if err := c.handleGetBlockBodies(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case NewPooledTransactionHashesCode:
			if err := c.handleNewPooledTransactionHashes(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case PooledTransactionsCode:
			if err := c.handlePooledTransactions(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		case GetReceiptsCode:
			if err := c.handleGetReceipts(ctx, code, data); err != nil {
				c.handleSessionError(ctx, err)
				return
			}
		default:
			c.log.WithField("code", code).Debug("received unhandled message code")
		}
	}
}
