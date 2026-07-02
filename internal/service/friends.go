package service

import (
	"context"

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
	return s.repo.AddFriend(ctx, userID, friendID)
}

func (s *FriendsService) DeleteFriend(ctx context.Context, userID, friendID string) error {
	return s.repo.DeleteFriend(ctx, userID, friendID)
}

func (s *FriendsService) ListFollowers(ctx context.Context, userID string) ([]string, error) {
	return s.repo.ListFollowerIDs(ctx, userID)
}
