package node

import (
	coreenr "github.com/ethpandaops/ethcore/pkg/ethereum/node/enr"
)

func Parse(record string) (*Record, error) {
	coreEnr, err := coreenr.Parse(record)
	if err != nil {
		return nil, err
	}

	return &Record{
		Enr:            coreEnr.Enr,
		Signature:      coreEnr.Signature,
		Seq:            coreEnr.Seq,
		ID:             coreEnr.ID,
		Secp256k1:      coreEnr.Secp256k1,
		IP4:            coreEnr.IP4,
		IP6:            coreEnr.IP6,
		TCP4:           coreEnr.TCP4,
		TCP6:           coreEnr.TCP6,
		UDP4:           coreEnr.UDP4,
		UDP6:           coreEnr.UDP6,
		ETH2:           coreEnr.ETH2,
		Attnets:        coreEnr.Attnets,
		Syncnets:       coreEnr.Syncnets,
		CGC:            coreEnr.CGC,
		NextForkDigest: coreEnr.NFD,
		NodeID:         coreEnr.NodeID,
		PeerID:         coreEnr.PeerID,
	}, nil
}
