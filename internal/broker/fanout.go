package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kirban/social-media/internal/cache"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

// FollowerLister resolves the followers who should receive an author's posts.
// Satisfied by *service.FriendsService.
type FollowerLister interface {
	ListFollowers(ctx context.Context, userID string) ([]string, error)
}

// Notifier pushes an opaque payload to every connection a user has open on this
// instance. Satisfied by *websocket.Hub.
type Notifier interface {
	Send(userID string, payload []byte)
}

// FeedFanout applies a post event to this instance's local state: it
// invalidates each follower's cached feed and, for new posts, pushes a
// notification to followers connected here. It holds exactly what the old
// fire-and-forget goroutines in PostsService held.
//
// Every operation is idempotent (cache delete and WS send are safe to repeat),
// so at-least-once / redelivered events cause no harm.
type FeedFanout struct {
	followers FollowerLister
	cache     cache.Cache
	hub       Notifier
	log       *logger.AppLogger
}

func NewFeedFanout(f FollowerLister, c cache.Cache, hub Notifier, log *logger.AppLogger) *FeedFanout {
	return &FeedFanout{followers: f, cache: c, hub: hub, log: log}
}

// HandlePostCreated invalidates followers' cached feeds and notifies the ones
// connected to this instance about the new post.
func (f *FeedFanout) HandlePostCreated(ctx context.Context, e model.PostCreatedEvent) error {
	followers, err := f.followers.ListFollowers(ctx, e.AuthorID)
	if err != nil {
		return fmt.Errorf("post created: list followers of %s: %w", e.AuthorID, err)
	}

	msg, err := json.Marshal(model.FeedPostedMessage{
		PostID:       e.PostID,
		PostText:     e.Text,
		AuthorUserID: e.AuthorID,
	})
	if err != nil {
		return fmt.Errorf("post created: marshal notification: %w", err)
	}

	for _, follower := range followers {
		f.invalidate(ctx, follower)
		f.hub.Send(follower, msg)
	}
	return nil
}

// HandlePostDeleted invalidates followers' cached feeds so the removed post
// drops out on the next read.
func (f *FeedFanout) HandlePostDeleted(ctx context.Context, e model.PostDeletedEvent) error {
	followers, err := f.followers.ListFollowers(ctx, e.AuthorID)
	if err != nil {
		return fmt.Errorf("post deleted: list followers of %s: %w", e.AuthorID, err)
	}

	for _, follower := range followers {
		f.invalidate(ctx, follower)
	}
	return nil
}

func (f *FeedFanout) invalidate(ctx context.Context, followerID string) {
	key := model.FeedCacheKey(followerID)
	if err := f.cache.Delete(ctx, key); err != nil {
		f.log.Error().Err(err).Str("key", key).Msg("fanout: feed cache invalidation failed")
	}
}
