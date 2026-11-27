# Project kickoff

**Date:** 2026-11-27

## What happened

- Wrote and approved the two v0.1 designs: product rules and backend
  (`docs/specs/2026-11-27-product-and-backend-design.md`) and repository, agent harness and open-source
  setup (`docs/specs/2026-11-27-repository-and-harness-design.md`).
- Created the public repository with squash-only merges, a protected `main` (PR, green checks, linear
  history), Discussions, Dependabot alerts, secret scanning with push protection and private
  vulnerability reporting.
- Opened the backlog as GitHub issues under two milestones: Phase 0 — Foundation, and v0.1 — Backend.
- Shipped the monorepo skeleton (#1): pnpm + Turborepo, the Go `horizon-money` binary with `serve` and
  `healthcheck`, a distroless image, and `compose.yaml` with Postgres 18 and Mailpit.
- Shipped the quality gates (#2): golangci-lint with project rules, lefthook, commitlint, and CI with
  pinned actions and a PR-title check.
- Added the agent harness and this documentation layout (#3).

## Learned

- The domain rules (forecast cut-off at midnight, the monthly calculations) were checked by hand against
  worked examples before any code. The acceptance fixture in the spec is fictional and recomputed by a
  script, so the expected values are exact.
- Turborepo runs Go through a thin `package.json` per module, which is its documented multi-language approach.
- pnpm 12 blocks dependency build scripts by default. lefthook needs `allowBuilds` in `pnpm-workspace.yaml`.
- `depguard` and `forbidigo` were proven against a throwaway file that violated both rules before they were trusted.
