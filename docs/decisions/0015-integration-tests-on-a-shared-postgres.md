# ADR-0015: Integration tests on a shared Postgres, not testcontainers

## Status
Accepted

## Date
2026-11-27

## Context
The repositories need tests against a real Postgres (§6.10). Developers already run the compose Postgres
for `pnpm dev`, and CI can start one as a service container. The spec left the test database strategy to
the builder (§8).

## Decision
- **Integration tests carry the `integration` build tag** and read `TEST_DATABASE_URL`, an admin
  connection. Locally it is the compose Postgres; in CI it is a `postgres:19-alpine` service container.
- **Each test package creates a throwaway database** `horizon_test_<random>` and drops it at the end.
- **Inside a package, tests isolate their data by creating their own users,** so they run in parallel
  without truncating.
- Without `TEST_DATABASE_URL`, `go test -tags=integration` skips those tests, while
  `pnpm test:integration` refuses to start, so the suite never passes without running.

## Alternatives considered
- **testcontainers:** self-contained, but it adds a heavy dependency, needs the Docker socket inside the
  tests, and starts a container per package.
- **One shared database truncated between tests:** forces the tests to run serially.

## Consequences
- `pnpm test` stays Docker-free.
- The `Integration` CI job is a required check.
- Running the suite locally needs `pnpm db:up` and a `.env` with `TEST_DATABASE_URL`.
