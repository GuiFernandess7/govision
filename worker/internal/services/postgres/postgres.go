package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns    = 5
	defaultMinConns    = 1
	defaultMaxConnLife = 30 * time.Minute
	defaultMaxConnIdle = 5 * time.Minute
	defaultHealthCheck = 1 * time.Minute
	defaultConnTimeout = 5 * time.Second
)

// NewConnection creates a connection pool to PostgreSQL using pgxpool.
// It validates the connection with a ping before returning.
func NewConnection(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	cfg.MaxConns = defaultMaxConns
	cfg.MinConns = defaultMinConns
	cfg.MaxConnLifetime = defaultMaxConnLife
	cfg.MaxConnIdleTime = defaultMaxConnIdle
	cfg.HealthCheckPeriod = defaultHealthCheck

	connCtx, cancel := context.WithTimeout(ctx, defaultConnTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(connCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("[POSTGRES] - Connection pool established successfully")
	return pool, nil
}
