package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kirban/social-media/internal/db"
	"github.com/kirban/social-media/internal/model"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *db.DB
}

func NewUserRepository(db *db.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, first_name, second_name, birthdate, biography, city, password_hash
		 FROM users WHERE id = $1`, id)

	var u model.User
	if err := row.Scan(&u.ID, &u.FirstName, &u.SecondName, &u.Birthdate, &u.Biography, &u.City, &u.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
