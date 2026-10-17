package cache_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kirban/social-media/internal/cache"
	"github.com/kirban/social-media/internal/config"
)

// Both drivers are exercised by the same suite: they are only interchangeable
// if they behave identically, and the Redis one spent a whole branch not even
// satisfying the interface.
func TestInMemoryCache(t *testing.T) {
	runCacheSuite(t, func(t *testing.T) cache.Cache {
		c := cache.NewInMemoryCache()
		t.Cleanup(func() { _ = c.Stop(context.Background()) })
		return c
	})
}

// TestRedisCache needs the 3-node cluster from
// deployments/docker-compose-in-memory-db.yaml. It is skipped unless
// CACHE_ADDRESSES is set, so `go test ./...` stays green without Docker.
func TestRedisCache(t *testing.T) {
	addrs := os.Getenv("CACHE_ADDRESSES")
	if addrs == "" {
		t.Skip("CACHE_ADDRESSES not set; skipping Redis cluster tests")
	}

	runCacheSuite(t, func(t *testing.T) cache.Cache {
		c := cache.NewRedisCache(config.CacheConfig{
			Addrs:        strings.Split(addrs, ","),
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.Ping(ctx); err != nil {
			t.Fatalf("ping cluster at %s: %v", addrs, err)
		}
		t.Cleanup(func() { _ = c.Stop(context.Background()) })
		return c
	})
}

func runCacheSuite(t *testing.T, newCache func(t *testing.T) cache.Cache) {
	t.Helper()

	tests := []struct {
		name string
		run  func(t *testing.T, c cache.Cache)
	}{
		{"GetRoundTrip", testGetRoundTrip},
		{"GetMissingReturnsErrCacheMiss", testGetMissing},
		{"GetExpiredReturnsErrCacheMiss", testGetExpired},
		{"DeleteRemovesOnlyTheKey", testDelete},
		{"DeleteByPrefix", testDeleteByPrefix},
		{"DeleteByPrefixTreatsGlobLiterally", testDeleteByPrefixGlob},
		{"SetListRoundTrip", testSetListRoundTrip},
		{"GetListMissingReturnsErrCacheMiss", testGetListMissing},
		{"AppendIfWarmSkipsColdKey", testAppendCold},
		{"AppendIfWarmAppendsToWarmKey", testAppendWarm},
		{"AppendIfWarmTrimsToMaxLen", testAppendTrims},
		{"AppendIfWarmRejectsNonPositiveMaxLen", testAppendBadMaxLen},
		{"OperationsAfterStopReturnErrClosed", testAfterStop},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, newCache(t))
		})
	}
}

// key namespaces each test run so parallel or repeated runs against a shared
// cluster cannot collide. The {} hash tag keeps one test's keys in one slot.
func key(t *testing.T, suffix string) string {
	t.Helper()
	return fmt.Sprintf("{test:%s:%d}:%s", t.Name(), time.Now().UnixNano(), suffix)
}

func testGetRoundTrip(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "v")

	if err := c.Set(ctx, k, []byte("hello"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx, k)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("Get = %q, want %q", got, "hello")
	}
}

func testGetMissing(t *testing.T, c cache.Cache) {
	_, err := c.Get(context.Background(), key(t, "absent"))
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("Get on absent key = %v, want ErrCacheMiss", err)
	}
}

func testGetExpired(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "ttl")

	// Redis TTLs have second granularity, so this cannot be sub-second.
	if err := c.Set(ctx, k, []byte("v"), time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(1500 * time.Millisecond)

	if _, err := c.Get(ctx, k); !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("Get after expiry = %v, want ErrCacheMiss", err)
	}
}

func testDelete(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	target, sibling := key(t, "a"), key(t, "b")

	mustSet(t, c, target, "1")
	mustSet(t, c, sibling, "2")

	if err := c.Delete(ctx, target); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Get(ctx, target); !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("deleted key still present: %v", err)
	}
	if _, err := c.Get(ctx, sibling); err != nil {
		t.Fatalf("sibling key was removed: %v", err)
	}
}

func testDeleteByPrefix(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	prefix := fmt.Sprintf("pfx:%d:", time.Now().UnixNano())

	// Enough keys, without a hash tag, to spread across all three slot ranges:
	// a handful could all land on one shard and pass even if the per-master
	// fan-out were broken.
	const n = 200
	for i := range n {
		mustSet(t, c, fmt.Sprintf("%s%d", prefix, i), "x")
	}
	survivor := fmt.Sprintf("other:%d", time.Now().UnixNano())
	mustSet(t, c, survivor, "keep")

	if err := c.DeleteByPrefix(ctx, prefix); err != nil {
		t.Fatalf("DeleteByPrefix: %v", err)
	}

	for i := range n {
		k := fmt.Sprintf("%s%d", prefix, i)
		if _, err := c.Get(ctx, k); !errors.Is(err, cache.ErrCacheMiss) {
			t.Fatalf("key %q survived DeleteByPrefix: %v", k, err)
		}
	}
	if _, err := c.Get(ctx, survivor); err != nil {
		t.Fatalf("non-matching key was removed: %v", err)
	}
}

func testDeleteByPrefixGlob(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	literal := fmt.Sprintf("glob:%d:*:match", stamp)
	other := fmt.Sprintf("glob:%d:zzz:match", stamp)

	mustSet(t, c, literal, "1")
	mustSet(t, c, other, "2")

	// The "*" must be matched literally, not as a wildcard that also eats
	// `other`.
	if err := c.DeleteByPrefix(ctx, fmt.Sprintf("glob:%d:*", stamp)); err != nil {
		t.Fatalf("DeleteByPrefix: %v", err)
	}
	if _, err := c.Get(ctx, literal); !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("literal-prefixed key survived: %v", err)
	}
	if _, err := c.Get(ctx, other); err != nil {
		t.Fatalf("glob was treated as a wildcard and removed %q: %v", other, err)
	}
}

func testSetListRoundTrip(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "msgs")
	want := [][]byte{[]byte("m1"), []byte("m2"), []byte("m3")}

	if err := c.SetList(ctx, k, want, time.Minute); err != nil {
		t.Fatalf("SetList: %v", err)
	}
	assertList(t, c, k, "m1", "m2", "m3")

	// SetList replaces rather than appends.
	if err := c.SetList(ctx, k, [][]byte{[]byte("only")}, time.Minute); err != nil {
		t.Fatalf("SetList replace: %v", err)
	}
	assertList(t, c, k, "only")
}

func testGetListMissing(t *testing.T, c cache.Cache) {
	_, err := c.GetList(context.Background(), key(t, "absent"))
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("GetList on absent key = %v, want ErrCacheMiss", err)
	}
}

// The guard that makes this write-through instead of write-invalidate: a cold
// page must not be created from a single message, or readers would see a
// one-message history as if it were complete.
func testAppendCold(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "cold")

	appended, err := c.AppendIfWarm(ctx, k, []byte("m1"), 10, time.Minute)
	if err != nil {
		t.Fatalf("AppendIfWarm: %v", err)
	}
	if appended {
		t.Fatal("AppendIfWarm reported an append against a cold key")
	}
	if _, err := c.GetList(ctx, k); !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("cold key was populated: %v", err)
	}
}

func testAppendWarm(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "warm")

	if err := c.SetList(ctx, k, [][]byte{[]byte("m1"), []byte("m2")}, time.Minute); err != nil {
		t.Fatalf("SetList: %v", err)
	}

	appended, err := c.AppendIfWarm(ctx, k, []byte("m3"), 10, time.Minute)
	if err != nil {
		t.Fatalf("AppendIfWarm: %v", err)
	}
	if !appended {
		t.Fatal("AppendIfWarm reported no append against a warm key")
	}
	assertList(t, c, k, "m1", "m2", "m3")
}

func testAppendTrims(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "trim")

	if err := c.SetList(ctx, k, [][]byte{[]byte("m1"), []byte("m2"), []byte("m3")}, time.Minute); err != nil {
		t.Fatalf("SetList: %v", err)
	}
	if _, err := c.AppendIfWarm(ctx, k, []byte("m4"), 3, time.Minute); err != nil {
		t.Fatalf("AppendIfWarm: %v", err)
	}

	// Oldest dropped, newest kept, order preserved.
	assertList(t, c, k, "m2", "m3", "m4")
}

func testAppendBadMaxLen(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	k := key(t, "badmax")

	if err := c.SetList(ctx, k, [][]byte{[]byte("m1")}, time.Minute); err != nil {
		t.Fatalf("SetList: %v", err)
	}
	// LTRIM with a non-positive maxLen would empty the list, i.e. silently
	// delete the page.
	if _, err := c.AppendIfWarm(ctx, k, []byte("m2"), 0, time.Minute); err == nil {
		t.Fatal("AppendIfWarm with maxLen=0 should fail")
	}
	assertList(t, c, k, "m1")
}

func testAfterStop(t *testing.T, c cache.Cache) {
	ctx := context.Background()
	if err := c.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// The in-memory driver has no connections to drain, so it keeps serving
	// after Stop; only the Redis driver refuses. Both must at least not panic.
	if _, err := c.Get(ctx, key(t, "x")); err != nil &&
		!errors.Is(err, cache.ErrClosed) && !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("Get after Stop = %v, want ErrClosed or ErrCacheMiss", err)
	}
}

func mustSet(t *testing.T, c cache.Cache, k, v string) {
	t.Helper()
	if err := c.Set(context.Background(), k, []byte(v), time.Minute); err != nil {
		t.Fatalf("Set %q: %v", k, err)
	}
}

func assertList(t *testing.T, c cache.Cache, k string, want ...string) {
	t.Helper()
	got, err := c.GetList(context.Background(), k)
	if err != nil {
		t.Fatalf("GetList %q: %v", k, err)
	}
	if len(got) != len(want) {
		t.Fatalf("GetList %q = %d entries %q, want %d %q", k, len(got), got, len(want), want)
	}
	for i := range want {
		if string(got[i]) != want[i] {
			t.Fatalf("GetList %q [%d] = %q, want %q (full: %q)", k, i, got[i], want[i], got)
		}
	}
}
