package discovery

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
)

func (d *Discovery) handleNode(ctx context.Context, node *enode.Node) error {
	d.log.Debug("Node received")

	enr := node.String()

	item, retrieved := d.duplicateCache.Node.GetOrSet(enr, time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		d.log.WithFields(logrus.Fields{
			"enr":                   enr,
			"time_since_first_item": time.Since(item.Value()),
		}).Debug("Duplicate node received")
		// TODO(savid): add metrics
		return nil
	}

	return d.handleNewNodeRecord(ctx, enr)
}
