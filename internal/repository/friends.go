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

func (r *FriendsRepository) DeleteFriend(ctx context.Context, userID, friendID string) error {
	_, err := r.cluster.Master().ExecContext(ctx, `
		DELETE FROM friends WHERE user_id = $1 AND friend_id = $2
	`, userID, friendID)
	return err
}
