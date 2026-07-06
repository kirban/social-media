package service

import (
	"context"

	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/repository"
)

type DialogServiceInterface interface {
	GetMessages(ctx context.Context, srcUser, dstUser model.UserID) ([]model.DialogMessage, error)
	SendMessage(ctx context.Context, srcUser, dstUser model.UserID, text string) (*model.DialogMessageID, error)
}

type DialogService struct {
	log  *logger.AppLogger
	repo *repository.DialogRepository
}

func NewDialogService(l *logger.AppLogger, r *repository.DialogRepository) *DialogService {
	return &DialogService{
		log:  l,
		repo: r,
	}
}

func (s *DialogService) GetMessages(ctx context.Context, srcUser, dstUser model.UserID) ([]model.DialogMessage, error) {
	dialogID, err := s.repo.GetDialogID(ctx, srcUser, dstUser)
	if err != nil {
		return nil, err
	}

	messages, err := s.repo.ListMessages(ctx, dialogID)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *DialogService) SendMessage(ctx context.Context, srcUser, dstUser model.UserID, text string) (*model.DialogMessageID, error) {
	dialogID, err := s.repo.GetDialogID(ctx, srcUser, dstUser)
	if err != nil {
		return nil, err
	}

	messageID, err := s.repo.CreateMessage(ctx, *dialogID, srcUser, dstUser, text)
	if err != nil {
		return nil, err
	}

	return messageID, nil
}
