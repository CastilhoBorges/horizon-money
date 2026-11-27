package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

// Postgres error codes the repositories translate.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

// Store gives the use cases their repositories over a connection pool.
type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewStore wraps an open pool.
func NewStore(pool *pgxpool.Pool, logger *slog.Logger) *Store {
	return &Store{pool: pool, logger: logger}
}

// Repositories binds every repository to the pool.
func (s *Store) Repositories() port.Repositories { return s.repositories(s.pool) }

// conn is a pool or a transaction: both run queries and can open a (nested) transaction.
type conn interface {
	sqlcgen.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

func (s *Store) repositories(db conn) port.Repositories {
	base := repo{db: db, q: sqlcgen.New(db), logger: s.logger}
	return port.Repositories{
		Users:      users{base.named("users")},
		Workspaces: workspaces{base.named("workspaces")},
	}
}

// repo is what every repository shares: the connection, its queries and the logger for unexpected
// failures.
type repo struct {
	db     conn
	q      *sqlcgen.Queries
	logger *slog.Logger
	entity string
}

// inTx runs fn in a transaction (a savepoint when the repository is already inside one).
func (r repo) inTx(ctx context.Context, fn func(q *sqlcgen.Queries) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(r.q.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r repo) named(entity string) repo {
	r.entity = entity
	return r
}

// fail logs an unexpected database failure once, as postgres.<entity>.<operation>.failed with ids only,
// and returns it wrapped. Callers do not log it again.
func (r repo) fail(ctx context.Context, operation string, err error, attrs ...any) error {
	r.logger.ErrorContext(ctx, "postgres."+r.entity+"."+operation+".failed", append(attrs, "error", err)...)
	return fmt.Errorf("postgres %s %s: %w", r.entity, operation, err)
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// violates reports whether err is the Postgres error code on the named constraint ("" matches any).
func violates(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && (constraint == "" || pgErr.ConstraintName == constraint)
}
