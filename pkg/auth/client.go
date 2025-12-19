package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type clientNameEntry struct {
	computedName string
	lastUsed     time.Time
}

type clientNameCache struct {
	cache *ttlcache.Cache[string, *clientNameEntry]
	mu    sync.Mutex
}

func newClientNameCache(size uint64) *clientNameCache {
	return &clientNameCache{
		cache: ttlcache.New[string, *clientNameEntry](
			ttlcache.WithTTL[string, *clientNameEntry](60*time.Minute),
			ttlcache.WithCapacity[string, *clientNameEntry](size),
		),
	}
}

func (c *clientNameCache) Start(ctx context.Context) {
	go c.cache.Start()
}

func (c *clientNameCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.cache.Get(key)
	if entry == nil {
		return "", false
	}

	return entry.Value().computedName, true
}

func (c *clientNameCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Set(key, &clientNameEntry{computedName: value, lastUsed: time.Now()}, 60*time.Minute)
}

func ComputeClientName(user, group, clientName, salt string) string {
	hash := sha256.New()
	hash.Write([]byte(clientName))
	hash.Write([]byte(salt))

	hashed := hash.Sum(nil)

	return fmt.Sprintf("%s/%s/hashed-%s", group, user, hex.EncodeToString(hashed)[:8])
}
