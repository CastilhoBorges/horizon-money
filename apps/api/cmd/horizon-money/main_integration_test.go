//go:build integration

package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/pgtest"
)

func TestMigrateCommandAppliesTheSchema(t *testing.T) {
	t.Parallel()

	dbURL := pgtest.New(t)
	getenv := func(key string) string {
		if key == "DATABASE_URL" {
			return dbURL
		}
		return ""
	}
	if err := run(context.Background(), []string{"migrate"}, getenv, &bytes.Buffer{}); err != nil {
		t.Fatalf("run(migrate) error = %v", err)
	}

	conn, err := pgx.Connect(t.Context(), dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(t.Context())) }()
	var users int
	if err := conn.QueryRow(t.Context(), "select count(*) from users").Scan(&users); err != nil {
		t.Fatalf("the users table is missing after migrate: %v", err)
	}
}
