// Command horizon-money runs the Horizon Money server and its admin commands.
//
// Usage:
//
//	horizon-money [serve]      start the HTTP + MCP server (default)
//	horizon-money healthcheck  exit 0 when the local server answers /healthz
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/httpapi"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/config"
)

const shutdownTimeout = 15 * time.Second

func main() {
	if err := runWithSignals(); err != nil {
		fmt.Fprintln(os.Stderr, "horizon-money:", err)
		os.Exit(1)
	}
}

// runWithSignals stops the process cleanly on SIGINT or SIGTERM.
func runWithSignals() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, os.Args[1:], os.Getenv, os.Stdout)
}

// run dispatches the subcommand. It takes its dependencies as arguments so tests can drive it.
func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	switch command {
	case "serve":
		return serve(ctx, cfg, stdout)
	case "healthcheck":
		return healthcheck(ctx, cfg)
	default:
		return fmt.Errorf("unknown command %q (want serve or healthcheck)", command)
	}
}

func serve(ctx context.Context, cfg config.Config, stdout io.Writer) error {
	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	server := &http.Server{
		Addr:              net.JoinHostPort("", strconv.Itoa(cfg.Port)),
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server.serve.started", "port", cfg.Port)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("server stopped: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped: %w", err)
	}
	logger.Info("server.serve.stopped")
	return nil
}

// healthcheck lets container runtimes probe the server without curl in the image.
func healthcheck(ctx context.Context, cfg config.Config) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port)) + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build healthcheck request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	return nil
}
