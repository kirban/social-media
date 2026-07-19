package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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

// EventPublisher publishes post events to the broker. Post-create/delete
// fan-out (feed invalidation + WS notifications) happens asynchronously in a
// consumer, decoupled from the request path. Satisfied by *broker.Publisher.
type EventPublisher interface {
	PublishPostCreated(ctx context.Context, e model.PostCreatedEvent) error
	PublishPostDeleted(ctx context.Context, e model.PostDeletedEvent) error
}

type PostsService struct {
	log       *logger.AppLogger
	repo      *repository.PostRepository
	cache     cache.Cache
	publisher EventPublisher
}

func NewPostsService(repo *repository.PostRepository, c cache.Cache, log *logger.AppLogger, pub EventPublisher) *PostsService {
	return &PostsService{
		repo:      repo,
		cache:     c,
		log:       log,
		publisher: pub,
	}
}

func (s *PostsService) GetFeed(ctx context.Context, userID string, limit, offset int64) ([]model.Post, error) {
	cacheKey := model.FeedCacheKey(userID)

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

// Create persists the post, then publishes a post.created event. The broker
// consumer performs the follower fan-out (cache invalidation + WS notify). A
// publish failure is logged, not returned: the post is already committed, so
// the request succeeds; the feed cache TTL heals any missed invalidation.
func (s *PostsService) Create(ctx context.Context, dto *model.Post) (string, error) {
	post, err := s.repo.Create(ctx, dto)
	if err != nil {
		return "", err
	}

	evt := model.PostCreatedEvent{
		PostID:    post.ID,
		AuthorID:  post.CreatorID,
		Text:      post.Text,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.publisher.PublishPostCreated(ctx, evt); err != nil {
		s.log.Error().Err(err).Str("postID", post.ID).Msg("create: publish post.created failed")
	}

	return post.ID, nil
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

// Delete removes the post, then publishes a post.deleted event so every
// instance invalidates the author's followers' cached feeds.
func (s *PostsService) Delete(ctx context.Context, id, userID string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	evt := model.PostDeletedEvent{PostID: id, AuthorID: userID}
	if err := s.publisher.PublishPostDeleted(ctx, evt); err != nil {
		s.log.Error().Err(err).Str("postID", id).Msg("delete: publish post.deleted failed")
	}

	return nil
}
