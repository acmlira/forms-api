package database

import (
	"context"
	"forms/internal/logger"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connection() *pgxpool.Pool {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		logger.Fatal("DATABASE_URL not set", nil)
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		logger.Fatal("Unable to parse DATABASE_URL", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		logger.Fatal("Unable to connect to database", err)
	}

	logger.Info("Connected to PostgreSQL!")
	return pool
}
