package cache

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryCache_CRD(t *testing.T) {
	cache := NewInMemoryCache(time.Millisecond * 3)

	item, exists := cache.Get("foo")
	assert.Nil(t, item)
	assert.False(t, exists)
	assert.Equal(t, uint32(0), cache.Count())

	now := time.Now()
	data := []byte("abc")
	ttl := time.Millisecond * 7

	cache.Set("foo", ttl, &Item{
		ModifiedTime: now,
		Content:      bytes.NewReader(data),
	})

	item, exists = cache.Get("foo")
	assert.Equal(t, uint32(1), cache.Count())
	assert.Equal(t, now, item.ModifiedTime)

	buf := make([]byte, len(data))
	_, _ = item.Content.Read(buf)

	assert.Equal(t, []byte("abc"), buf)
	assert.True(t, exists)

	time.Sleep(ttl)

	item, exists = cache.Get("foo")
	assert.Nil(t, item)
	assert.False(t, exists)

	time.Sleep(ttl)

	assert.Equal(t, uint32(0), cache.Count())
}
