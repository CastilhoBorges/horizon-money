# ADR-0002: PostgreSQL with pgx, sqlc and goose

## Status
Accepted

## Date
2026-11-27

## Context
The data is relational: users, workspaces, movements, series, transfers and tags. It needs strict
constraints and atomic transfers, and one install may host several people.

## Decision
Use PostgreSQL 18:
- `pgx` is the driver;
- `sqlc` generates typed Go from hand-written SQL, so there is no ORM;
- `goose` migrations are embedded in the binary and run on start.

## Alternatives considered
- **An ORM (GORM, ent):** less SQL to write, but query shape and constraints are harder to see and review.
- **SQLite:** zero setup, but weaker for several users at once and for future scaling.

## Consequences
- Queries are reviewed as SQL, and the compiler checks the Go that calls them.
- Database constraints act as the second lock behind the use cases.
- Self-hosters run a Postgres container, which `compose.yaml` provides.
