// Package db holds the SQL that ships with the binary: goose migrations and sqlc queries.
package db

import "embed"

// Migrations are the goose migration files, embedded so the binary migrates its own database.
//
//go:embed migrations/*.sql
var Migrations embed.FS
