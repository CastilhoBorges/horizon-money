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

	for _, command := range []string{"serve", "migrate"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			port := strconv.Itoa(freePort(t))
			getenv := func(key string) string {
				if key == "PORT" {
					return port
				}
				return ""
			}
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			err := run(ctx, []string{command}, getenv, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
				t.Fatalf("run(%s) without DATABASE_URL error = %v, want one naming DATABASE_URL", command, err)
			}
		})
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
