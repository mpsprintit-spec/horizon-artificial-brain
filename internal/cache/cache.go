// Package cache defines cache interfaces and an in-memory foundation implementation.
package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("cache: key not found")

type Cache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	Delete(context.Context, string) error
	Close() error
}
type item struct {
	value     []byte
	expiresAt time.Time
}
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]item
	now  func() time.Time
}

func NewMemory() *MemoryCache { return &MemoryCache{data: map[string]item{}, now: time.Now} }
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	it, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || (!it.expiresAt.IsZero() && c.now().After(it.expiresAt)) {
		if ok {
			_ = c.Delete(ctx, key)
		}
		return nil, ErrNotFound
	}
	out := append([]byte(nil), it.value...)
	return out, nil
}
func (c *MemoryCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var exp time.Time
	if ttl > 0 {
		exp = c.now().Add(ttl)
	}
	c.mu.Lock()
	c.data[key] = item{value: append([]byte(nil), val...), expiresAt: exp}
	c.mu.Unlock()
	return nil
}
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
	return nil
}
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	c.data = map[string]item{}
	c.mu.Unlock()
	return nil
}
