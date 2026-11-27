# ADR-0013: PostgreSQL 19

## Status
Accepted

## Date
2026-11-27

## Context
ADR-0002 chose PostgreSQL 18. PostgreSQL 19 is the current major release, and the image bump came in
(#23) before the first migration existed, so no stored data has to be upgraded.

## Decision
Run PostgreSQL 19: `postgres:19-alpine` in `compose.yaml`, and the same major in CI service containers.
The rest of ADR-0002 stands: `pgx`, `sqlc` and `goose`.

## Alternatives considered
- **Stay on 18:** it is supported for years, but moving later means a `pg_upgrade` on every self-hoster's
  data volume. Moving now, with no schema yet, costs nothing.

## Consequences
- New installs start on 19, and no data migration is needed.
- A local volume first created by 18 keeps that cluster unused in `18/docker`; 19 starts a fresh one.
- A future major bump needs an upgrade note in the release, because by then installs hold data.
