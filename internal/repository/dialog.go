package repository

import (
	"context"

	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

type DialogRepositoryInterface interface {
	CreateMessage(ctx context.Context, dialogID model.DialogID, from, to model.UserID, text string) (*model.DialogMessageID, error)
	GetDialogID(ctx context.Context, srcUser, dstUser model.UserID) (*model.DialogID, error)
	ListMessages(ctx context.Context, dialogID *model.DialogID) ([]model.DialogMessage, error)

	GetByID(ctx context.Context, id model.DialogID) (*model.Dialog, error)
	Create(ctx context.Context, dto any) (*model.DialogID, error)
	Update(ctx context.Context) error
	Delete(ctx context.Context) error
	ListDialogs(ctx context.Context, userID model.UserID) ([]model.Dialog, error)
}

type DialogRepository struct {
	cluster *db.Cluster
	log     *logger.AppLogger
}

func NewDialogRepository(c *db.Cluster, l *logger.AppLogger) *DialogRepository {
	return &DialogRepository{
		cluster: c,
		log:     l,
	}
}

func (r *DialogRepository) CreateMessage(ctx context.Context, dialogID model.DialogID, from, to model.UserID, text string) (*model.DialogMessageID, error) {
	return nil, nil
}

func (r *DialogRepository) GetDialogID(ctx context.Context, srcUser, dstUser model.UserID) (*model.DialogID, error) {
	return nil, nil
}

func (r *DialogRepository) ListMessages(ctx context.Context, dialogID *model.DialogID) ([]model.DialogMessage, error) {
	return nil, nil
}
