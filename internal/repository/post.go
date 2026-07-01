package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/model"
)

type PostRepositoryInterface interface {
	Create(ctx context.Context, post *model.Post) (string, error)
	GetById(ctx context.Context, id string) (*model.Post, error)
	Update(ctx context.Context, id string, post *model.Post) error
	Delete(ctx context.Context, id string) error
	GetFeed(ctx context.Context, userID string) ([]model.Post, error)
}

type PostRepository struct {
	cluster *db.Cluster
}

func NewPostRepository(cluster *db.Cluster) *PostRepository {
	return &PostRepository{cluster: cluster}
}

func (r *PostRepository) Create(ctx context.Context, p *model.Post) (string, error) {
	var id string
	err := r.cluster.Master().QueryRowContext(ctx, `
		INSERT INTO posts (text, creator_id) VALUES ($1, $2)
		RETURNING id
	`, p.Text, p.CreatorID).Scan(&id)
	return id, err
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*model.Post, error) {
	row := r.cluster.Replica().QueryRowContext(ctx, `
		SELECT id, text, creator_id, created_at, updated_at FROM posts
		WHERE id = $1
	`, id)

	var post model.Post
	if err := row.Scan(&post.ID, &post.Text, &post.CreatorID, &post.CreatedAt, &post.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) Update(ctx context.Context, id string, post *model.Post) error {
	result, err := r.cluster.Master().ExecContext(ctx, `
		UPDATE posts SET text = $1, updated_at = NOW() WHERE id = $2
	`, post.Text, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostRepository) Delete(ctx context.Context, id string) error {
	result, err := r.cluster.Master().ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostRepository) GetFeed(ctx context.Context, userID string) ([]model.Post, error) {
	rows, err := r.cluster.Replica().QueryContext(ctx, `
		WITH friends_ids AS (
			SELECT friend_id FROM friends WHERE user_id = $1
		)
		SELECT id, text, creator_id, created_at, updated_at
		FROM posts
		WHERE creator_id IN friends_ids
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feed := make([]model.Post, 0)

	for rows.Next() {
		var post model.Post
		if err := rows.Scan(&post.ID, &post.Text, &post.CreatorID, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, err
		}
		feed = append(feed, post)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return feed, nil
}
