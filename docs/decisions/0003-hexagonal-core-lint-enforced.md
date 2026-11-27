# ADR-0003: Hexagonal core with lint-enforced boundaries

## Status
Accepted

## Date
2026-11-27

## Context
The same use cases are reached through HTTP, MCP and account pages. They talk to Postgres, the BCB rate
API, Google and SMTP. Forecast rules depend on "today" and must be testable at any date.

## Decision
Use ports and adapters:
- `internal/domain` holds pure rules with no I/O and no wall clock;
- `internal/app` holds use cases and depends only on `domain` and `internal/port`;
- `internal/adapter/*` implements the doors and the outbound clients;
- wiring happens in `cmd/` with manual constructor injection, without a DI library.

The boundaries are enforced in `.golangci.yml`, not by convention:
- `depguard` keeps the domain free of I/O and adapters, and keeps use cases off adapters;
- `forbidigo` bans `time.Now` in `domain` and `app`.

## Alternatives considered
- **Flat packages:** simpler at first, but a rule would end up duplicated between HTTP and MCP.
- **A DI library (wire, fx):** not needed at this size, and it adds indirection.

## Consequences
- HTTP and MCP stay thin and in parity.
- Domain tests need no database.
- Breaking a boundary fails the build.
