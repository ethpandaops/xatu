package store

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/server/store/memory"
	redisCluster "github.com/ethpandaops/xatu/pkg/server/store/redis/cluster"
	redisServer "github.com/ethpandaops/xatu/pkg/server/store/redis/server"
)

type Type string

const (
	TypeUnknown      Type = "unknown"
	TypeMemory       Type = memory.Type
	TypeRedisServer  Type = redisServer.Type
	TypeRedisCluster Type = redisCluster.Type
)

type Cache interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
	Get(ctx context.Context, key string) (*string, error)
	GetOrSet(ctx context.Context, key, value string, ttl time.Duration) (storedValue *string, retrieved bool, err error)
	GetAndDelete(ctx context.Context, key string) (deletedValue *string, exists bool, err error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
