package p2p

import (
	"context"
	"crypto/ecdsa"
	"net"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/sirupsen/logrus"
)

type DiscV5 struct {
	config *Config

	log logrus.FieldLogger

	listener *discover.UDPv5

	handlerFunc func(context.Context, *enode.Node) error
}

func NewDiscV5(ctx context.Context, config *Config, log logrus.FieldLogger, handlerFunc func(context.Context, *enode.Node) error) *DiscV5 {
	return &DiscV5{
		log:         log.WithField("module", "discovery/p2p/discV5"),
		config:      config,
		handlerFunc: handlerFunc,
	}
}

func (d *DiscV5) Start(ctx context.Context) error {
	privKey, err := gcrypto.GenerateKey()
	if err != nil {
		return err
	}

	listener, err := d.startDiscovery(ctx, privKey)
	if err != nil {
		return err
	}

	d.listener = listener

	go d.listenForNewNodes(ctx)

	return nil
}

func (d *DiscV5) listenForNewNodes(ctx context.Context) {
	iterator := d.listener.RandomNodes()
	iterator = enode.Filter(iterator, d.filterPeer)

	defer iterator.Close()

	for {
		exists := iterator.Next()
		if !exists {
			break
		}

		node := iterator.Node()

		err := d.handlerFunc(ctx, node)
		if err != nil {
			d.log.WithError(err).Error("Could not handle new node")
		}
	}
}

func (d *DiscV5) createListener(
	ctx context.Context,
	privKey *ecdsa.PrivateKey,
) (*discover.UDPv5, error) {
	var bindIP net.IP

	ipAddr := net.IPv4zero

	bindIP = ipAddr
	udpAddr := &net.UDPAddr{
		IP:   bindIP,
		Port: int(0),
	}
	conn, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		return nil, err
	}

	localNode, err := d.createLocalNode(
		ctx,
		privKey,
		ipAddr,
		int(0),
		int(0),
	)

	if err != nil {
		return nil, err
	}

	dv5Cfg := discover.Config{
		PrivateKey: privKey,
	}

	dv5Cfg.Bootnodes = []*enode.Node{}

	discv5BootStrapAddr := d.config.BootNodes
	for _, addr := range discv5BootStrapAddr {
		bootNode, parseErr := enode.Parse(enode.ValidSchemes, addr)
		if parseErr != nil {
			return nil, err
		}

		dv5Cfg.Bootnodes = append(dv5Cfg.Bootnodes, bootNode)
	}

	listener, err := discover.ListenV5(conn, localNode, dv5Cfg)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (d *DiscV5) createLocalNode(
	ctx context.Context,
	privKey *ecdsa.PrivateKey,
	ipAddr net.IP,
	udpPort, tcpPort int,
) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}

	localNode := enode.NewLocalNode(db, privKey)

	ipEntry := enr.IP(ipAddr)
	udpEntry := enr.UDP(udpPort)
	tcpEntry := enr.TCP(tcpPort)

	localNode.Set(ipEntry)
	localNode.Set(udpEntry)
	localNode.Set(tcpEntry)
	localNode.SetFallbackIP(ipAddr)
	localNode.SetFallbackUDP(udpPort)

	return localNode, nil
}

func (d *DiscV5) startDiscovery(
	ctx context.Context,
	privKey *ecdsa.PrivateKey,
) (*discover.UDPv5, error) {
	listener, err := d.createListener(ctx, privKey)
	if err != nil {
		return nil, err
	}

	record := listener.Self()
	d.log.WithField("ENR", record.String()).Info("Started discovery v5")

	return listener, nil
}

func (d *DiscV5) filterPeer(node *enode.Node) bool {
	// Ignore nil node entries passed in.
	if node == nil {
		return false
	}

	// ignore nodes with no ip address stored.
	if node.IP() == nil {
		return false
	}

	// do not dial nodes with their tcp ports not set
	if err := node.Record().Load(enr.WithEntry("tcp", new(enr.TCP))); err != nil {
		if !enr.IsNotFound(err) {
			d.log.WithError(err).Debug("Could not retrieve tcp port")
		}

		return false
	}

	return true
}
