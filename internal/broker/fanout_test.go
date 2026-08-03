package broker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

func testLogger() *logger.AppLogger {
	return &logger.AppLogger{Logger: zerolog.Nop()}
}

type fakeFollowers struct {
	list []string
	err  error
}

func (f fakeFollowers) ListFollowers(context.Context, string) ([]string, error) {
	return f.list, f.err
}

// fakeCache records Delete calls; other methods satisfy the cache.Cache
// interface but are unused by the fan-out.
type fakeCache struct {
	mu      sync.Mutex
	deleted []string
}

func (c *fakeCache) Get(context.Context, string) ([]byte, bool, error)        { return nil, false, nil }
func (c *fakeCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (c *fakeCache) DeleteByPrefix(context.Context, string) error             { return nil }
func (c *fakeCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted = append(c.deleted, key)
	return nil
}

type fakeHub struct {
	mu   sync.Mutex
	sent map[string]int
}

func newFakeHub() *fakeHub { return &fakeHub{sent: map[string]int{}} }
func (h *fakeHub) Send(userID string, _ []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sent[userID]++
}

func TestHandlePostCreated_InvalidatesAndNotifiesEveryFollower(t *testing.T) {
	followers := []string{"u1", "u2", "u3"}
	c := &fakeCache{}
	hub := newFakeHub()
	f := NewFeedFanout(fakeFollowers{list: followers}, c, hub, testLogger())

	err := f.HandlePostCreated(context.Background(), model.PostCreatedEvent{
		PostID:   "p1",
		AuthorID: "author",
		Text:     "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, u := range followers {
		key := model.FeedCacheKey(u)
		if !contains(c.deleted, key) {
			t.Errorf("expected cache invalidation for %s (key %s)", u, key)
		}
		if hub.sent[u] != 1 {
			t.Errorf("expected 1 WS notification for %s, got %d", u, hub.sent[u])
		}
	}
}

func TestHandlePostDeleted_InvalidatesButDoesNotNotify(t *testing.T) {
	followers := []string{"u1", "u2"}
	c := &fakeCache{}
	hub := newFakeHub()
	f := NewFeedFanout(fakeFollowers{list: followers}, c, hub, testLogger())

	if err := f.HandlePostDeleted(context.Background(), model.PostDeletedEvent{
		PostID:   "p1",
		AuthorID: "author",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.deleted) != len(followers) {
		t.Errorf("expected %d invalidations, got %d", len(followers), len(c.deleted))
	}
	if len(hub.sent) != 0 {
		t.Errorf("post.deleted must not push WS notifications, got %v", hub.sent)
	}
}

func TestHandlePostCreated_ListFollowersError(t *testing.T) {
	f := NewFeedFanout(fakeFollowers{err: errors.New("db down")}, &fakeCache{}, newFakeHub(), testLogger())

	err := f.HandlePostCreated(context.Background(), model.PostCreatedEvent{AuthorID: "author"})
	if err == nil {
		t.Fatal("expected error when ListFollowers fails")
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
