//go:build integration

package main

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

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

// TestServeAnswersHealthcheckAndStopsOnCancel moved here from the unit suite when serve started to need a
// database (persistence spec, Decision 3). serve migrates before it listens.
func TestServeAnswersHealthcheckAndStopsOnCancel(t *testing.T) {
	t.Parallel()

	port := strconv.Itoa(freePort(t))
	dbURL := pgtest.New(t)
	getenv := func(key string) string {
		switch key {
		case "PORT":
			return port
		case "DATABASE_URL":
			return dbURL
		}
		return ""
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	var logs bytes.Buffer
	go func() { done <- run(ctx, []string{"serve"}, getenv, &logs) }()

	deadline := time.Now().Add(10 * time.Second)
	for {
		err := run(context.Background(), []string{"healthcheck"}, getenv, &bytes.Buffer{})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("healthcheck never succeeded: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	conn, err := pgx.Connect(t.Context(), dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	var users int
	if err := conn.QueryRow(t.Context(), "select count(*) from users").Scan(&users); err != nil {
		t.Errorf("serve answered before migrating: %v", err)
	}
	_ = conn.Close(context.WithoutCancel(t.Context()))

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v after cancel, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}
