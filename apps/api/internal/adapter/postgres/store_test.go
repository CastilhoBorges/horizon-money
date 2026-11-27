//go:build integration

package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/pgtest"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

// shared is one migrated database for the whole package. Tests isolate their data by creating their own
// users, so they run in parallel without truncating anything.
var shared struct {
	once sync.Once
	pool *pgxpool.Pool
	drop func(context.Context) error
	err  error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if shared.pool != nil {
		shared.pool.Close()
	}
	if shared.drop != nil {
		if err := shared.drop(context.Background()); err != nil {
			code = 1
		}
	}
	os.Exit(code)
}

func sharedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	if admin == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	shared.once.Do(func() {
		ctx := context.Background()
		var dbURL string
		if dbURL, shared.drop, shared.err = pgtest.Create(ctx, admin); shared.err != nil {
			return
		}
		if shared.pool, shared.err = postgres.Connect(ctx, dbURL); shared.err != nil {
			return
		}
		shared.err = postgres.Migrate(ctx, shared.pool)
	})
	if shared.err != nil {
		t.Fatalf("shared test database: %v", shared.err)
	}
	return shared.pool
}

// newStore is a store over the shared database.
func newStore(t *testing.T) *postgres.Store {
	t.Helper()
	return postgres.NewStore(sharedPool(t), slog.New(slog.DiscardHandler))
}

// uniqueEmail is a fictional address no other test uses.
func uniqueEmail(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random email: %v", err)
	}
	return "person-" + hex.EncodeToString(b) + "@example.test"
}

// newUser creates a user and returns its id.
func newUser(t *testing.T, repos port.Repositories) domain.ID {
	t.Helper()
	u, err := repos.Users.Create(t.Context(), port.NewUser{Email: uniqueEmail(t), DisplayName: "Test Person"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID
}
