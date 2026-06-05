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

func (r *UserRepository) Create(ctx context.Context, u model.User) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (first_name, second_name, birthdate, biography, city, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		u.FirstName, u.SecondName, u.Birthdate, u.Biography, u.City, u.PasswordHash,
	).Scan(&id)
	return id, err
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

func (r *UserRepository) FindByNames(ctx context.Context, fname, lname string) (*[]model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, first_name, second_name, birthdate, COALESCE(biography, ''), city
		 FROM users WHERE first_name ILIKE $1 AND second_name ILIKE $2 ORDER BY id`,
		fname, lname,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer rows.Close()

	var result []model.User

	for rows.Next() {
		var u model.User

		err := rows.Scan(&u.ID, &u.FirstName, &u.SecondName, &u.Birthdate, &u.Biography, &u.City)
		if err != nil {
			return nil, err
		}

		result = append(result, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &result, nil
}
