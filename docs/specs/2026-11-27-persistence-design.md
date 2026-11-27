# Persistence design (issue #8)

- **Date:** 2026-11-27
- **Status:** Draft
- **Builds on:** `2026-11-27-product-and-backend-design.md` (§4, §6.3, §6.4, §6.10), ADR-0002, ADR-0003,
  ADR-0012, ADR-0013, and the domain package as built (`docs/architecture/domain.md`)
- **Skills consulted:** superpowers:brainstorming, supabase-postgres-best-practices, golang-database,
  golang-testing, golang-continuous-integration, github-actions-hardening, golang-dependency-management,
  golang-how-to

## 1. Intent

`internal/domain` computes balances, forecasts and summaries from values passed in. Nothing stores them
yet. This work gives the domain a durable home:
- a Postgres schema whose constraints are the second lock behind the use cases;
- migrations that ship inside the binary and run on start;
- typed queries;
- repositories that always answer "for this user".

The use cases (#9) then only orchestrate: load, call the domain, save.

It also brings the harness that database work needs: an integration job in CI, a check that generated
query code is fresh, and one database per worktree.

## 2. Goals, non-goals, anti-goals

**Goals**
- A schema for the money data of v0.1:
  - users, with their settings;
  - workspaces and forecast categories;
  - tags;
  - movements: one-offs and series, with exceptions and series groups;
  - transfers and checked days.
- goose migrations embedded in the binary. They run when `serve` starts, and through a `migrate` command.
- sqlc queries over pgx, and repositories behind ports in `internal/port`, every one scoped by user.
- A suite of integration tests against a real Postgres, run in CI.
- The harness items from issue #8.

**Non-goals (owned elsewhere)**
- Passwords, sessions and personal access tokens (#10); Google identities (#11); reset tokens (#12);
  OAuth clients and grants (#13).
- The cache of exchange rates (#15) and idempotency keys (#16).
- Use-case rules such as "D0 cannot move later past data" or "currency is frozen once there is data".
  They live in `internal/app` (#9); the schema only backs them up.
- Pagination shapes and DTOs (#9, #14).

**Anti-goals**
- A query that can return or change another user's row because a filter was forgotten.
- Half a transfer: a transfer is one row, written in one statement.
- A schema that disagrees with the domain. For example, a series column the domain has no field for, or
  a rule enforced differently in SQL than in Go.
- Floats anywhere near money.

## 3. Decisions taken with the maintainer

1. **A workspace with transfers cannot be deleted.** The repository refuses with
   `workspace.has_transfers`, and the client suggests moving the balance to another workspace and
   archiving this one. A workspace without transfers deletes together with its movements, categories
   and checked days.
2. **One `movements` table holds one-offs and series** (approach A below).
3. **The `serve` end-to-end test moves to the integration suite.** `TestServeAnswersHealthcheckAndStopsOnCancel`
   now needs a database, so it runs under the `integration` tag against a throwaway one. A new unit test
   pins that `serve` refuses to start without `DATABASE_URL`, with a clear configuration error. `pnpm test`
   stays Docker-free.
4. **`Integration` becomes a required check right after this work merges**, through `gh api` on the `main`
   ruleset. Doing it earlier would leave every open pull request without the job (Dependabot's included)
   waiting on a check that never reports.

## 4. Approaches considered (movement storage)

| | **A. One `movements` table (chosen)** | B. `movements` + `series` tables | C. One row per occurrence |
|---|---|---|---|
| For | One id space: an occurrence is "movement X on date D". Turning a one-off into a series is an `UPDATE`. One tag join table. | No nullable columns; each table maps to one domain type. | Simple reads. |
| Against | Repetition columns are null on one-offs, so `CHECK` constraints keep each kind coherent. | Two id spaces the API must tell apart. Two tag join tables. Adding a repetition moves the row between tables. | Endless rows for open series. An `all` edit rewrites hundreds of rows. It contradicts §6.3. |

## 5. Schema

Conventions:
- lower-case snake_case names;
- `uuid` primary keys defaulting to `uuidv7()`, which is built into PostgreSQL 18+ and is time-ordered,
  so it needs no extension and causes no index scatter;
- `bigint` minor units for money, `date` for calendar dates, and `timestamptz` for `created_at` and
  `updated_at`. Both default to `now()`, and every update sets `updated_at = now()`;
- `text` with `CHECK` for enumerations.

UUIDs are chosen over `bigint` identities because, on an install shared by several people, sequential ids
reveal how much the others write. Ids are opaque to clients.

**Ownership.** Every owned table carries `user_id`. A child references its parent through a composite key
`(parent_id, user_id) → parent(id, user_id)`, so Postgres itself refuses a row that mixes two users. For
example, a tag of user A cannot land on a movement of user B, even when a use case forgets to check.
Every parent therefore has `unique (id, user_id)`.

### 5.1 Tables

**users**
- `id`
- `email text not null`, unique on `lower(email)`
- `display_name text not null`
- `main_currency text not null default 'BRL'`, checked `~ '^[A-Z]{3}$'`
- `timezone text not null default 'America/Sao_Paulo'`
- `savings_rate_target_min integer not null default 2000` and `savings_rate_target_max integer not null
  default 3000`, in basis points, with `0 ≤ min ≤ max ≤ 10000`
- `created_at`, `updated_at`
- #10 adds the credential columns in its own migration.

**workspaces**
- `id`, `user_id → users on delete cascade`
- `name text not null` (non-empty)
- `currency` (same check as `main_currency`)
- `initial_balance bigint not null`
- `initial_balance_date date not null` (D0)
- `low_balance_threshold bigint not null default 0`
- `forecast_divisor smallint not null default 30`, between 28 and 31
- `archived_at timestamptz` (null means active)
- `created_at`, `updated_at`
- Index on `(user_id)`.

**forecast_categories**
- `id`, `user_id`, `workspace_id`, with `(workspace_id, user_id) → workspaces on delete cascade`
- `name text not null`
- `monthly_amount bigint not null > 0`
- `created_at`
- Index on `(workspace_id)`.

**tags**
- `id`, `user_id → users on delete cascade`
- `name text not null`, 1–30 characters, unique on `(user_id, lower(name))`
- `color text not null`, one of gray, blue, yellow, green, red, purple, pink or brown
- `created_at`

**movements**
- `id`, `user_id`, `workspace_id`, with `(workspace_id, user_id) → workspaces on delete cascade`
- `type text not null`: income, expense, daily, saving or card
- `amount bigint not null > 0`: a one-off's amount, a series' amount per occurrence, or the part amount of
  installments
- `description text`, at most 100 characters
- `date date not null`: a one-off's date, or a series' first occurrence (`Series.Start`)
- Repetition columns, all null on a one-off:
  - `frequency text`: daily, weekly, monthly or installments;
  - `end_date date`, which must be ≥ `date`;
  - `anchor_day smallint`, 1–31;
  - `last_amount bigint > 0`, `parts smallint` (2–360) and `first_part smallint` (1 ≤ first_part ≤ parts),
    for installments only;
  - `series_group_id uuid`.
- `created_at`, `updated_at`
- One `CHECK` per kind keeps the row coherent:
  - one-off: every repetition column is null;
  - daily, weekly or monthly: `anchor_day` and `series_group_id` are set, and the installment columns are
    null;
  - installments: `anchor_day`, `last_amount`, `parts`, `first_part` and `series_group_id` are all set.
- A new series takes its own id as its `series_group_id`. The tail of a `following` split copies the
  head's group.
- Indexes: `(workspace_id, date)`; a partial index on `(workspace_id)` where `frequency is not null`;
  and `(series_group_id)` where it is not null.

**movement_tags**
- `(movement_id, user_id) → movements on delete cascade` and `(tag_id, user_id) → tags on delete cascade`
- Primary key `(movement_id, tag_id)`, and an index on `(tag_id)`.

**movement_exceptions**: one row per changed occurrence of a series.
- `(movement_id, user_id) → movements on delete cascade`
- `original_date date not null` (the date the series produced), with primary key
  `(movement_id, original_date)`
- `deleted boolean not null default false`
- The override snapshot, all null when `deleted`:
  - `date date` and `amount bigint > 0`, both set when not deleted;
  - `description text` (at most 100 characters).
- `created_at`, `updated_at`
- The override's tags live in **movement_exception_tags**: `(movement_id, original_date, user_id)` → the
  exception (which therefore has `unique (movement_id, original_date, user_id)`), plus
  `(tag_id, user_id) → tags`, all on delete cascade.
- **An override is a full snapshot** (ADR-0012, Decision 5). A `this` edit copies the series' current
  description and tags into the exception and then applies the change. Null description means "none" and
  no tag rows mean "no tags", never "inherit", so a later `all` edit cannot reach the occurrence.
- **The type is the series'.** The schema has no type on an exception, and the domain has none either. A
  `this` edit that changes the type becomes, in one transaction (#9), a deletion exception plus a new
  one-off with that type. *(Builder's call.)*

**transfers**
- `id`, `user_id → users on delete cascade`
- `from_workspace_id` and `to_workspace_id`, each `(…, user_id) → workspaces`, with
  `from_workspace_id <> to_workspace_id`
- `amount_sent bigint > 0` and `amount_received bigint > 0`, each in its workspace's currency
- `date date not null`
- `description text`, at most 100 characters
- `created_at`, `updated_at`
- Indexes on `(from_workspace_id, date)` and `(to_workspace_id, date)`.

The workspace keys use the default `NO ACTION`, not `RESTRICT`. `NO ACTION` is checked at the end of the
statement, so deleting a user still cascades cleanly through its transfers and workspaces. Deleting a
single workspace that has transfers fails, and the repository maps that failure to
`workspace.has_transfers` (Decision 1).

**checked_days**
- `(workspace_id, user_id) → workspaces on delete cascade`
- `date date not null`, with primary key `(workspace_id, date)`

### 5.2 Mapping to the domain

A movement row with `frequency` null is a one-off and becomes a `domain.Entry`. Any other row is a
`domain.Series`, field for field:
- `date` → `Start`, and `end_date` → `End`;
- `anchor_day` → `AnchorDay`, `amount` → `Amount`, and `last_amount` → `LastAmount`;
- `parts` → `Parts`, and `first_part` → `FirstPart`.

Its `movement_exceptions` become the `map[Date]Exception` that `Expand` takes. Rows are rebuilt with a
validating constructor (a `Series.Validate`, which the domain review deferred to #8), so a corrupt row
surfaces as an error instead of a wrong balance.

Ids are a new `domain.ID` value type: 16 bytes, with `ParseID` and `String`. A malformed id fails to parse
with `id.invalid`, so the doors return 400, never 500 (golden rule 3). It adds no dependency.

## 6. Code layout

```
apps/api/
├─ db/
│  ├─ migrations/*.sql        goose files, up and down
│  ├─ queries/*.sql           sqlc input
│  └─ embed.go                package db: exposes the migrations as an embed.FS
├─ sqlc.yaml
└─ internal/
   ├─ port/                   repository interfaces, Store, port errors
   └─ adapter/postgres/
      ├─ sqlcgen/             generated by sqlc, committed, never edited
      ├─ migrate.go           runs goose on a pool
      ├─ store.go             pgxpool, InTx, error mapping
      └─ <entity>.go          one repository per aggregate
```

- **Ports.** There is one repository interface per aggregate: users, workspaces (with forecast
  categories), tags, movements (with exceptions and tags), transfers and checked days.
  - Every method takes the caller's user id first and filters by it.
  - A row of another user is indistinguishable from a missing one: `port.ErrNotFound`.
  - `Store.InTx(ctx, func(Repositories) error)` runs several repository calls in one transaction. #9
    needs it for `following` splits and `all` edits; one transfer needs no transaction, since it is one row.
- **The ledger read.** For one workspace up to a date, the movement repository returns:
  - its one-offs up to that date;
  - every series that starts on or before the date and has not ended before D0, plus every series with a
    non-deleted exception whose override date falls between D0 and the date, each with all of its
    exceptions (an override may move an occurrence across the series' own bounds); *(builder's call)*
  - its transfer legs.
- **Errors.**
  - The adapter maps Postgres errors to port errors: unique violation on a tag name →
    `tag.name_taken`; the workspace delete above → `workspace.has_transfers`; no row → `ErrNotFound`.
  - Any other database failure is logged once by the repository, at `error`, as
    `postgres.<entity>.<operation>.failed` with ids only, and returned wrapped. Callers do not log it
    again.
  - Business denials are not logged here; the use case logs them at `warn`.
- **Dependencies.**
  - Go modules: `github.com/jackc/pgx/v5` and `github.com/pressly/goose/v3` (ADR-0002). Goose runs over
    pgx's `database/sql` driver.
  - sqlc needs cgo, so it does not join `go.mod`. `pnpm gen` runs a pinned `sqlc/sqlc` Docker image, and
    CI runs the same image.
- **Configuration.** `DATABASE_URL` joins `config.Config`. `serve` and `migrate` require it;
  `healthcheck` does not.
- **Startup.** `serve` opens the pool, runs the migrations, then listens. A migration failure stops the
  process with a clear error, so the server never runs on a half-migrated schema.

## 7. Testing (strict TDD)

Integration tests carry the `integration` build tag and need `TEST_DATABASE_URL`, an admin connection to
a Postgres such as the compose one. Without it they skip, and `pnpm test` stays Docker-free.
- Each test package creates a throwaway database `horizon_test_<random>` in `TestMain`, migrates it, and
  drops it at the end.
- Inside a package, every test creates its own users. Data is isolated by user, so tests run in parallel
  with no truncation.

**Proposed seams (to confirm before the first test):**

| Seam | Catches | Leaves out |
|---|---|---|
| Migrations: up from empty, down to empty, up again | broken or irreversible SQL, missing `down` | data migrations (none yet) |
| Each repository through its port interface | round-trips with the domain types, scoping, error mapping, constraints | HTTP/MCP mapping (#9) |
| Cross-user access on every repository method | a forgotten `user_id` filter or composite key | — |
| The ledger read feeding `domain.Ledger` | the right one-offs, series and exceptions reach the domain; the §4.14 fixture stored and read back gives the same balances | the use-case wiring (#9) |
| `pnpm db:worktree` (node test) | slug validation, database name, the printed URL | Docker itself |
| `serve` (Decision 3) | unit: no `DATABASE_URL` is a configuration error; integration: it migrates, answers `/healthz` and stops on cancel | — |

Constraint tests use the repositories where the domain could let a bad value through. They also write raw
SQL where only the second lock stands: a tag of another user on a movement, a transfer between the same
workspace twice, and a series row missing its anchor day.

## 8. Harness (from issue #8)

- **CI job `Integration`** in `.github/workflows/ci.yml`:
  - a `postgres:19-alpine` service container with a health check;
  - `TEST_DATABASE_URL` set from the service;
  - `go test -race -count=1 -shuffle=on -tags=integration ./...`;
  - `permissions: contents: read`, actions pinned by SHA, `persist-credentials: false`.

  It joins the required checks of the `main` ruleset right after the merge (Decision 4). Issue #8 says
  Postgres 18; ADR-0013 moved to 19.
- **Generated code check:** CI runs the pinned sqlc image (`generate`), then `git diff --exit-code`.
- **`pnpm db:worktree <slug>`** (`scripts/db-worktree.mjs`):
  - it accepts `[a-z0-9-]+`;
  - it creates `horizon_<slug>` (dashes become underscores) on the compose Postgres if it is missing;
  - it runs `horizon-money migrate` against it;
  - it prints the `DATABASE_URL` line for the worktree's `.env`.

  AGENTS.md rule 5 points to it.
- **Local scripts:** `pnpm test:integration` runs the integration suite, and `pnpm gen` runs sqlc. The
  integration line in `.env.example` gets a default `TEST_DATABASE_URL` for the compose Postgres.

## 9. Left to the builder

- Exact query names, the repository method list beyond what §6 and the plan's tests need, and the Go shape
  of the entity records (`domain.Workspace`, `domain.Movement`, …).
- How the migrations are split into files.
- The sqlc options, such as pointer types for nullable columns.
- The pool settings.
- The sqlc image version, which is pinned by tag and digest.
- Whether `down` migrations live in the same file as `up` (goose annotations).

The plan records each of these as it is made, and any that changes a public behaviour gets an ADR.
