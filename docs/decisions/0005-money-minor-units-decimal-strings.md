# ADR-0005: Money as minor units inside, decimal strings at the edge

## Status
Accepted

## Date
2026-11-27

## Context
Floating-point money produces cent errors. Currencies have different scales (BRL 2, JPY 0). LLM agents
misread minor units: they send `26` meaning 26.00, which would be read as 0.26.

## Decision
- Inside the domain and Postgres, amounts are `int64` minor units, and the currency scale comes from
  ISO 4217 data.
- At HTTP and MCP, amounts are decimal strings (`"26.00"`) validated against the currency's scale.
- Rounding rules live in one place: half-up for the daily forecast, floor plus remainder on the last
  installment.

## Alternatives considered
- **Minor units in the API:** common in payment APIs, but error-prone for agents.
- **Decimal types end to end:** heavier, and not needed once rounding is centralised.

## Consequences
- No float ever touches money, and a lint or review flags one.
- Every boundary parses and formats with the same helpers.
