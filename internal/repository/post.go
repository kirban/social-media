package repository

import (
	"context"

	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/model"
)

type PostRepositoryInterface interface {
	Create(ctx context.Context, post *model.Post) (string, error)
	GetById(ctx context.Context, id string) (*model.Post, error)
	Update(ctx context.Context, id string, post *model.Post) error
	Delete(ctx context.Context, id string) error
	// GetFeed() ([]struct{}, error)
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

func (r *PostRepository) GetById(ctx context.Context, id string) (*model.Post, error) {
	return &model.Post{}, nil
}

func (r *PostRepository) Update(ctx context.Context, id string, post *model.Post) error {
	return nil
}

func (r *PostRepository) Delete(ctx context.Context, id string) error {
	return nil
}
