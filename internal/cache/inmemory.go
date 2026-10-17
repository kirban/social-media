package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultCleanupInterval = 5 * time.Minute

type entry struct {
	value     []byte
	expiresAt time.Time
}

// listEntry mirrors a Redis list: an ordered, oldest-first sequence under its
// own TTL. Lists live in a separate map from scalars so that a key cannot be
// both, matching Redis' WRONGTYPE behaviour rather than silently reinterpreting
// the value.
type listEntry struct {
	values    [][]byte
	expiresAt time.Time
}

type InMemoryCache struct {
	mu        sync.RWMutex
	entries   map[string]entry
	lists     map[string]listEntry
	stop      chan struct{}
	closeOnce *sync.Once
}

func NewInMemoryCache() *InMemoryCache {
	c := &InMemoryCache{
		entries:   make(map[string]entry),
		lists:     make(map[string]listEntry),
		stop:      make(chan struct{}),
		closeOnce: &sync.Once{},
	}
	go c.cleanup(defaultCleanupInterval)
	return c
}

func (c *InMemoryCache) Stop(ctx context.Context) error {
	c.closeOnce.Do(func() { close(c.stop) })
	return nil
}

func (c *InMemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, ErrCacheMiss
	}
	return e.value, nil
}

func (c *InMemoryCache) GetList(_ context.Context, key string) ([][]byte, error) {
	c.mu.RLock()
	e, ok := c.lists[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) || len(e.values) == 0 {
		return nil, ErrCacheMiss
	}

	// Copy: callers must not be able to mutate cached state through the slice.
	out := make([][]byte, len(e.values))
	copy(out, e.values)
	return out, nil
}

func (c *InMemoryCache) SetList(ctx context.Context, key string, values [][]byte, ttl time.Duration) error {
	if len(values) == 0 {
		return c.Delete(ctx, key)
	}

	stored := make([][]byte, len(values))
	copy(stored, values)

	c.mu.Lock()
	c.lists[key] = listEntry{values: stored, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
	return nil
}

// AppendIfWarm mirrors the dialog_append Lua function: the whole check-then-
// append is done under the write lock, so it is atomic against concurrent
// senders in the same way the function is atomic on a Redis node.
func (c *InMemoryCache) AppendIfWarm(_ context.Context, key string, value []byte, maxLen int, ttl time.Duration) (bool, error) {
	if maxLen <= 0 {
		return false, fmt.Errorf("cache append %q: maxLen must be positive, got %d", key, maxLen)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.lists[key]
	if !ok || time.Now().After(e.expiresAt) {
		return false, nil
	}

	values := append(e.values, value)
	if len(values) > maxLen {
		values = values[len(values)-maxLen:]
	}
	c.lists[key] = listEntry{values: values, expiresAt: time.Now().Add(ttl)}
	return true, nil
}

func (c *InMemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	c.entries[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	delete(c.lists, key)
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) DeleteByPrefix(_ context.Context, prefix string) error {
	c.mu.Lock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
	for k := range c.lists {
		if strings.HasPrefix(k, prefix) {
			delete(c.lists, k)
		}
	}
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, e := range c.entries {
				if now.After(e.expiresAt) {
					delete(c.entries, k)
				}
			}
			for k, e := range c.lists {
				if now.After(e.expiresAt) {
					delete(c.lists, k)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}
