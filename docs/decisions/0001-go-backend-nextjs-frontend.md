# ADR-0001: Go backend, Next.js frontend

## Status
Accepted

## Date
2026-11-27

## Context
Three consumers need the same business rules: the HTTP API (for the future web app), the MCP server
(for agents) and the server-rendered account pages. The app is self-hosted, often on small machines.
Most code is written by AI agents and reviewed by people.

## Decision
Write the backend in Go: `net/http` with Huma for HTTP, the official MCP Go SDK, and `log/slog`. Write the
web app later in Next.js on Node. Generate the web app's types from the OpenAPI document that Huma
produces.

## Alternatives considered
- **TypeScript everywhere (Bun + Hono):** a single language with shared schemas. Rejected because Bun
  adds runtime risk, and shared types are recovered through generated OpenAPI anyway.
- **Python (FastAPI):** mature MCP SDK. Rejected because it has two languages without Go's deployment
  story or TypeScript's shared types.

## Consequences
- A single static binary in a small image makes self-hosting easy.
- The language is small and the compiler strict, which keeps AI-written code easy to review.
- The monorepo has two toolchains. The OpenAPI document becomes a contract, checked for drift in CI.
