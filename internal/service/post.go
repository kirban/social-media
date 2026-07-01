package service

import (
	"context"

	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/repository"
)

type PostsServiceInterface interface {
	GetFeed(ctx context.Context) ([]struct{}, error)
	Create(ctx context.Context, dto struct{}) (string, error)
	GetById(ctx context.Context, id string) (struct{}, error)
	Update(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type PostsService struct {
	repo *repository.PostRepository
}

func NewPostsService(repo *repository.PostRepository) *PostsService {
	return &PostsService{
		repo: repo,
	}
}

func (s *PostsService) GetFeed(ctx context.Context) ([]model.Post, error) {
	return nil, nil
}

func (s *PostsService) Create(ctx context.Context, dto *model.Post) (string, error) {
	id, err := s.repo.Create(ctx, dto)
	return id, err
}

func (s *PostsService) GetById(ctx context.Context, id string) (*model.Post, error) {
	return nil, nil
}

func (s *PostsService) Update(ctx context.Context, id string) error {
	return nil
}

func (s *PostsService) Delete(ctx context.Context, id string) error {
	return nil
}
