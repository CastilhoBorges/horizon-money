//go:build integration

package postgres_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/pgtest"
)

// schemaTables are the tables the v0.1 money schema creates (persistence spec §5.1).
var schemaTables = []string{
	"checked_days", "forecast_categories", "movement_exception_tags", "movement_exceptions",
	"movement_tags", "movements", "tags", "transfers", "users", "workspaces",
}

func tables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(),
		`select table_name from information_schema.tables
		 where table_schema = 'public' and table_name <> 'goose_db_version' order by table_name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}
	return names
}

func TestMigrate_UpDownUp(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	pool, err := postgres.Connect(ctx, pgtest.New(t))
	if err != nil {
		t.Fatalf("Connect error = %v", err)
	}
	defer pool.Close()

	for step, run := range []struct {
		name     string
		migrate  func(context.Context, *pgxpool.Pool) error
		expected []string
	}{
		{name: "up from empty", migrate: postgres.Migrate, expected: schemaTables},
		{name: "up again changes nothing", migrate: postgres.Migrate, expected: schemaTables},
		{name: "down to empty", migrate: postgres.MigrateDownAll, expected: nil},
		{name: "up after down", migrate: postgres.Migrate, expected: schemaTables},
	} {
		if err := run.migrate(ctx, pool); err != nil {
			t.Fatalf("step %d (%s): error = %v", step, run.name, err)
		}
		if got := tables(t, pool); !slices.Equal(got, run.expected) {
			t.Fatalf("step %d (%s): tables = %v, want %v", step, run.name, got, run.expected)
		}
	}
}
