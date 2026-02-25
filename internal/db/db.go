package db

import (
	"context"
	"time"

	// "github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute

	// Database tracing disabled to reduce noise
	// config.ConnConfig.Tracer = otelpgx.NewTracer()

	return pgxpool.NewWithConfig(ctx, config)
}
