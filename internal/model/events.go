package model

import (
	"fmt"
	"time"
)

// FeedCacheKey is the cache key holding a user's precomputed feed post-ID list.
// Shared by the feed read path (which populates it) and the broker fan-out
// (which invalidates it) so the two never drift.
func FeedCacheKey(userID string) string {
	return fmt.Sprintf("user:%s:feed", userID)
}

// PostCreatedEvent is published to the broker after a post is committed. A
// per-instance consumer fans it out to the author's followers: invalidating
// their cached feed and pushing a FeedPostedMessage over WebSocket.
type PostCreatedEvent struct {
	PostID    string    `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// PostDeletedEvent is published after a post is deleted, so every instance
// invalidates the author's followers' cached feeds.
type PostDeletedEvent struct {
	PostID   string `json:"post_id"`
	AuthorID string `json:"author_id"`
}
