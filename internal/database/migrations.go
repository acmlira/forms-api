package database

import (
	"forms/internal/logger"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrations() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		logger.Fatal("DATABASE_URL not set", nil)
	}

	migration, err := migrate.New(
		"file://migrations",
		url,
	)
	if err != nil {
		logger.Fatal("migration init error", err)
	}

	if _, dirty, _ := migration.Version(); dirty {
		logger.Error("database is dirty, forcing version", nil)
		if err := migration.Force(2025041002); err != nil {
			logger.Fatal("failed to force migration version", err)
		}
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("migration failed", err)
	} else {
		logger.Info("migrations applied")
	}
}
