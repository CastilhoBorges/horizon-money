# Persistence

**Date:** 2026-11-27

## What happened

- Wrote the persistence design (`docs/specs/2026-11-27-persistence-design.md`) and built it for #8:
  - the v0.1 money schema on PostgreSQL 19, with per-user ownership keys and `CHECK` constraints;
  - goose migrations embedded in the binary, run by `serve` before it listens and by a new `migrate`
    command;
  - sqlc queries and one repository per aggregate behind `internal/port`;
  - `Store.InTx` for multi-call transactions.
- The domain gained `ID`, stored records (`User`, `Workspace`, `Tag`, `Movement`, `Transfer`, …),
  `Series.Validate` and `NewLedger`.
- The §4.14 acceptance fixture written through the repositories and read back gives the same balances.
- Harness:
  - an `Integration` CI job on a Postgres 19 service container;
  - a check that the sqlc output is fresh;
  - `pnpm db:worktree <slug>`;
  - `pnpm gen` and `pnpm test:integration`;
  - `scripts/with-env.mjs` to load `.env` into package scripts.
- ADR-0014 records UUIDv7 ids and the ownership keys. ADR-0015 records the test database strategy.

## Learned

- With `NO ACTION`, deleting a user cascades through transfers and workspaces in one statement, while
  deleting one workspace with transfers still fails. `RESTRICT` would have blocked the user deletion too.
- An `ON CONFLICT DO NOTHING` upsert skips the foreign-key check on a conflicting row, so repositories
  check that the parent is the caller's before writing.
- golangci-lint ignores files behind build tags unless `run.build-tags` lists them.
- Turborepo does not load `.env` into tasks, and `node --env-file` does not reach a `node --run` script,
  so the package scripts load `.env` through a small runner.
