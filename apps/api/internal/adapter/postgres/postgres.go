// Package postgres implements the repository ports on PostgreSQL through pgx and sqlc-generated queries.
package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/CastilhoBorges/horizon-money/apps/api/db"
)

// Pool limits. One install serves a handful of people, so a small pool is plenty.
const (
	maxConns        = 10
	maxConnLifetime = 30 * time.Minute
	maxConnIdleTime = 5 * time.Minute
)

// Connect opens a connection pool and checks that the database answers.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MaxConnLifetime = maxConnLifetime
	cfg.MaxConnIdleTime = maxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies every pending migration embedded in the binary. Running it again changes nothing.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return withProvider(pool, func(p *goose.Provider) error {
		_, err := p.Up(ctx)
		return err
	})
}

// MigrateDownAll rolls every migration back. It exists for tests and local resets, never for serve.
func MigrateDownAll(ctx context.Context, pool *pgxpool.Pool) error {
	return withProvider(pool, func(p *goose.Provider) error {
		_, err := p.DownTo(ctx, 0)
		return err
	})
}

func withProvider(pool *pgxpool.Pool, run func(*goose.Provider) error) error {
	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		return fmt.Errorf("prepare migrations: %w", err)
	}
	if err := run(provider); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
