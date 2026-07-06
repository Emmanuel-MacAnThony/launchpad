package db

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("parse database url: %v", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	return pool
}
