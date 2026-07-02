package cache

import (
	"context"
	"time"
)

const DefaultTTL = 3 * time.Minute

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
}
