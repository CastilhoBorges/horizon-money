# ADR-0007: Turborepo and pnpm monorepo

## Status
Accepted

## Date
2026-11-27

## Context
The repository holds a Go server and, later, a Next.js app and generated TypeScript types. Some tasks
depend on others: the web build depends on types, which depend on the OpenAPI document the Go build
produces. Contributors should run everything with one command.

## Decision
Use pnpm workspaces with Turborepo. The Go module has a thin `package.json` whose scripts call Go, which
is the approach documented in Turborepo's multi-language guide. Root `pnpm` scripts are the single entry
point. Inside `apps/api`, plain `go` commands keep working.

## Alternatives considered
- **Nx:** heavier, and Go support comes from a community plugin.
- **moon:** natively multi-language, but less known by contributors.
- **Makefile only:** no task graph and no caching.

## Consequences
- One task graph across languages, with caching. Go's own cache still applies underneath.
- Contributors need Node and pnpm even for Go-only work at the root.
