# AGENTS.md — Operating manual for Horizon Money

Every coding agent (Claude Code, Codex, Cursor, Gemini CLI, Copilot, OpenCode, …) and every human
contributor reads this at the start of a session. It defines **how to work in this repo**. The `docs/`
folder defines **what is true about the project** (see the documentation map at the end).

## What this project is

Horizon Money is an open-source, self-hosted personal-finance forecasting app whose main face is an
**MCP connector**: people plug it into their own AI agent and talk to it. It projects the balance day by
day for the coming months, across several workspaces (one balance and one currency each).

The approved design lives in `docs/specs/`. Read the relevant spec before changing behaviour.

## Before you start

- **Skills.** Shared skills live in `.agents/skills/`. Claude Code reads them through the links in
  `.claude/skills/` and gets **superpowers** from the plugin enabled in `.claude/settings.json`.
- **Other agents:** load the `using-superpowers` skill from `.agents/skills/` before any task. It explains
  how to find and use every other skill.
- **Spec-driven flow:** superpowers brainstorming → spec in `docs/specs/` → plan in `docs/plans/` when the
  work needs one → implementation in a worktree → PR.

## Tech stack

- **Backend (`apps/api`, Go):**
  - HTTP: `net/http` + Huma;
  - Postgres: pgx, sqlc and goose;
  - logging: `log/slog`;
  - MCP: the official MCP Go SDK, served in the same binary.
- **Frontend (`apps/web`, later):** Next.js on Node.
- **Monorepo:** Turborepo + pnpm workspaces.
- **Packaging:** Docker (distroless image) and `compose.yaml`.

## Golden rules (non-negotiable)

1. **Strict TDD.** Write the test first, watch it fail, then implement to green.
   - **Never edit a test to force a pass.** Tests encode the business rule. If one looks wrong, raise it.
   - Load the `tdd` skill before writing tests.
   - Agree the test seams with the maintainer before the first test of a feature.
   - There are two suites: unit (no Docker) and integration (real Postgres).
2. **English everywhere.** Code, identifiers, comments, docs and commit messages are in English.
   - The backend returns error **codes**, never user-facing prose. Clients translate them.
   - Server-rendered pages take every string from the message catalogue (pt-BR and en). Translation keys are in English.
3. **The server validates everything.** HTTP routes and MCP tools are public doors, and an agent is a client like any other.
   - Computed values (balances, totals, installment parts, effective rates) are never read from a request.
   - Every id received is looked up **scoped to the caller**.
   - A malformed id is a 400 or 404, never a 500.
   - Postgres constraints are the second lock, never the business rule.
4. **Git flow.** Never commit to `main`; a ruleset enforces this.
   - Branches: `feat/…`, `fix/…`, `chore/…`, `docs/…`, `ci/…`.
   - Open a PR and merge only with green CI, as a squash merge whose PR title is a Conventional Commit.
5. **One worktree per task, created with its branch:**

   ```bash
   git worktree add -b feat/<slug> .claude/worktrees/feat-<slug> main
   cp .env .claude/worktrees/feat-<slug>/.env      # .env is gitignored
   pnpm install --dir .claude/worktrees/feat-<slug>
   ```

   - Each worktree uses **its own database** (`horizon_<slug>`) on the shared Postgres container and its
     own `PORT`, so a reset in one task never breaks another. `pnpm db:worktree <slug>` creates and
     migrates it and prints the `DATABASE_URL` line for the worktree's `.env`.
   - Remove the worktree and branch after merge.
6. **Conventional Commits.** commitlint checks them in the commit-msg hook, and CI checks the PR title.
7. **Structured logging at every boundary:** HTTP, MCP, repositories and outbound clients.
   - Use `slog` JSON, one line per event, named `domain.function.what_happened`.
   - Levels: `error` = unexpected infra failure · `warn` = validation, permission or business denial ·
     `info` = successful state change.
   - Never swallow an error.
   - Never log secrets, tokens, passwords, **descriptions or amounts** (personal financial data). Log ids and codes.
8. **Doors are thin.** A handler or MCP tool authenticates, authorizes, calls **one** use case and maps a DTO.
   - A business rule in a handler is a defect.
   - HTTP and MCP stay in parity, and a test checks that they return the same numbers.
9. **Money and time invariants.** These are tests, not comments.
   - Minor units (`int64`) inside, decimal strings at the edge, never floats.
   - Rounding lives in one place in `internal/domain`.
   - The monthly balance identity holds.
   - A transfer is written atomically.
   - Installments sum to the total.
   - No invented exchange rates.
   - Creates are idempotent.
   - `internal/domain` and `internal/app` never call `time.Now()`. Time comes from the `Clock` port, and lint enforces it.
10. **Spec time is skill time.** A spec or design phase starts by checking the **live** skill list and
    invoking every relevant skill. The spec lists the skills it consulted. A hook reminds you, but the
    rule is the requirement.
11. **Reuse before writing; the third copy is a defect.**
    - Search before adding a function, type or rule:

      ```bash
      rg -n "<what it does, in words>" apps packages
      ls apps/api/internal/domain apps/api/internal/app apps/api/internal/adapter
      ```

    - One copy is code. Two is a coincidence you may leave, with a reason. Three is a defect: extract the
      code into the layer that owns it and rewrite every call site in the same PR.
    - The PR says what it reused or extracted.
    - The shelves are `internal/domain` (pure rules), `internal/app` (use cases), `internal/port`,
      `internal/adapter/*` and `packages/*` (only when a second app consumes it).
12. **Where a value lives.**
    - A business parameter a user could want to change is a **typed DB column** with a seeded default.
      A default is still a parameter.
    - A validation limit is a **constant in `internal/domain`**.
    - A secret or an ops switch lives in the **env**.
    - Settings are typed columns, never key-value.
13. **Every payload carries only what its consumer uses.**
    - Responses go through DTOs; a database row is never serialized, and spreading a row is a defect.
    - A test pins the fields a caller must not receive, such as password or token hashes.
    - MCP responses stay compact.
14. **Docs grow sideways** (`docs/README.md`).
    - History (ADRs, devlog) is one new file per entry, never rewritten.
    - Living docs stay under ~300 lines and are split by area when they grow.
    - Indexes are generated (`pnpm docs:index`).
15. **The repo is public.** Docs, fixtures and commits never carry personal data or private context.
    Test data is fictional.

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go skills the task needs.

## Quality gates

| Gate | How | Where |
|---|---|---|
| Format + lint | `golangci-lint` (`apps/api/.golangci.yml`; forbidigo + depguard encode rules 9 and the hexagonal boundaries) | pre-commit, CI |
| Tests | `go test -race ./...` | CI |
| Vulnerabilities | `govulncheck` | CI |
| Commit message | commitlint (Conventional Commits) | commit-msg hook |
| PR title | Conventional Commit check | CI |
| Docs | line limit, generated indexes, third-party notices | CI |

## Commands (run from the repository root)

```bash
pnpm install          # install tooling and git hooks
pnpm db:up            # Postgres + Mailpit for local development
pnpm dev              # run the server
pnpm test             # unit tests
pnpm lint             # lint and format check
pnpm build            # build everything
pnpm docs:index       # regenerate docs indexes and THIRD_PARTY_NOTICES.md
pnpm docs:check       # what CI checks for docs
docker compose up     # self-hosted stack (Postgres + server)
```

## Definition of Done

A task is done only when all of these hold:
- Tests were written first and pass.
- Lint, vet and docs checks are clean.
- Reuse was checked, nothing exists a third time, and the PR says what was reused or extracted.
- It is committed on a feature branch, and the PR title is a Conventional Commit.
- CI is green and the PR **closes an issue** (`Closes #N`).
- If it changes behaviour, a **devlog entry** is added (`docs/devlog/YYYY/YYYY-MM-DD-<slug>.md`).
- If it makes or changes a decision, an **ADR** is added.
- `docs/architecture/*` and `docs/GLOSSARY.md` are updated if the PR changes what they describe.
- `docs/ROADMAP.md` is updated if a phase starts or ends.

## Documentation map — consult before inventing

| When you need to know… | Read |
|---|---|
| The approved design: rules, data model, architecture | `docs/specs/` |
| What the project is and its scope | `docs/PROJECT.md` |
| A domain term or the pt-BR → code naming | `docs/GLOSSARY.md` |
| Folder structure, layers, boundaries, data flow | `docs/architecture/` |
| Why a choice was made | `docs/decisions/` |
| What was done and learned over time | `docs/devlog/` |
| What comes next | `docs/ROADMAP.md` and the GitHub issues |

If you are unsure about a name, a decision or a rule, consult the docs above **before** inventing. If the
answer is not there, ask; do not guess.
