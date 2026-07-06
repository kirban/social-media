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
