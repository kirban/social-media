package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kirban/social-media/internal/config"
	"github.com/redis/go-redis/v9"
)

// deleteByPrefixFn is the Lua function loaded into every master by
// sm-redis-cluster-init (see scripts/udf/udf.lua). It SCANs the local keyspace
// for a glob pattern and UNLINKs what it finds.
const deleteByPrefixFn = "delete_dialog_by_prefix"

// scanChunkSize is the COUNT passed to SCAN inside the Lua function. Keeping it
// modest bounds how long a single SCAN step blocks the node.
const scanChunkSize = 250

type RedisCache struct {
	cc *redis.ClusterClient

	mu       sync.RWMutex
	wg       sync.WaitGroup
	isClosed bool
}

func NewRedisCache(cfg config.CacheConfig) *RedisCache {
	cl := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        cfg.Addrs,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		Username:     cfg.Username,
		Password:     cfg.Password,
	})

	return &RedisCache{cc: cl}
}

// enter admits one operation, or reports that the cache is shutting down. The
// returned func must be called once the operation completes, so that Stop can
// wait for in-flight work before closing the connections.
func (r *RedisCache) enter() (func(), error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.isClosed {
		return nil, ErrClosed
	}
	r.wg.Add(1)
	return r.wg.Done, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	done, err := r.enter()
	if err != nil {
		return nil, err
	}
	defer done()

	b, err := r.cc.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("cache get %q: %w", key, err)
	}
	return b, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	done, err := r.enter()
	if err != nil {
		return err
	}
	defer done()

	if err := r.cc.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set %q: %w", key, err)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	done, err := r.enter()
	if err != nil {
		return err
	}
	defer done()

	if err := r.cc.Unlink(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete %q: %w", key, err)
	}
	return nil
}

// DeleteByPrefix removes every key starting with prefix.
//
// SCAN only ever sees the keyspace of the node it runs on, and a keyless FCALL
// would be routed to one arbitrary shard, so the function is invoked on every
// master in turn. Keys the function UNLINKs are by construction local to the
// node that produced them, which is what keeps the cluster from rejecting the
// undeclared-key access.
func (r *RedisCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	done, err := r.enter()
	if err != nil {
		return err
	}
	defer done()

	pattern := globEscape(prefix) + "*"

	var deleted atomic.Int64
	err = r.cc.ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
		n, err := node.FCall(ctx, deleteByPrefixFn, nil, pattern, scanChunkSize).Int64()
		if err != nil {
			return fmt.Errorf("fcall %s on %s: %w", deleteByPrefixFn, node.Options().Addr, err)
		}
		deleted.Add(n)
		return nil
	})
	if err != nil {
		return fmt.Errorf("cache delete by prefix %q: %w", prefix, err)
	}

	return nil
}

// globEscape neutralises the glob metacharacters Redis' MATCH understands, so
// that a prefix containing them matches literally rather than as a pattern.
func globEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		switch c {
		case '*', '?', '[', ']', '^', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// Stop blocks new operations, waits for in-flight ones to finish (bounded by
// ctx) and then closes the connections.
func (r *RedisCache) Stop(ctx context.Context) error {
	r.mu.Lock()
	if r.isClosed {
		r.mu.Unlock()
		return nil
	}
	r.isClosed = true
	r.mu.Unlock()

	drained := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(drained)
	}()

	select {
	case <-drained:
	case <-ctx.Done():
		return fmt.Errorf("shutdown forced before in-flight requests finished: %w", ctx.Err())
	}

	return r.cc.Close()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.cc.Ping(ctx).Err()
}
