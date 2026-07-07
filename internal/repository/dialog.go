package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

type DialogRepositoryInterface interface {
	CreateMessage(ctx context.Context, dialogID *model.DialogID, from, to model.UserID, text string) (*model.DialogMessageID, error)
	GetDialogID(ctx context.Context, srcUser, dstUser model.UserID) (*model.DialogID, error)
	ListMessages(ctx context.Context, dialogID *model.DialogID) ([]model.DialogMessage, error)
	CreateDialog(ctx context.Context, dto *model.CreateDialogDTO) (*model.DialogID, error)

	GetByID(ctx context.Context, id model.DialogID) (*model.Dialog, error)
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

func (r *DialogRepository) CreateMessage(ctx context.Context, dialogID *model.DialogID, from, to model.UserID, text string) (*model.DialogMessageID, error) {
	var messID model.DialogMessageID
	err := r.cluster.Master().QueryRowContext(ctx, `INSERT INTO "dialog_message" ("dialog_id", "from", "to", "text")
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`, dialogID, from, to, text).Scan(&messID)
	if err != nil {
		return nil, fmt.Errorf("failed create message: %w", err)
	}
	return &messID, nil
}

func (r *DialogRepository) GetDialogID(ctx context.Context, srcUser, dstUser model.UserID) (*model.DialogID, error) {
	var dialogID model.DialogID
	err := r.cluster.Replica().QueryRowContext(
		ctx,
		`SELECT dialog_id
		FROM "dialog_user"
		WHERE user_id IN ($1::UUID, $2::UUID)
		GROUP BY dialog_id
		HAVING COUNT(DISTINCT user_id) = 2;`, srcUser, dstUser).Scan(&dialogID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &dialogID, nil
}

func (r *DialogRepository) ListMessages(ctx context.Context, dialogID *model.DialogID) ([]model.DialogMessage, error) {
	rows, err := r.cluster.Replica().QueryContext(ctx,
		`SELECT id, "from", "to", dialog_id, text, created_at, updated_at
		FROM "dialog_message"
		WHERE dialog_id = $1
		ORDER BY created_at ASC;`,
		dialogID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []model.DialogMessage
	for rows.Next() {
		var m model.DialogMessage
		if err := rows.Scan(&m.ID, &m.From, &m.To, &m.DialogID, &m.Text, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate messages: %w", err)
	}

	return messages, nil
}

func (r *DialogRepository) CreateDialog(ctx context.Context, dto *model.CreateDialogDTO) (*model.DialogID, error) {
	txOpts := &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
	tx, err := r.cluster.Master().BeginTx(ctx, txOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	var dialogID model.DialogID
	err = tx.QueryRowContext(ctx, `INSERT INTO "dialog" DEFAULT VALUES RETURNING id;`).Scan(&dialogID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert dialog: %w", err)
	}

	bulkInsertQ := `INSERT INTO "dialog_user" (dialog_id, user_id) VALUES ($1, $2), ($1, $3);`
	_, err = tx.ExecContext(ctx, bulkInsertQ, dialogID, dto.Users[0], dto.Users[1])
	if err != nil {
		return nil, fmt.Errorf("failed to insert dialog users: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &dialogID, nil
}
