package model

import "time"

type DialogID = string
type DialogMessageID = string

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
