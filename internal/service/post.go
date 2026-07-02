package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kirban/social-media/internal/cache"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/repository"
)

type PostsServiceInterface interface {
	GetFeed(ctx context.Context, userID string, limit, offset int64) ([]model.Post, error)
	Create(ctx context.Context, dto *model.Post) (string, error)
	GetByID(ctx context.Context, id string) (*model.Post, error)
	Update(ctx context.Context, id string, post *model.Post) error
	Delete(ctx context.Context, id, userID string) error
}

type FollowerLister interface {
	ListFollowers(ctx context.Context, userID string) ([]string, error)
}

type PostsService struct {
	log     *logger.AppLogger
	repo    *repository.PostRepository
	cache   cache.Cache
	friends FollowerLister
}

func NewPostsService(repo *repository.PostRepository, c cache.Cache, f FollowerLister, log *logger.AppLogger) *PostsService {
	return &PostsService{
		repo:    repo,
		cache:   c,
		friends: f,
		log:     log,
	}
}

func (s *PostsService) GetFeed(ctx context.Context, userID string, limit, offset int64) ([]model.Post, error) {
	cacheKey := fmt.Sprintf("user:%s:feed", userID)

	if data, ok, err := s.cache.Get(ctx, cacheKey); err == nil && ok {
		var ids []string
		if err := json.Unmarshal(data, &ids); err == nil {
			start := int(offset)
			if start >= len(ids) {
				return []model.Post{}, nil
			}
			end := start + int(limit)
			if end > len(ids) {
				end = len(ids)
			}
			return s.repo.GetByIDs(ctx, ids[start:end])
		}
	}

	ids, err := s.repo.GetFeedIDs(ctx, userID, 1000)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(ids); err == nil {
		_ = s.cache.Set(ctx, cacheKey, data, cache.DefaultTTL)
	}

	start := int(offset)
	if start >= len(ids) {
		return []model.Post{}, nil
	}
	end := start + int(limit)
	if end > len(ids) {
		end = len(ids)
	}
	return s.repo.GetByIDs(ctx, ids[start:end])
}

func (s *PostsService) Create(ctx context.Context, dto *model.Post) (string, error) {
	id, err := s.repo.Create(ctx, dto)
	if err == nil {
		go s.invalidateFeedForUser(context.WithoutCancel(ctx), dto.CreatorID)
	}
	return id, err
}

func (s *PostsService) GetByID(ctx context.Context, id string) (*model.Post, error) {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (s *PostsService) Update(ctx context.Context, id string, post *model.Post) error {
	if err := s.repo.Update(ctx, id, post); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *PostsService) Delete(ctx context.Context, id, userID string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	go s.invalidateFeedForUser(context.WithoutCancel(ctx), userID)

	return nil
}

func (s *PostsService) invalidateFeedForUser(ctx context.Context, userID string) error {
	// get followers ids
	followersIDs, err := s.friends.ListFollowers(ctx, userID)
	if err != nil {
		return err
	}

	// delete cached feed entries
	for _, followerID := range followersIDs {
		cacheKey := fmt.Sprintf("user:%s:feed", followerID)
		err := s.cache.Delete(ctx, cacheKey)
		if err != nil {
			msg := fmt.Sprintf("failed to invalidate user feed(%s). deleting key %s", userID, cacheKey)
			s.log.Error().Err(err).Msg(msg)
			continue
		}
	}
	return nil
}
