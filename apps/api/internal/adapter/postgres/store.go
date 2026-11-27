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
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
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
		Users:       users{base.named("users")},
		Workspaces:  workspaces{base.named("workspaces")},
		Tags:        tags{base.named("tags")},
		Movements:   movements{base.named("movements")},
		Transfers:   transfers{base.named("transfers")},
		CheckedDays: checkedDays{base.named("checked_days")},
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

// done finishes a repository operation's error handling in one place: ErrNotFound and the domain's rule
// errors pass through, a foreign-key violation means a referenced record is missing or another user's, and
// anything else is an unexpected failure, logged once.
func (r repo) done(ctx context.Context, operation string, err error, attrs ...any) error {
	var rule *domain.Error
	switch {
	case err == nil:
		return nil
	case errors.Is(err, port.ErrNotFound), errors.As(err, &rule):
		return err
	case violates(err, foreignKeyViolation, ""):
		return port.ErrNotFound
	default:
		return r.fail(ctx, operation, err, attrs...)
	}
}

// workspaceExists returns ErrNotFound unless the workspace is the user's. Database errors come back raw,
// for done to handle.
func workspaceExists(ctx context.Context, q *sqlcgen.Queries, userID, workspaceID domain.ID) error {
	exists, err := q.WorkspaceExists(ctx, sqlcgen.WorkspaceExistsParams{ID: toUUID(workspaceID), UserID: toUUID(userID)})
	if err != nil {
		return err
	}
	if !exists {
		return port.ErrNotFound
	}
	return nil
}

// movementExists returns ErrNotFound unless the movement is the user's. Database errors come back raw.
func movementExists(ctx context.Context, q *sqlcgen.Queries, userID, movementID domain.ID) error {
	exists, err := q.MovementExists(ctx, sqlcgen.MovementExistsParams{ID: toUUID(movementID), UserID: toUUID(userID)})
	if err != nil {
		return err
	}
	if !exists {
		return port.ErrNotFound
	}
	return nil
}

// affected turns a write that touched no row into ErrNotFound: the id is missing or another user's.
func (r repo) affected(ctx context.Context, operation string, n int64, err error, id domain.ID) error {
	if err != nil {
		return r.fail(ctx, operation, err, "id", id.String())
	}
	if n == 0 {
		return port.ErrNotFound
	}
	return nil
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// violates reports whether err is the Postgres error code on the named constraint ("" matches any).
func violates(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && (constraint == "" || pgErr.ConstraintName == constraint)
}
