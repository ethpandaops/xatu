package node

import (
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type s256raw []byte

func (s256raw) ENRKey() string { return "secp256k1" }

type eth2 []byte

func (eth2) ENRKey() string { return "eth2" }

type attnets []byte

func (attnets) ENRKey() string { return "attnets" }

type syncnets []byte

func (syncnets) ENRKey() string { return "syncnets" }

type cgc []byte

func (cgc) ENRKey() string { return "cgc" }

func Parse(record string) (*Record, error) {
	n, err := enode.Parse(enode.ValidSchemes, record)
	if err != nil {
		return nil, fmt.Errorf("failed to parse enr: %w", err)
	}

	return &Record{
		Enr:       record,
		Signature: parseSignature(n),
		Seq:       parseSeq(n),
		ID:        parseID(n),
		Secp256k1: parseSecp256k1(n),
		IP4:       parseIP4(n),
		IP6:       parseIP6(n),
		TCP4:      parseTCP4(n),
		TCP6:      parseTCP6(n),
		UDP4:      parseUDP4(n),
		UDP6:      parseUDP6(n),
		ETH2:      parseETH2(n),
		Attnets:   parseAttnets(n),
		Syncnets:  parseSyncnets(n),
		CGC:       parseCGC(n),
		NodeID:    parseNodeID(n),
		PeerID:    parsePeerID(n),
	}, nil
}

func parseSignature(node *enode.Node) *[]byte {
	signature := node.Record().Signature()

	return &signature
}

func parseSeq(node *enode.Node) *uint64 {
	seq := node.Seq()

	return &seq
}

func parseID(node *enode.Node) *string {
	id := node.Record().IdentityScheme()

	return &id
}

func parseSecp256k1(node *enode.Node) *[]byte {
	field := s256raw{}
	err := node.Record().Load(&field)

	if err != nil {
		return nil
	}

	f := []byte(field)

	return &f
}

func parseIP4(node *enode.Node) *string {
	ip := node.IP()
	if ip.IsUnspecified() || ip.String() == "<nil>" {
		return nil
	}

	f := ip.String()

	return &f
}

func parseIP6(node *enode.Node) *string {
	var field enr.IPv6

	err := node.Record().Load(&field)
	if err == nil {
		return nil
	}

	ip := net.IP(field)
	if ip.IsUnspecified() || ip.String() == "<nil>" {
		return nil
	}

	f := ip.String()

	return &f
}

func parseTCP4(node *enode.Node) *uint32 {
	field := uint32(node.TCP())

	if field == 0 {
		return nil
	}

	return &field
}

func parseTCP6(node *enode.Node) *uint32 {
	var field enr.TCP6

	err := node.Record().Load(&field)
	if err == nil {
		return nil
	}

	f := uint32(field)

	if f == 0 {
		return nil
	}

	return &f
}

func parseUDP4(node *enode.Node) *uint32 {
	field := uint32(node.UDP())

	if field == 0 {
		return nil
	}

	return &field
}

func parseUDP6(node *enode.Node) *uint32 {
	var field enr.UDP6

	err := node.Record().Load(&field)
	if err == nil {
		return nil
	}

	f := uint32(field)

	if f == 0 {
		return nil
	}

	return &f
}

func parseETH2(node *enode.Node) *[]byte {
	field := eth2{}

	err := node.Record().Load(&field)

	if err != nil {
		return nil
	}

	f := []byte(field)

	return &f
}

func parseAttnets(node *enode.Node) *[]byte {
	field := attnets{}

	err := node.Record().Load(&field)

	if err != nil {
		return nil
	}

	f := []byte(field)

	return &f
}

func parseSyncnets(node *enode.Node) *[]byte {
	field := syncnets{}

	err := node.Record().Load(&field)

	if err != nil {
		return nil
	}

	f := []byte(field)

	return &f
}

func parseCGC(node *enode.Node) *[]byte {
	field := cgc{}

	err := node.Record().Load(&field)

	if err != nil {
		return nil
	}

	f := []byte(field)

	return &f
}

func parseNodeID(node *enode.Node) *string {
	f := node.ID().String()

	return &f
}

func parsePeerID(node *enode.Node) *string {
	key := node.Pubkey()
	pubkey := crypto.FromECDSAPub(key)

	sPubkey, err := libp2pcrypto.UnmarshalSecp256k1PublicKey(pubkey)
	if err != nil {
		return nil
	}

	peerID, err := peer.IDFromPublicKey(sPubkey)
	if err != nil {
		return nil
	}

	f := peerID.String()

	return &f
}
