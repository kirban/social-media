package cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

const defaultCleanupInterval = 5 * time.Minute

type entry struct {
	value     []byte
	expiresAt time.Time
}

type InMemoryCache struct {
	mu        sync.RWMutex
	entries   map[string]entry
	stop      chan struct{}
	closeOnce *sync.Once
}

func NewInMemoryCache() *InMemoryCache {
	c := &InMemoryCache{
		entries:   make(map[string]entry),
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
		return nil, nil
	}
	return e.value, nil
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
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}
