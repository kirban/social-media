package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/kirban/social-media/internal/config"
)

type DB struct {
	*sql.DB
}

func New(cfg config.DBConfig) (*DB, error) {
	dsn := buildDSN(cfg)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &DB{sqlDB}, nil
}

func buildDSN(cfg config.DBConfig) string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.DBName, cfg.Username, cfg.Password, cfg.SSLMode,
	)
}
