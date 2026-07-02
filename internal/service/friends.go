package service

import (
	"context"
	"fmt"

	"github.com/kirban/social-media/internal/cache"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/repository"
)

type FriendsService struct {
	repo  *repository.FriendsRepository
	cache cache.Cache
	log   *logger.AppLogger
}

func NewFriendsService(repo *repository.FriendsRepository, c cache.Cache, l *logger.AppLogger) *FriendsService {
	return &FriendsService{
		repo:  repo,
		cache: c,
		log:   l,
	}
}

func (s *FriendsService) AddFriend(ctx context.Context, userID, friendID string) error {
	err := s.repo.AddFriend(ctx, userID, friendID)
	if err == nil {
		go s.invalidateFeed(context.WithoutCancel(ctx), userID)
	}
	return err
}

func (s *FriendsService) DeleteFriend(ctx context.Context, userID, friendID string) error {
	err := s.repo.DeleteFriend(ctx, userID, friendID)
	if err == nil {
		go s.invalidateFeed(context.WithoutCancel(ctx), userID)
	}
	return err
}

func (s *FriendsService) ListFollowers(ctx context.Context, userID string) ([]string, error) {
	return s.repo.ListFollowerIDs(ctx, userID)
}

func (s *FriendsService) invalidateFeed(ctx context.Context, userID string) {
	cacheKey := fmt.Sprintf("user:%s:feed", userID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.log.Error().Err(err).Msgf("failed to invalidate feed for user %s", userID)
	}
}
