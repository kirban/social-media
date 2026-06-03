package db

import (
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

func (d *DB) Migrate() error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(d.DB, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
