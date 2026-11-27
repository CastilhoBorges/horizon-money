package config_test

import (
	"log/slog"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/config"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want INFO", cfg.LogLevel)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(env(map[string]string{"PORT": "9090", "LOG_LEVEL": "debug"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want DEBUG", cfg.LogLevel)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]string{
		"port not a number": {"PORT": "abc"},
		"port out of range": {"PORT": "70000"},
		"port zero":         {"PORT": "0"},
		"unknown log level": {"LOG_LEVEL": "loud"},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := config.Load(env(values)); err == nil {
				t.Fatalf("Load(%v) error = nil, want an error", values)
			}
		})
	}
}
