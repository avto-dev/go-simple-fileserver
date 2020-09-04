package cache

import (
	"io"
	"time"

	"github.com/patrickmn/go-cache"
)

type (
	// Cacher interface describes something, who can read and write data into fast storage (faster than local
	// filesystem).
	Cacher interface {
		// Get an item from the cache. Returns the item or nil, and a bool indicating whether the key was found.
		Get(key string) (*Item, bool)

		// Set an item to the cache, replacing any existing item.
		Set(key string, ttl time.Duration, item *Item)

		// Count returns the number of items in the cache.
		Count() uint32
	}

	// Item is structured cache item.
	Item struct {
		ModifiedTime time.Time
		Content      io.ReadSeeker
	}
)

// InMemoryCache implements Cacher interface and uses memory as a storage.
type InMemoryCache struct {
	engine *cache.Cache
}

// NewInMemoryCache creates cacher implementation, that uses memory as a storage.
func NewInMemoryCache(cleanupInterval time.Duration) *InMemoryCache {
	return &InMemoryCache{
		engine: cache.New(cache.NoExpiration, cleanupInterval),
	}
}

// Get an item from the cache. Returns the item or nil, and a bool indicating whether the key was found.
func (c *InMemoryCache) Get(key string) (*Item, bool) {
	item, ok := c.engine.Get(key)

	if item == nil {
		return nil, ok
	}

	return item.(*Item), ok
}

// Set an item to the cache, replacing any existing item. If the duration is -1, the item never expires.
func (c *InMemoryCache) Set(key string, ttl time.Duration, item *Item) {
	c.engine.Set(key, item, ttl)
}

// Count returns the number of items in the cache. This may include items that have expired, but have not yet been
// cleaned up.
func (c *InMemoryCache) Count() uint32 {
	return uint32(c.engine.ItemCount())
}
