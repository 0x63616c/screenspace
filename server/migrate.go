package main

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed db/migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(databaseURL string) error {
	source, err := iofs.New(migrationsFS, "db/migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	// golang-migrate pgx5 driver uses "pgx5" scheme, replace postgres:// or postgresql://
	migrateURL := databaseURL
	migrateURL = strings.Replace(migrateURL, "postgresql://", "pgx5://", 1)
	migrateURL = strings.Replace(migrateURL, "postgres://", "pgx5://", 1)

	m, err := migrate.NewWithSourceInstance("iofs", source, migrateURL)
	if err != nil {
		return fmt.Errorf("migration init: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up: %w", err)
	}

	return nil
}
