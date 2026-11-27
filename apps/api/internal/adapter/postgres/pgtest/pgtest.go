//go:build integration

// Package pgtest gives integration tests throwaway Postgres databases. It needs TEST_DATABASE_URL, an
// admin connection to a Postgres such as the compose one; without it, the tests that ask for a database
// skip, so `go test -tags=integration` still runs anywhere.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// AdminURL is the admin connection the throwaway databases are created through, or "" when unset.
func AdminURL() string { return os.Getenv("TEST_DATABASE_URL") }

// Create makes an empty database named horizon_test_<random> and returns its URL and a function that
// drops it.
func Create(ctx context.Context, adminURL string) (dbURL string, drop func(context.Context) error, err error) {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return "", nil, fmt.Errorf("random database name: %w", err)
	}
	name := "horizon_test_" + hex.EncodeToString(suffix)

	if err := exec(ctx, adminURL, "create database "+pgx.Identifier{name}.Sanitize()); err != nil {
		return "", nil, fmt.Errorf("create test database: %w", err)
	}
	u, err := url.Parse(adminURL)
	if err != nil {
		return "", nil, fmt.Errorf("parse TEST_DATABASE_URL: %w", err)
	}
	u.Path = "/" + name
	drop = func(ctx context.Context) error {
		return exec(ctx, adminURL, "drop database if exists "+pgx.Identifier{name}.Sanitize()+" with (force)")
	}
	return u.String(), drop, nil
}

// New creates an empty throwaway database for one test and drops it when the test ends. It skips the test
// when TEST_DATABASE_URL is unset.
func New(t *testing.T) string {
	t.Helper()
	admin := AdminURL()
	if admin == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	dbURL, drop, err := Create(t.Context(), admin)
	if err != nil {
		t.Fatalf("pgtest.New: %v", err)
	}
	t.Cleanup(func() {
		if err := drop(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
	return dbURL
}

func exec(ctx context.Context, dbURL, sql string) error {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}
