package cache

import (
	"context"
	"errors"
	"time"
)

const DefaultTTL = 3 * time.Minute

var (
	// ErrCacheMiss is returned by Get when the key is absent or expired.
	ErrCacheMiss = errors.New("cache: miss")
	// ErrClosed is returned when an operation is attempted on a cache that is
	// shutting down.
	ErrClosed = errors.New("cache: closed")
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error

	// GetList returns the entries of the list at key, oldest first. It returns
	// ErrCacheMiss when the key is cold: an existing list is never empty, so
	// "no entries" and "no key" are the same state.
	GetList(ctx context.Context, key string) ([][]byte, error)
	// SetList replaces the list at key with values and sets its TTL. An empty
	// values slice deletes the key.
	SetList(ctx context.Context, key string, values [][]byte, ttl time.Duration) error
	// AppendIfWarm appends value to the list at key, trims it to the newest
	// maxLen entries and refreshes its TTL — but only if the list is already
	// warm. It reports whether the append happened.
	//
	// The guard is what makes this write-through rather than write-invalidate:
	// a cold key is left cold (populating it from a single message would serve
	// a truncated history), while a warm key stays warm instead of being
	// dropped on every write.
	AppendIfWarm(ctx context.Context, key string, value []byte, maxLen int, ttl time.Duration) (bool, error)

	Stop(ctx context.Context) error
}

// Implementations must stay interchangeable: RedisCache went unnoticed as a
// non-implementor for a whole branch because nothing forced the match.
var (
	_ Cache = (*InMemoryCache)(nil)
	_ Cache = (*RedisCache)(nil)
)
