// Package config reads the process configuration from environment variables (12-factor).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// Config is the process configuration. Business parameters never live here: they are typed columns in
// the database.
type Config struct {
	Port     int
	LogLevel slog.Level
	// DatabaseURL is the Postgres connection string. serve and migrate require it; healthcheck does not.
	DatabaseURL string
}

// Load builds a Config from getenv, applying defaults and rejecting invalid values.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{Port: 8080, LogLevel: slog.LevelInfo, DatabaseURL: getenv("DATABASE_URL")}
	var errs []error

	if raw := getenv("PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("PORT must be a number between 1 and 65535, got %q", raw))
		} else {
			cfg.Port = port
		}
	}

	if raw := getenv("LOG_LEVEL"); raw != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(strings.ToUpper(raw))); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", raw))
		}
	}

	return cfg, errors.Join(errs...)
}
