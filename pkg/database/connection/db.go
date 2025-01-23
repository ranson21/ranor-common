package connection

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ranson21/ranor-common/pkg/database/config"
)

type Database interface {
	Ping(ctx context.Context) error
	Close()
	GetPool() *pgxpool.Pool
}

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(cfg *config.DatabaseConfig) (Database, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("error parsing connection string: %w", err)
	}

	// Apply connection pool settings
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnIdleTime = cfg.MaxIdleTime
	poolConfig.MaxConnLifetime = cfg.MaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close() // Clean up if connection test fails
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Implement Database interface methods
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *DB) GetPool() *pgxpool.Pool {
	return db.pool
}
