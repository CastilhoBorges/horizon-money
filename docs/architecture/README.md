# Architecture

This folder describes the code **as it is**. Add one file per area as the area is built (`http.md`,
`mcp.md`, `persistence.md`, `money-and-dates.md`, `auth.md`, …) and list it below. The target design is
in [`../specs/`](../specs/).

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
| Configuration from env | Built (port, log level) |
| HTTP | `GET /healthz` only |
| Everything else | See the v0.1 milestone |
