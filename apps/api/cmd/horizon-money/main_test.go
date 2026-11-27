package main

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	var lc net.ListenConfig
	l, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", l.Addr())
	}
	return addr.Port
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := run(context.Background(), []string{"bogus"}, func(string) string { return "" }, &out)
	if err == nil {
		t.Fatal("run(bogus) error = nil, want an error")
	}
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	for _, command := range []string{"migrate"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			err := run(context.Background(), []string{command}, func(string) string { return "" }, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
				t.Fatalf("run(%s) without DATABASE_URL error = %v, want one naming DATABASE_URL", command, err)
			}
		})
	}
}

func TestServeAnswersHealthcheckAndStopsOnCancel(t *testing.T) {
	t.Parallel()

	port := strconv.Itoa(freePort(t))
	getenv := func(key string) string {
		if key == "PORT" {
			return port
		}
		return ""
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	var logs bytes.Buffer
	go func() { done <- run(ctx, []string{"serve"}, getenv, &logs) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		err := run(context.Background(), []string{"healthcheck"}, getenv, &bytes.Buffer{})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("healthcheck never succeeded: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

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

func TestHealthcheckFailsWhenNothingListens(t *testing.T) {
	t.Parallel()

	port := strconv.Itoa(freePort(t))
	getenv := func(key string) string {
		if key == "PORT" {
			return port
		}
		return ""
	}
	if err := run(context.Background(), []string{"healthcheck"}, getenv, &bytes.Buffer{}); err == nil {
		t.Fatal("healthcheck error = nil with no server, want an error")
	}
}
