package main

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed db/migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(pool *pgxpool.Pool) error {
	source, err := iofs.New(migrationsFS, "db/migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	connStr := pool.Config().ConnConfig.ConnString()
	m, err := migrate.NewWithSourceInstance("iofs", source, "pgx5://"+connStr)
	if err != nil {
		return fmt.Errorf("migration init: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up: %w", err)
	}

	return nil
}
