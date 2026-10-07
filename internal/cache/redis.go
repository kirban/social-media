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

// appendFn is the write-through append used by the dialog send path. It is
// loaded into every master by sm-redis-cluster-init; see scripts/udf/udf.lua.
const appendFn = "dialog_append"

// scanChunkSize is the COUNT passed to each SCAN step. Keeping it modest bounds
// how long a single step blocks the node.
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
// SCAN only ever sees the keyspace of the node it runs on, so the scan runs on
// every master in turn. It cannot be pushed into a Lua function: in cluster
// mode a script may only touch keys of a single slot, and a node owns ~5461
// slots, so a SCAN-then-UNLINK function is rejected outright. Each UNLINK is
// therefore issued as its own single-key command, pipelined per batch to keep
// the round trips down.
//
// This walks the whole keyspace of every master, so it is a cold path. Keys
// that are invalidated per request should be deleted by name instead.
func (r *RedisCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	done, err := r.enter()
	if err != nil {
		return err
	}
	defer done()

	pattern := globEscape(prefix) + "*"

	var deleted atomic.Int64
	err = r.cc.ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
		var cursor uint64
		for {
			batch, next, err := node.Scan(ctx, cursor, pattern, scanChunkSize).Result()
			if err != nil {
				return fmt.Errorf("scan %q on %s: %w", pattern, node.Options().Addr, err)
			}

			if len(batch) > 0 {
				pipe := node.Pipeline()
				for _, k := range batch {
					pipe.Unlink(ctx, k)
				}
				cmds, err := pipe.Exec(ctx)
				if err != nil {
					return fmt.Errorf("unlink on %s: %w", node.Options().Addr, err)
				}
				for _, cmd := range cmds {
					if n, err := cmd.(*redis.IntCmd).Result(); err == nil {
						deleted.Add(n)
					}
				}
			}

			cursor = next
			if cursor == 0 {
				return nil
			}
		}
	})
	if err != nil {
		return fmt.Errorf("cache delete by prefix %q: %w", prefix, err)
	}

	return nil
}

// GetList reads the whole cached list. A Redis list with zero elements does
// not exist, so an empty LRANGE result is exactly a cold key — no separate
// EXISTS round trip is needed to tell the two apart.
func (r *RedisCache) GetList(ctx context.Context, key string) ([][]byte, error) {
	done, err := r.enter()
	if err != nil {
		return nil, err
	}
	defer done()

	vals, err := r.cc.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("cache list range %q: %w", key, err)
	}
	if len(vals) == 0 {
		return nil, ErrCacheMiss
	}

	out := make([][]byte, len(vals))
	for i, v := range vals {
		out[i] = []byte(v)
	}
	return out, nil
}

// SetList replaces the list at key. DEL + RPUSH + EXPIRE needs no branching,
// so a MULTI/EXEC pipeline is enough — all three touch one key, hence one
// slot, which is what keeps this valid on a cluster.
func (r *RedisCache) SetList(ctx context.Context, key string, values [][]byte, ttl time.Duration) error {
	done, err := r.enter()
	if err != nil {
		return err
	}
	defer done()

	if len(values) == 0 {
		return r.Delete(ctx, key)
	}

	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}

	tx := r.cc.TxPipeline()
	tx.Del(ctx, key)
	tx.RPush(ctx, key, args...)
	tx.Expire(ctx, key, ttl)
	if _, err := tx.Exec(ctx); err != nil {
		return fmt.Errorf("cache set list %q: %w", key, err)
	}
	return nil
}

// AppendIfWarm delegates to the dialog_append function, which performs the
// EXISTS guard and the RPUSH/LTRIM/EXPIRE atomically on the owning shard.
//
// The key is declared (rather than passed as an argument) so the cluster
// routes the call to the shard that owns it, and so the function is allowed to
// touch it. This is the ordinary case that delete_dialog_by_prefix, with its
// keyless whole-keyspace scan, is not.
func (r *RedisCache) AppendIfWarm(ctx context.Context, key string, value []byte, maxLen int, ttl time.Duration) (bool, error) {
	done, err := r.enter()
	if err != nil {
		return false, err
	}
	defer done()

	if maxLen <= 0 {
		return false, fmt.Errorf("cache append %q: maxLen must be positive, got %d", key, maxLen)
	}

	appended, err := r.cc.FCall(ctx, appendFn, []string{key}, value, maxLen, int64(ttl.Seconds())).Int64()
	if err != nil {
		return false, fmt.Errorf("fcall %s on %q: %w", appendFn, key, err)
	}
	return appended == 1, nil
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
