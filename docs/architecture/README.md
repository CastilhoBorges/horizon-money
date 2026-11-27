# Architecture

This folder describes the code **as it is**. Add one file per area as the area is built (`http.md`,
`mcp.md`, `persistence.md`, `auth.md`, …) and list it below. The target design is in
[`../specs/`](../specs/).

## Areas

- [`domain.md`](domain.md): money, dates, series, forecast, balances and summaries (`internal/domain`)
- [`persistence.md`](persistence.md): schema, migrations, sqlc queries, repositories and their tests

## Repository layout

```
horizon-money/
├─ apps/
│  ├─ api/                 Go module: the horizon-money binary
│  └─ web/                 Next.js app (later)
├─ packages/               shared TS packages (e.g. generated API client, later)
├─ docs/                   this documentation
├─ .agents/skills/         agent skills for every coding agent (linked into .claude/skills/)
├─ .github/workflows/      CI
└─ compose.yaml            Postgres + server (+ Mailpit in the dev profile)
```

## The Go server (`apps/api`): hexagonal

```
cmd/horizon-money/         entrypoint: `serve` and admin commands; wiring only
internal/
├─ config/                 environment configuration (12-factor)
├─ domain/                 pure rules: money, dates, recurrence, forecast, balances, summaries
├─ app/                    use cases (primary ports); authorize, call domain and secondary ports
├─ port/                   secondary ports: repositories, RateProvider, Clock, mailer, …
└─ adapter/
   ├─ httpapi/             HTTP routes (driving)
   ├─ mcpserver/           MCP tools and prompts (driving)
   └─ …                    postgres, bcb, google, smtp, web pages, oauth (as they are built)
```

Dependencies point inward. Adapters depend on `app` and `port`, `app` depends on `domain` and `port`,
and `domain` depends on nothing. The boundaries are enforced by lint, not by convention
(`apps/api/.golangci.yml`):
- `depguard` stops `internal/domain` from importing adapters, `app`, `net/http`, `database/sql` or pgx,
  and stops `internal/app` from importing adapters or pgx;
- `forbidigo` stops `internal/domain` and `internal/app` from calling `time.Now`, so time always comes
  from the `Clock` port.

## Current state

| Area | Status |
|---|---|
| Entry point (`serve`, `healthcheck`, graceful shutdown, slog JSON) | Built |
| Entry point `migrate` | Built: applies pending migrations and exits |
| Configuration from env | Built (port, log level, database URL) |
| Domain (`internal/domain`) and the `Clock` port | Built: money, dates, series, forecast, balances, summaries |
| Persistence (`db/`, `internal/port`, `internal/adapter/postgres`) | Built: money schema, migrations, repositories |
| HTTP | `GET /healthz` only |
| Everything else | See the v0.1 milestone |
