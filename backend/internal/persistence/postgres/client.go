package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentinelexam/backend/internal/platform/config"
)

// DB encapsulates the PostgreSQL connection pool.
type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewClient initializes a connection pool to the authoritative PostgreSQL database.
func NewClient(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*DB, error) {
	dsn := cfg.DatabaseDSN()
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres dsn: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.DatabaseMaxOpenConns)
	poolConfig.MinConns = int32(cfg.DatabaseMaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.DatabaseConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	return &DB{
		Pool:   pool,
		logger: logger,
	}, nil
}

// Ping verifies database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	if db.Pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	return db.Pool.Ping(ctx)
}

// Close gracefully closes all database connections.
func (db *DB) Close() {
	if db.Pool != nil {
		db.logger.Info("closing postgres connection pool")
		db.Pool.Close()
	}
}
