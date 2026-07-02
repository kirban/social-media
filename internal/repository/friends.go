package repository

import (
	"context"

	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/logger"
)

type FriendsRepository struct {
	cluster *db.Cluster
	log     *logger.AppLogger
}

func NewFriendsRepository(cluster *db.Cluster, l *logger.AppLogger) *FriendsRepository {
	return &FriendsRepository{cluster: cluster, log: l}
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

func (r *FriendsRepository) ListFollowerIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.cluster.Replica().QueryContext(ctx, `
		SELECT user_id FROM friends
		WHERE friend_id = $1;
	`, userID)
	if err != nil {
		r.log.Error().Err(err).Msg("followers get: during sql request")
		return nil, err
	}
	defer rows.Close()

	followerIDs := make([]string, 0)

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			r.log.Error().Err(err).Msg("followers get: during rows scan")
			return nil, err
		}
		followerIDs = append(followerIDs, id)
	}

	if err := rows.Err(); err != nil {
		r.log.Error().Err(err).Msg("followers get: after rows scan")
		return nil, err
	}

	return followerIDs, nil
}
