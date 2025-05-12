package persistence

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/huandu/go-sqlbuilder"
)

var nodeRecordConsensusStruct = sqlbuilder.NewStruct(new(node.Consensus)).For(sqlbuilder.PostgreSQL)

func (c *Client) InsertNodeRecordConsensus(ctx context.Context, record *node.Consensus) error {
	return errors.New("not implemented")
}

func (c *Client) ListNodeRecordConsensus(ctx context.Context, networkIds []uint64, forkDigests [][]byte, limit int) ([]*node.Consensus, error) {
	return nil, errors.New("not implemented")
}
