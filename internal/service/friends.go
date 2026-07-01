package service

import (
	"context"

	"github.com/kirban/social-media/internal/repository"
)

type FriendsService struct {
	repo *repository.FriendsRepository
}

func NewFriendsService(repo *repository.FriendsRepository) *FriendsService {
	return &FriendsService{repo: repo}
}

func (s *FriendsService) AddFriend(ctx context.Context, userID, friendID string) error {
	return s.repo.AddFriend(ctx, userID, friendID)
}
