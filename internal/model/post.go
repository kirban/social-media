package model

import "time"

type Post struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatorID string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
