# ADR-0004: MCP server inside the API binary

## Status
Accepted

## Date
2026-11-27

## Context
The MCP connector is the product's main face. It must expose every use case with the same rules and
numbers as the HTTP API, and self-hosters should run a single process.

## Decision
Serve MCP from the same binary, at `/mcp`, over stateless Streamable HTTP with the official Go SDK. Tools
call the same use cases as HTTP handlers. Tools are named with the `horizon_` prefix, declare output
schemas, paginate lists, return actionable errors and carry read-only, destructive and idempotent
annotations. Ready-made prompts (for example "close my month") ship with the tools.

## Alternatives considered
- **A separate MCP service calling the HTTP API:** a second deployable, a second auth path, and a risk
  that the two drift apart.
- **stdio only:** works for local tools but not for cloud agents such as the Claude apps.

## Consequences
- One deployment and one auth system.
- A parity test checks that HTTP and MCP return the same numbers.
- Cloud agents need the install to be reachable over public HTTPS.
