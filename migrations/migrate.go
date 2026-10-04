package migrations

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

func Run(databaseURL string) error {
	source, err := iofs.New(sqlFiles, "sql")
	if err != nil {
		return fmt.Errorf("load migration files: %w", err)
	}

	migrateURL := strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
	migrateURL = strings.Replace(migrateURL, "postgresql://", "pgx5://", 1)

	m, err := migrate.NewWithSourceInstance("iofs", source, migrateURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
