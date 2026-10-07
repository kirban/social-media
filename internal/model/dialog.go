package model

import (
	"fmt"
	"time"
)

type DialogID = string
type DialogMessageID = string

// DialogMessagesCacheKey is the cache key holding the newest page of a
// dialog's messages, oldest first.
//
// The {dialog:<id>} hash tag puts every key belonging to one dialog in the same
// cluster slot. That is what lets a Lua function touch several of them in one
// call — without it, any multi-key function would be rejected with CROSSSLOT.
func DialogMessagesCacheKey(dialogID DialogID) string {
	return fmt.Sprintf("{dialog:%s}:msgs", dialogID)
}

// DialogCacheKeyPrefix matches every cache key for one dialog, for bulk
// invalidation.
func DialogCacheKeyPrefix(dialogID DialogID) string {
	return fmt.Sprintf("{dialog:%s}:", dialogID)
}

type DialogMessage struct {
	ID        DialogMessageID `json:"-"`
	From      UserID          `json:"from"`
	To        UserID          `json:"to"`
	DialogID  DialogID        `json:"dialog"`
	Text      string          `json:"text"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type DialogUser struct {
	ID     string   `json:"id"`
	User   UserID   `json:"user_id"`
	Dialog DialogID `json:"dialog_id"`
}

type Dialog struct {
	ID        DialogID  `json:"id"`
	Users     [2]UserID `json:"users"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateDialogDTO struct {
	Users [2]UserID
}

// DialogSummary is one entry in a user's conversation list: the other
// participant and the dialog they share. It deliberately carries no last-message
// preview — messages are sharded by dialog_id, so a preview would fan the query
// out across shards, while participants live in a reference table.
type DialogSummary struct {
	DialogID DialogID `json:"dialog_id"`
	UserID   UserID   `json:"user_id"`
}
