# Horizon Money — Repository, agent harness and open-source setup

- **Date:** 2026-11-27
- **Status:** Draft
- **Companion spec:** `2026-11-27-product-and-backend-design.md`
- **Skills consulted:** superpowers:brainstorming, find-skills, turborepo, golang-project-layout, documentation-and-adrs

## 1. Intent

AI agents will write most of the code, and outside people will contribute. The repository therefore needs:

1. **A harness:** written rules, automatic gates and docs that keep agents and people building the same way.
2. **A community-ready open-source setup:** from the GitHub settings down to the files in the root.

Goals:
- Any agent or contributor can clone the repo, read one manual, and work correctly.
- Gates catch violations before review.
- Docs stay small enough to read.

Anti-goals:
- Rules that live only in someone's head.
- A doc file that grows forever.
- Parallel tasks that keep conflicting on shared files.

## 2. Monorepo

**Turborepo + pnpm workspaces.** Node LTS and pnpm run the JS side and the task graph. The Go module lives
inside the workspace with a thin `package.json` whose scripts call Go.

```
horizon-money/
├─ apps/
│  ├─ api/                 # Go module (see backend spec §6.1); package.json wraps go commands
│  └─ web/                 # Next.js (later phase)
├─ packages/
│  └─ api-client/          # TS types generated from apps/api/openapi.json (created with the web phase)
├─ docs/                   # see §5
├─ .agents/skills/        # all agent skills, universal location (Codex, Cursor, Gemini CLI, Copilot, OpenCode, Amp…)
├─ .claude/
│  ├─ settings.json        # enabled plugins (superpowers) + hooks (§3.3)
│  └─ skills/              # symlinks into .agents/skills (project skills only, see §3.5)
├─ .github/                # CI, templates, community files (§6)
├─ AGENTS.md               # the agent manual (§3)
├─ CLAUDE.md               # one line: @AGENTS.md (+ Claude-only notes, if any)
├─ compose.yaml            # Postgres + API for self-hosting; `dev` profile adds Mailpit for local email
├─ turbo.json · pnpm-workspace.yaml · package.json · skills-lock.json
└─ LICENSE (MIT) · README.md · CONTRIBUTING.md · CODE_OF_CONDUCT.md · SECURITY.md · CHANGELOG.md
```

- **Task graph:** `api#gen` (sqlc + OpenAPI export) → `api-client#gen` (openapi-typescript) → `web#build`.
  `test`, `lint` and `build` run per package.
- **Turbo caching for Go:** Turbo caches Go tasks through declared `inputs` (`**/*.go`, `go.mod`, `go.sum`,
  `db/**`). Go's own build and test cache still applies underneath.
- **Turbo docs:** the builder reads the docs bundled with the installed `turbo` before configuring it
  (turborepo skill).
- **Root commands** are the single entry point (exact names are the builder's):
  `pnpm dev`, `pnpm test`, `pnpm test:integration`, `pnpm test:contract`, `pnpm lint`, `pnpm format`,
  `pnpm gen`, `pnpm build`, `pnpm db:up`, `pnpm docs:index`.
- **Inside `apps/api`,** plain `go test ./...` keeps working for Go-only contributors.
- **Go module path:** `github.com/CastilhoBorges/horizon-money/apps/api`.

## 3. Agent harness

### 3.1 AGENTS.md (manual for every agent), CLAUDE.md imports it

`AGENTS.md` is the single manual, because contributors may use Claude Code, Codex, Cursor or others. It holds:
- what the project is, the stack and the golden rules;
- the quality gates, the commands and the Definition of Done;
- the docs map;
- the always-load directive for the `golang-how-to` skill (that skill's configure mode).

### 3.2 Golden rules (adapted)

1. **Strict TDD.** Write the test first, watch it fail, then implement. Never edit a test to force a pass;
   if a test looks wrong, raise it. Load the `tdd` skill before writing tests, and agree the test seams
   with the maintainer before the first test of a feature. There are two suites: unit (no Docker) and
   integration (real Postgres).
2. **English everywhere.** Code, identifiers, comments, docs and commits are in English. The backend
   returns error **codes**, never user-facing prose. The frontend translates them (next-intl: `pt-BR`
   default, `en`), and translation keys are in English.
3. **The server validates everything.** HTTP routes and MCP tools are public doors, and an agent is a client like any other.
   - Computed values (balances, totals, installment parts, effective rates) are never read from a request.
   - Every id received is looked up scoped to the caller.
   - A malformed id is a 400 or 404, never a 500.
   - Postgres constraints are the second lock, never the business rule.
4. **Git flow.** Never commit to `main`. Use `feat/`, `fix/`, `chore/` or `docs/` branches and open a PR.
   Merge only with green CI, as a squash merge whose PR title is a Conventional Commit.
5. **One worktree per task,** created together with its branch:
   `git worktree add -b feat/<slug> .claude/worktrees/feat-<slug> main`.
   - Copy `.env` from the main checkout and run `pnpm install`.
   - Each worktree gets **its own database** (`horizon_<slug>`) on the shared Postgres container and its own `PORT`,
     so one task can reset its database without breaking another.
   - Remove the worktree and its branch after the merge. `.claude/worktrees/` is gitignored.
6. **Conventional Commits.** commitlint checks them locally, and CI checks the PR title.
7. **Structured logging at every boundary:** HTTP, MCP, repositories and the BCB client.
   - Use slog JSON with event names `domain.function.what_happened`.
   - Levels: `error` for unexpected infra failures, `warn` for denials, `info` for state changes.
   - Never swallow an error.
   - Never log secrets, tokens, descriptions or amounts.
8. **Doors are thin.** A handler or tool authenticates, authorizes, calls one use case and maps a DTO.
   A business rule in a handler is a defect. HTTP and MCP stay in parity, checked by a test.
9. **Money and date invariants.** These are tests, not comments:
   - Minor units inside, decimal strings at the edge, never floats.
   - Rounding lives in one place.
   - The monthly balance identity holds.
   - A transfer is written atomically.
   - Installments sum to the total.
   - No invented exchange rates.
   - Creates are idempotent.
   - The domain never calls `time.Now()`. Time comes from the `Clock` port, and a linter enforces this (e.g. forbidigo).
10. **Spec time is skill time.** The spec phase starts by checking the session's live skill list and
    invoking every relevant skill. The spec says which skills were consulted. A hook reminds (§3.3), but
    the rule is the requirement.
11. **Reuse before writing; the third copy is a defect.**
    - Search before adding a function, type or rule.
    - The shelves are: `internal/domain`, `internal/app`, `internal/adapter/*` and `packages/*`. The test for
      `packages/` is whether a second app consumes it.
    - At the third copy, extract the code and rewrite every call site in the same PR.
    - The PR says what it reused or extracted.
12. **Where a value lives.** A business parameter is a typed DB column with a seeded default. A validation
    limit is a constant in `domain`. A secret or switch goes in the env. A default is still a business
    parameter if a user could want to change it.
13. **Every payload carries only what its consumer uses.**
    - Responses always go through DTOs; a DB row is never serialized.
    - A test pins the fields a caller must not receive.
    - MCP responses stay compact.
14. **Docs follow §5:** one item per file, about 300 lines at most, generated indexes, immutable ADRs.

### 3.3 Hooks (`.claude/settings.json`)

- **SessionStart:** prints the docs map and the reuse reminder (rule 11).
- **PreToolUse on writing a new `.go`/`.ts` file:** prints the reuse reminder.
- **PreToolUse on `superpowers:brainstorming` / `superpowers:writing-plans`:** prints the rule 10 reminder.
- **`enabledPlugins`:** `superpowers@claude-plugins-official` (already set). Claude Code users who clone
  the repo are offered it.

### 3.4 Spec-driven flow (superpowers)

brainstorming → spec in `docs/specs/YYYY-MM-DD-<topic>-design.md` → writing-plans → plan in
`docs/plans/YYYY-MM-DD-<topic>.md` → implementation through subagent-driven development, one worktree per
task, with code review before merge. These paths override superpowers' default `docs/superpowers/...`.

### 3.5 Project skills (every agent, not only Claude)

- **Location.** Skills live in `.agents/skills/`, the shared project location read by Codex, Cursor,
  Gemini CLI, GitHub Copilot, OpenCode, Amp, Cline, Zed, Warp and others. They are committed and pinned
  by source commit and hash in `skills-lock.json`. Claude Code reads only `.claude/skills/`, which holds
  relative symlinks into `.agents/skills/`.
- **Project skills (23):** find-skills, tdd, the golang-* subset (how-to, project-layout, design-patterns,
  code-style, naming, error-handling, testing, database, security, lint, continuous-integration,
  dependency-management, observability, documentation), mcp-builder (official, for tool design and evaluations), go-mcp-server-generator, github-actions-hardening,
  turborepo, documentation-and-adrs, supabase-postgres-best-practices, docker-compose-patterns.
- **Superpowers (15 skills)** is in `.agents/skills/` for non-Claude agents. Claude Code gets it from the
  plugin (`enabledPlugins`), which also brings its hooks and agents. Superpowers is therefore **not**
  linked into `.claude/skills/`, so Claude does not see each skill twice.
- **Bootstrapping superpowers in other agents.** AGENTS.md tells non-Claude agents to load
  `using-superpowers` before any task. This replaces the SessionStart hook the plugin gives Claude.
- **Restoring and updating.** Windows users without symlink support, or anyone with a broken checkout,
  run `npx skills experimental_install -a claude-code universal`, then remove any superpowers links it
  adds under `.claude/skills/`. Updates go through `npx skills update`, reviewed in a PR like any dependency.
- **Licenses.** The vendored skills are MIT, except `docker-compose-patterns` and `mcp-builder` (Apache-2.0, with its LICENSE.txt kept in the skill folder).
  `THIRD_PARTY_NOTICES.md` at the root lists each source, its commit and its license.
- **Frontend phase:** add the official `vercel/next.js` skills.

## 4. Quality gates

| Gate | Go (`apps/api`) | TS (later) |
|---|---|---|
| Format | gofmt/goimports check | Biome |
| Lint | `go vet`, golangci-lint (config from the golang-lint skill + forbidigo for `time.Now` in domain/app) | Biome |
| Types | the compiler | `tsc --noEmit`, `strict: true` |
| Tests | `go test -race` unit; integration with Postgres | Vitest |
| Security | govulncheck | `pnpm audit` |
| Generated code fresh | sqlc output, `openapi.json`, docs indexes | api-client types |
| Docs | line limit, index freshness | — |

### 4.1 Definition of Done (in AGENTS.md; the PR template mirrors it)

A task is done only when:
- tests were written first and pass;
- every gate in the table above is clean;
- reuse was checked, nothing exists a third time, and the PR says what was reused or extracted;
- it is committed on a feature branch, and the PR title is a Conventional Commit;
- CI is green and the PR closes an issue (`Closes #N`);
- a **devlog entry** (`docs/devlog/YYYY/YYYY-MM-DD-<slug>.md`) is added, if the PR changes behaviour;
- an **ADR** is added, if the PR makes or changes a decision;
- `docs/architecture/*` and `GLOSSARY.md` are updated, if the PR changes what they describe;
- `ROADMAP.md` is updated, if a phase starts or ends.

CHANGELOG is left entirely to release-please.

- **Local hooks:** lefthook, so one config covers Go and TS. pre-commit formats and lints the
  staged files; commit-msg runs commitlint.
- **CI (GitHub Actions) on every PR:** all gates, integration tests against a Postgres service container,
  the BCB contract test (may skip), and the PR-title check.

## 5. Docs layout: files that grow sideways, not down

```
docs/
├─ README.md            # map of the folders; agents read this first
├─ PROJECT.md           # vision and scope (short, rarely changes)
├─ GLOSSARY.md          # pt-BR → code names (split by domain if it outgrows the limit)
├─ ROADMAP.md           # phases only; detailed backlog = GitHub issues + milestones
├─ architecture/        # living docs, one file per area: README.md (overview + index), http.md, mcp.md,
│                       # persistence.md, money-and-dates.md, logging.md, ...
├─ decisions/           # ADRs, one file each: 0001-go-backend.md, ... ; README.md index is generated
├─ devlog/              # one file per entry: 2026/2026-11-27-project-kickoff.md ; index generated
├─ specs/               # one design per feature
└─ plans/               # one implementation plan per feature
```

- **History is append-only, one file per entry.**
  - An ADR is never rewritten. A new ADR supersedes it, and the old one only gets its status line updated.
  - Each devlog entry is its own file.
  - Two parallel PRs therefore never conflict on docs.
- **Living docs are split by area.** Past about 300 lines a file is split. The limit applies to living
  docs only: `PROJECT`, `GLOSSARY`, `ROADMAP`, `architecture/*` and the READMEs. Specs, plans, ADRs and
  devlog entries are point-in-time documents, one per item, and are exempt.
- **Indexes are generated** by `pnpm docs:index`.
- **CI fails** on an oversize doc or a stale index.
- **The roadmap lives in GitHub issues.** Every PR closes an issue (`Closes #N`), which replaces the source
  project's "tick ROADMAP.md" rule with something automatic. `ROADMAP.md` changes only when a phase changes.

## 6. Open-source setup

### 6.1 Files

- **LICENSE:** MIT, `Copyright (c) 2026 Augusto Castilho Borges`.
- **README:** leads with **"connect Horizon Money to your agent"**, with example conversations, then:
  - the `docker compose up` quick start;
  - exposing the server over HTTPS (domain + reverse proxy, or a tunnel), because cloud agents need it;
  - connector setup for the Claude apps, Claude Code and other MCP clients;
  - optional Google sign-in (creating the OAuth client in Google Cloud) and optional SMTP, with example
    settings for common providers;
  - features, configuration, links to docs, and badges (CI, OpenSSF Scorecard, license).
- **CONTRIBUTING:**
  - dev setup, the TDD and worktree flow, commit and PR rules;
  - how to get superpowers and the project skills;
  - an **AI-assisted contributions policy**: disclose AI use; the human who opens the PR understands it and owns it.
- **CODE_OF_CONDUCT:** the current Contributor Covenant version.
- **SECURITY.md:** report privately through GitHub private vulnerability reporting, never in a public issue.
- **`.github/`:**
  - issue forms (bug, feature) with `config.yml` (blank issues off, questions sent to Discussions);
  - a PR template that mirrors the Definition of Done;
  - CODEOWNERS.
- **Root files:** `.editorconfig`, `.gitattributes`, `.gitignore`, `.env.example`. CHANGELOG.md is managed by release-please.

### 6.2 GitHub repository settings

Applied with `gh` where possible:
- **Merging:** squash merge only, using the PR title; delete branches on merge.
- **Ruleset on `main`:** PR required, required status checks, no force push or deletion, linear history.
- **Community:** Discussions on; labels `good first issue`, `help wanted`, `bug`, `enhancement`, `docs`.
- **Security:** private vulnerability reporting, Dependabot alerts and security updates, secret scanning
  with push protection, CodeQL default setup.

### 6.3 CI security (OpenSSF Scorecard practices)

- Least-privilege `permissions:` in every workflow.
- Actions pinned by commit SHA.
- Dependabot for gomod, npm, github-actions and docker.
- No `pull_request_target` running fork code.
- A Scorecard workflow and badge.
- The github-actions-hardening skill reviews every workflow.

### 6.4 Releases

- SemVer, starting at `v0.x`. release-please builds the version bump and CHANGELOG from Conventional Commits.
- On release, a multi-arch (amd64/arm64) image is published to GHCR with a build-provenance attestation.
- No telemetry.

## 7. Delivery order (Phase 0, before any feature)

1. `git init` and create the public GitHub repository.
2. Monorepo skeleton, Go module, gates, local hooks, CI, `compose.yaml`.
3. AGENTS.md / CLAUDE.md, hooks, docs skeleton, and the generated-index tooling.
4. ADRs for the decisions already taken: Go backend, Postgres + sqlc, hexagonal core, MCP in-process,
   money representation, MIT, Turborepo + pnpm, docs layout, superpowers as the spec flow.
5. Community files and repo settings.

Then the backend features from the companion spec, each through its own plan.

## 8. Design decisions worth knowing

1. `docs/specs` and `docs/plans` instead of superpowers' default `docs/superpowers/...`.
2. lefthook for local hooks; commitlint stays for messages.
3. Each worktree gets its own database on the shared Postgres container.
4. The roadmap lives in GitHub issues, and every PR closes one.
5. A soft limit of about 300 lines per doc file, enforced by CI.
6. pnpm scripts at the root are the single command entry point, with no Makefile.
