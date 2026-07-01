package repository

import (
	"context"

	"github.com/kirban/social-media/internal/db"
)

type FriendsRepository struct {
	cluster *db.Cluster
}

func NewFriendsRepository(cluster *db.Cluster) *FriendsRepository {
	return &FriendsRepository{cluster: cluster}
}

func (r *FriendsRepository) AddFriend(ctx context.Context, userID, friendID string) error {
	_, err := r.cluster.Master().ExecContext(ctx, `
		INSERT INTO friends (user_id, friend_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, friendID)
	return err
}
