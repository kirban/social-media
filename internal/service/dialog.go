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
