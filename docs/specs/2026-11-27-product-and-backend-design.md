# Horizon Money — Product rules and backend design (v1)

- **Date:** 2026-11-27
- **Status:** Draft
- **Companion spec:** `2026-11-27-repository-and-harness-design.md` (monorepo, agent harness, docs layout, open-source setup)
- **Skills consulted:** superpowers:brainstorming, golang-project-layout, golang-design-patterns (hexagonal reference), supabase-postgres-best-practices, turborepo, documentation-and-adrs, find-skills, mcp-builder

## 1. Intent

Horizon Money is an **open-source, self-hosted** personal-finance forecasting app whose main face is an
**MCP connector**. You plug it into the AI agent you already use (Claude, ChatGPT, Cursor and others),
then talk to it: "I spent 30 on lunch", "how much will I have on the 15th?", "close my month". You can
log income, fixed expenses, daily spending, savings and card spending. The app projects your balance day
by day for the coming months, so you see in advance how much money you will have and whether you are
spending too much.

Horizon Money is not an agent and runs no AI model. It is what lets **your** agent understand **your**
money. On top of that core it offers:

- **Several currencies.** You can keep separate balances in, say, BRL, USD and CAD.
- **A web app.** v1 ships only the account pages the connector needs (sign-up, login, consent, onboarding).
  The full Next.js app comes later and consumes the HTTP API.

This spec covers the domain rules and the backend, including those account pages.

## 2. Goals, non-goals, anti-goals

**Goals (v1)**
- The core feature set (§4): five movement types, recurrence and installments, tags,
  daily forecast, day-by-day balance, multi-month horizon, monthly and yearly calculations, "checked day".
- One profile owns several **workspaces**, each with its own currency and a single balance; manual
  transfers between workspaces.
- Daily reference exchange rate from the Banco Central do Brasil (PTAX) to pre-fill transfers and to
  build a consolidated view.
- One application core exposed through two doors: **HTTP** (frontend) and **MCP** (agents).
- **Connect from any MCP client:**
  - OAuth 2.1 authorization, so Horizon Money works as a connector in the Claude apps (web, desktop,
    mobile) and other clients that require OAuth;
  - personal access tokens for developer tools and scripts.
- **Accounts:** email + password, or Google (optional per install). Server-rendered account pages:
  sign-up, login, OAuth consent, onboarding, and "connect your agent".
- **Optional email over SMTP** for password resets.
- Runs with `docker compose up`.

**Non-goals (v1)**
- Bank connectors, automatic imports and history imports.
- Exchange providers (Wise, Revolut, exchange houses) and rate syncing.
- A credit-card registry (closing day, due day, invoices).
- Holidays in the calendar (weekends only).
- Email verification at sign-up and any email other than password resets.
- A built-in AI model or chat; the intelligence comes from the user's own agent.
- Recurring transfers.
- The full web app beyond the account pages.

**Anti-goals** (failure even if it technically works)
- HTTP and MCP giving different numbers for the same question.
- Cent errors (floating point) or rounding that differs from §4.
- The app breaking when the BCB API is down. Only transfer pre-fill and the consolidated view may degrade.

## 3. Constraints

- Open source (MIT) and self-hosted. One install may serve several people, and each person's data is isolated.
- Code, identifiers, API fields, docs and commits are in English. User-facing text is translated by the
  frontend; the API returns error **codes**.
- Decided stack (ADR candidates): Go backend, Next.js frontend later, Postgres, Turborepo + pnpm monorepo.

## 4. Domain rules

### 4.1 Glossary (Portuguese term → code name)

| pt-BR | code | Meaning |
|---|---|---|
| perfil | `user` | A person with a login |
| workspace | `workspace` | One balance in one currency |
| entrada | `income` | Money in |
| saída | `expense` | Fixed costs, bills, rent |
| diário | `daily` | Variable day-to-day spending |
| economia | `saving` | Money set aside (reserve, investment); leaves the balance |
| gasto com cartão | `card` | A card purchase or a whole card bill |
| transferência | `transfer` | Money moved between two workspaces of the same user |
| previsão de diário | `daily forecast` | Expected daily spending used for future days |
| dia conferido | `checked day` | Manual "I reviewed this day" mark |
| horizonte de saldos | `horizon` | Day-by-day projected balance over several months |
| performance | `performance` | Month's net result |
| economizado | `savings rate` | Savings ÷ income |
| custo de vida | `cost of living` | What it costs to live, excluding savings |
| diário médio | `average daily` | Average daily spending |

### 4.2 User and sign-in

- Display name and email.
- **Sign-in methods:** email + password, and/or Google.
  - Google appears only when the install sets `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`.
- **Account linking:** a Google sign-in whose email matches an existing password account does **not**
  merge automatically, because sign-up emails are not verified. The user links Google from their profile
  while signed in. This closes the "someone registered my email first" takeover.
- **Forgotten password:**
  - with SMTP configured: a reset link by email. The token is single-use, valid for 30 minutes and stored
    hashed. The response never reveals whether the email exists.
  - always available: the server command `horizon-money users reset-password <email>`, run by whoever
    hosts the install.
  - Google users need no password, and a signed-in user can change theirs in the profile.
- `main_currency` (default `BRL`): currency of the consolidated view.
- `timezone` (IANA, default `America/Sao_Paulo`): defines "today" and the midnight at which the forecast
  for a day disappears.
- `savings_rate_target_min` / `_max` (defaults 20% / 30%): the "ideal" band for the savings rate.
  It is a user setting, not a constant.

### 4.3 Workspace

- `name`, `currency` (ISO 4217; must be BRL or a currency the rate provider supports),
  `initial_balance`, `initial_balance_date` (D0), `low_balance_threshold` (default 0),
  `forecast_divisor` (integer 28–31, default 30).
- **One balance.** `initial_balance` is the balance at the start of D0.
- **Start from zero.** There is no history import; a workspace begins on D0 with that day's balance.
  - One-off movements dated before D0 are rejected.
  - A **series may start before D0**, and only its occurrences on or after D0 count. For example, an
    August 12x purchase entered with D0 = Nov 1 shows parts 4/12 … 12/12, and rent that began years ago
    is entered with its real start date.
  - Moving D0 **earlier** is always allowed.
  - Moving D0 **later** is refused (code `workspace.data_before_start_date`, listing the offending items)
    while any one-off movement or transfer would fall before it.
- `currency` cannot change once the workspace has movements or transfers.
- A workspace can be archived. Archived workspaces leave the consolidated view and lists by default.

### 4.4 Tags

- `name` (1–30 chars, unique per user, case-insensitive), `color` ∈ {gray, blue, yellow, green, red, purple, pink, brown}.
- Belong to the **user**, usable in any workspace and on any movement type. A tag is a label only and never changes a calculation.
- No count limit.

### 4.5 Movements

- `type` ∈ {income, expense, daily, saving, card}; `amount` > 0 (the type gives the sign: income adds, all others subtract);
  `description` (optional, ≤ 100 chars); `date` (a calendar date, no time); `tags` (0..n).
- **Repetition** ∈ {none, daily, weekly, monthly, installments(N)}.
  - daily/weekly/monthly repeat the same amount with no end by default. An end date is optional.
    A monthly series anchored on day 29–31 falls on the month's last day in shorter months.
  - installments(N), 2 ≤ N ≤ 360: the total is split into N monthly parts. Each part is the total ÷ N
    rounded down to the currency's minor unit, and the last part takes the remainder, so the parts always
    sum exactly to the total. Each occurrence knows its index ("3/12").
- **Edit or delete an occurrence of a series** with a scope:
  - `this`: only that occurrence.
  - `following`: that occurrence and the ones after it. The old series ends the day before, and a new series starts with the new values.
  - `all`: the whole series. Occurrences edited individually before keep their individual edit.
  - A `following` split keeps both halves in one **series group**, so a later `all` edit or delete from
    any occurrence reaches every part of the purchase or recurrence it came from.
  - **Installments under edits.** The new half continues the numbering (5/12 … 12/12). An edited amount
    is the value **per part** from that occurrence on, not a new total. The total is never stored as truth:
    it is the sum of the current parts. "Parts sum to the total" is checked at creation. After edits, the
    total is simply whatever the parts now sum to.
- **Card** is just a type. It leaves the balance on the date it was logged. Logging individual purchases or the whole bill is the user's choice.

### 4.6 Transfers

- Between two **different workspaces of the same user**, same or different currency.
- Fields: `from_workspace`, `to_workspace`, `amount_sent` (source currency), `amount_received`
  (destination currency), `date`, `description` (optional).
- **Suggestion:** given source, destination and amount sent, the API returns the amount at the latest PTAX rate.
  The user confirms or adjusts it. The stored `amount_received` is the real one.
- Effective rate = received ÷ sent. It is derived and never stored.
- Moves both balances. **Never** counts as income, expense, saving, card, cost of living or savings rate.
  Performance shows it on a separate line.
- No repetition in v1. No tags in v1.

### 4.7 Daily forecast (per workspace)

- Categories: `name` + `monthly_amount`.
- `daily_value = round_half_up(sum(monthly_amount) ÷ forecast_divisor)` to the minor unit.
  For example, 1,000.00 ÷ 30 = 33.33.
- The forecast applies to every day `d` with `d > today` (user's timezone) and `d ≥ D0`. Today and
  past days never carry it: at midnight the day's forecast disappears, and only what was logged counts.
  A past day with nothing logged counts as zero spending.
- There is one current configuration, with no history per month. Changing it only affects future days,
  which are the only days where it shows.

### 4.8 Checked day

A manual mark per (workspace, date). It is **display-only** and never changes a calculation.

### 4.9 Daily balance

```
end(D0 - 1) = initial_balance
end(d) = end(d-1) + income(d) - expense(d) - daily(d) - saving(d) - card(d)
         + transfers_in(d) - transfers_out(d) - forecast(d)
```

Per day the API also returns:
- the totals per type;
- `forecast(d)`;
- `checked`;
- `is_weekend` (Saturday or Sunday, from the date alone);
- `status`: `negative` if the balance is below 0, `low` if it is between 0 and `low_balance_threshold`, `ok` otherwise.

The **horizon** is the same series over a range of months (default: the current month and the next two).

### 4.10 Monthly calculations (per workspace, month M)

Sums are over movements dated in M. `forecast_remaining(M)` = `daily_value` × the number of days in M
with `d > today` and `d ≥ D0`.

| Name | Formula | Label |
|---|---|---|
| performance | income − (expense + daily + saving + card + forecast_remaining); transfers in/out shown beside it | ≥ 0 "money left", < 0 "short" |
| savings rate | saving ÷ income (null when income = 0) | below / within / above the user's target band (inclusive bounds) |
| cost of living | expense + daily + card + forecast_remaining | ≤ income "within income", else "above income" |
| average daily | daily ÷ elapsed days | compared with `daily_value` |

For average daily, the elapsed days are:
- **current month:** days from max(month start, D0) through today, inclusive;
- **past months:** the days of the month on or after D0;
- **future months:** null.

The monthly view also lists the totals per type and `forecast_remaining` with its day count.

**Invariant (tested):** end(last day of M) − end(last day of M−1) = performance(M) + transfers_in(M) − transfers_out(M).

### 4.11 Yearly view

Each of the four calculations month by month for the 12 months. Future months are projections and
include the forecast. Year totals:
- performance: the sum;
- savings rate: Σ saving ÷ Σ income;
- cost of living: the sum;
- average daily: Σ daily ÷ Σ elapsed days.

### 4.12 Queries the API must answer

- Balance table of a month, optionally filtered to one type (§4.9).
- Day detail: the day's movements and transfers, optional type filter, plus the forecast line on future days.
- Movements in a date range, filterable by type and tag.
- Per-tag totals for a workspace and month, split by type. A movement with several tags counts under each.
- Monthly summary (§4.10), yearly summary (§4.11), horizon.
- **Consolidated view:** for each non-archived workspace, today's end balance in its own currency,
  converted to `main_currency` with the rate and the rate's date. Plus the total. A workspace with no
  available rate is listed apart and left out of the total.

### 4.13 Exchange rates (PTAX)

- Source: BCB Olinda PTAX API, closing bulletin (`Fechamento`). The selling rate (`cotacaoVenda`) is
  used for conversions.
- Cross rates go through BRL: X→Y = amount × rate(X→BRL) ÷ rate(Y→BRL), with BRL = 1.
- Weekends, holidays or outages use the latest available bulletin. Responses always state the rate's date.
- Rates are cached in Postgres per (currency, date). The app never invents a rate (e.g. 1) when none exists.
- The source sits behind a `RateProvider` port, so other providers (e.g. ECB) can be contributed.
- The list of supported currencies = BRL + the provider's currency list, exposed by the API.

### 4.13b Onboarding and connecting an agent

1. **Create the account:** email + password, or Google.
2. **Create the first workspace:** name, currency, today's balance. This is one form. D0 = today.
   Forecast categories are optional and can be set later, also by the agent.
3. **"Connect your agent":** the MCP URL (`<APP_BASE_URL>/mcp`) with copy-paste steps for the Claude apps,
   Claude Code and other clients, plus a button to create a personal access token for tools that need one.

Adding the connector in an OAuth client opens the consent page ("<client> wants to access your
Horizon Money"). It shows the client name, the redirect domain and the access requested. The user signs
in if needed and approves. Connected clients are listed in the profile and can be revoked.

### 4.14 Acceptance fixture

The first end-to-end test of the domain. All amounts are fictional.

Today = 2026-11-27, timezone America/Sao_Paulo. Workspace in BRL, D0 = 2026-11-01, initial balance 250.00.
The forecast categories are 350 + 300 + 600 = 1,250.00 with divisor 30, so the daily value is
41.67 (41.666… rounded half-up).

| Day | Movements |
|---|---|
| 1 | daily 23.40 |
| 2 | daily 8.90 |
| 3 | income 600.00 · expense 18.50, 42.75, 96.30 |
| 4 | daily 35.20 |
| 5 | daily 52.10 |
| 6 | income 6,400.00 · saving 1,900.00 · card 2,135.80 · expense 1,450.00 |
| 7 | daily 64.60 |

Expected end balances:
- November: 1 → 226.60, 2 → 217.70, 3 → 660.15, 4 → 624.95, 5 → 572.85, 6 → 1,487.05,
  7…27 → 1,422.45, 28 → 1,380.78, 29 → 1,339.11, 30 → 1,297.44.
- December: 1 → 1,255.77, 2 → 1,214.10.

Expected November summary:

| Line | Value |
|---|---|
| income | 7,000.00 |
| expense | 1,607.55 |
| daily | 184.20 |
| saving | 1,900.00 |
| card | 2,135.80 |
| forecast remaining | 125.01 (3 days × 41.67) |
| performance | 1,047.44 ("money left") |
| savings rate | 27.14% ("within" the 20–30% band) |
| cost of living | 4,052.56 ("within income") |
| average daily | 6.82 (184.20 ÷ 27) |

The §4.10 invariant also holds here: 1,297.44 − 250.00 = 1,047.44.

## 5. Approaches considered

| | A. TypeScript everywhere (Bun + Hono) | **B. Go backend + Next.js frontend (chosen)** | C. Python (FastAPI) + Next.js |
|---|---|---|---|
| For | One language; schemas shared by API, MCP and frontend | Single small binary, easy self-hosting; strict, simple language makes AI-written code easy to review; slog and net/http in stdlib; official Go MCP SDK | Mature MCP SDK |
| Against | Bun adds runtime risk; shared types were its only real edge | Two languages; frontend types come from generated OpenAPI | Two languages with neither Go's deployment story nor TS's shared types |

B was chosen. The gap in shared types is closed by generating OpenAPI from the Go handlers and TS
types from it (companion spec).

## 6. Architecture

### 6.1 Shape: hexagonal (ports and adapters)

```
apps/api/
├─ cmd/server/main.go        # wiring only: config, adapters, services, HTTP + MCP on one server
├─ internal/
│  ├─ domain/                # pure rules: money, dates, recurrence expansion, forecast,
│  │                         # balance series, summaries. No I/O, no time.Now()
│  ├─ app/                   # use cases (primary ports). Each authorizes and calls domain + secondary ports
│  ├─ port/                  # secondary ports: repositories, RateProvider, Clock, IdGenerator
│  └─ adapter/
│     ├─ http/               # Huma handlers (driving)
│     ├─ mcp/                # MCP tools and prompts (driving)
│     ├─ web/                # server-rendered account pages: sign-up, login, consent, onboarding (driving)
│     ├─ oauth/              # OAuth 2.1 authorization server endpoints (driving)
│     ├─ postgres/           # repositories over sqlc-generated code (driven)
│     ├─ bcb/                # PTAX client (driven)
│     ├─ google/             # Google OpenID Connect sign-in (driven)
│     └─ smtp/               # email sender (driven; a no-op sender when SMTP is not configured)
├─ db/migrations/            # goose SQL migrations
├─ db/queries/               # sqlc query files
└─ openapi.json              # generated, committed, checked in CI
```

- **Dependency injection:** manual constructor injection in `cmd/server`, with no DI library.
- **HTTP and MCP are thin.** They parse and validate the input, call one use case, and map the result to a DTO.
  A business rule never lives in a handler or a tool.
- **Parity:** every use case is exposed over both HTTP and MCP, unless a written reason says otherwise.
  A test checks that both return the same numbers for the same query.

### 6.2 Money and dates

- Amounts are `int64` minor units in the domain and in Postgres (`bigint`).
- The minor-unit scale comes from ISO 4217 data (e.g. `golang.org/x/text/currency`). BRL, USD and CAD have 2 decimals; JPY has 0.
- **At the API edge (HTTP and MCP), amounts are decimal strings** (`"26.00"`) validated against the
  currency's scale. Decimal strings are less error-prone for LLM agents than minor units.
- Rounding lives in one place in `domain`: half-up for the daily forecast, floor plus remainder for installments.
- Dates are calendar dates (`date` in Postgres). "Today" comes from a `Clock` port in the user's timezone,
  so tests can pin it.

### 6.3 Recurrence storage (recommended; another model is acceptable if the §4.5 semantics hold)

- A series row stores the template (type, amount, description, tags), the frequency, the start date, an
  optional end date and the installment count.
- Exception rows store, per occurrence date, either an override or a deletion.
- Occurrences are expanded on read for the requested range. This is the calendar (RFC 5545-like) model,
  and the data volume of one person makes on-the-fly expansion cheap.
- `following` splits the series. Both halves share a `series_group_id`, and installment halves carry
  their part offset. `all` edits every template in the group.

### 6.4 Persistence

- PostgreSQL 18 with pgx (driver) and sqlc (typed Go generated from SQL, no ORM).
- goose migrations, embedded in the binary and run on startup.
- Use `bigint` identities or UUIDv7 for ids (builder's call).
- Every query on owned data is scoped by `user_id`.
- Constraints act as the second lock: amount > 0, FKs, unique tag name per user, divisor 28–31,
  transfer workspaces must differ.
- A transfer is one row with both sides, written in one transaction. A half transfer is impossible.

### 6.5 HTTP API

- Huma on top of `net/http` (Go 1.22+ routing). Base path `/api/v1`, JSON.
- OpenAPI 3.1 is served at `/api/openapi.json` and docs at `/api/docs`.
- Errors use RFC 9457 problem details with a stable `code` extension (e.g. `movement.amount_not_positive`) that the frontend translates.
- Malformed ids return 400, ids that are not the caller's return 404, and neither ever returns 500.
- Suggested resources (exact paths are the builder's):
  - Auth: `/auth/signup|login|logout`, `/me`, `/me/tokens`.
  - Workspaces: `/workspaces`, plus forecast categories under each workspace.
  - Tags: `/tags`.
  - Movements: `/workspaces/{id}/movements` and `/movements/{id}?scope=`.
  - Transfers: `/transfers` and `/transfers/suggestion`.
  - Checked days: `PUT|DELETE /workspaces/{id}/days/{date}/check`.
  - Read views: `/workspaces/{id}/days?month=`, `/workspaces/{id}/days/{date}`,
    `/workspaces/{id}/horizon`, `/workspaces/{id}/months/{yyyy-mm}`, `/workspaces/{id}/years/{yyyy}`,
    `/workspaces/{id}/tags-summary?month=`, `/consolidated`.
  - Reference data: `/currencies`, `/rates`.
- **Idempotency:** `POST` creates accept an `Idempotency-Key` header. A repeat with the same key and user
  within 24 h returns the first result without creating anything.

### 6.6 MCP server

- Official `github.com/modelcontextprotocol/go-sdk`, Streamable HTTP, mounted at `/mcp` on the same
  server. Stateless.
- **Auth:** `Authorization: Bearer <token>`, where the token is either an OAuth access token (§6.7) or a
  personal access token. An unauthenticated request gets `401` with a `WWW-Authenticate` header pointing
  at the protected-resource metadata, so clients can discover the OAuth flow.
- **Reachability:** cloud clients such as the Claude apps call the server from the internet, so the install
  needs a public HTTPS URL. The README covers a domain with a reverse proxy, and tunnels such as Cloudflare
  Tunnel or Tailscale Funnel. Local clients (Claude Code, Cursor) also work on `localhost` with a token.
- **Prompts:** ready-made MCP prompts for common jobs, such as "close my month", "can I afford X?" and
  "log today's spending". They are as much a part of the product as the tools.
- Tools map 1:1 to use cases: list/get workspaces, month summary, year summary, balance table, day detail,
  horizon, list/create/update/delete movement (with scope), create transfer, transfer suggestion,
  list/create tags, get/set forecast categories, set checked day, consolidated view.
- **Tool design** (mcp-builder guidance):
  - Names are action-oriented with one prefix (`horizon_list_workspaces`, `horizon_create_movement`,
    `horizon_get_month_summary`, …).
  - Descriptions are concise and in English, with the pt-BR terms in parentheses, so agents understand
    "diário", "previsão" and the rest.
  - Every tool declares an output schema and returns structured content plus a short text summary.
  - List tools paginate and filter, so a response never floods the agent's context.
  - Errors are actionable: they name the problem and the next step, e.g. "date is before the workspace
    start date 2026-11-01; use a date on or after it".
  - Annotations: `readOnlyHint` on reads, `destructiveHint` on deletes, `idempotentHint` where true,
    `openWorldHint: false` everywhere except the rate lookup.
- Create tools take an optional `idempotency_key`.
- Responses are compact DTOs, because agent context is expensive.

### 6.7 Authentication and security

- Passwords are hashed with argon2id.
- Sessions are opaque random tokens stored hashed in Postgres and sent in an `HttpOnly`, `SameSite=Lax`
  cookie. `Secure` is on unless `COOKIE_SECURE=false` (local dev).
- Cross-site request protection uses Go's `http.CrossOriginProtection`.
- Personal access tokens: named, shown once, stored as SHA-256, revocable, with `last_used_at`.
  They are used by developer tools and scripts.
- **OAuth 2.1 authorization server**, built into the binary, for MCP clients:
  - Discovery metadata: protected resource (RFC 9728) and authorization server (RFC 8414).
  - Client registration: Client ID Metadata Documents, which the current MCP spec prefers, **and**
    Dynamic Client Registration (RFC 7591), which the Claude apps use today.
  - Authorization code flow with PKCE `S256` required. Public clients use `token_endpoint_auth_method: none`.
  - Short-lived access tokens. Refresh tokens rotate on every use, and reuse revokes the whole family.
  - Tokens are bound to the MCP resource (`resource` parameter / audience).
  - Exact redirect-URI matching.
  - Consent is recorded per user and client, and can be revoked from the profile.
  - The builder targets the MCP authorization spec revision supported by the Go SDK at plan time, and tests
    against the Claude apps' connector flow.
- **Google sign-in:** OpenID Connect authorization code flow with PKCE, `state` and `nonce`. Only a
  verified Google email is accepted.
- **Account pages:** server-rendered (`html/template`) with no frontend framework, in pt-BR and en, under a
  strict Content-Security-Policy. Forms are CSRF-protected.
- **Email:** SMTP is configured through `SMTP_*` env vars and works with any provider (Gmail app password,
  Resend, Postmark, Brevo, Amazon SES, a self-run server). Without SMTP, email features turn off quietly
  and the reset command remains. Emails are plain text + HTML, in the user's language.
- `SIGNUP_ENABLED` (env, default `true`). The README tells self-hosters to turn it off after creating their accounts.
- Login, sign-up, password reset and token endpoints are rate-limited per IP and per account.
- **DTOs only:** a database row never leaves as-is. A test pins that password and token hashes are never serialized.

### 6.8 Logging

- `log/slog` with a JSON handler on stdout, one line per event.
- Events are named `domain.function.what_happened`.
- Levels: `error` for unexpected infra failures, `warn` for validation, permission and business denials, `info` for successful state changes.
- A request id runs through HTTP and MCP.
- **Never logged:** secrets, tokens, passwords, **descriptions or amounts** (personal financial data). Ids and codes only.

### 6.9 Configuration

Environment only (12-factor):

| Group | Variables | Notes |
|---|---|---|
| Core | `DATABASE_URL`, `PORT`, `LOG_LEVEL` | |
| Public URL | `APP_BASE_URL` | the public HTTPS URL; also the OAuth issuer and the base of the MCP URL |
| Sign-up and cookies | `SIGNUP_ENABLED`, `COOKIE_SECURE` | |
| Google (optional) | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | |
| Email (optional) | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS` | |
| Rates | `BCB_BASE_URL` | overridable for tests |

User-editable business parameters (§4.2, §4.3) are typed columns with defaults, never key-value.

### 6.10 Testing (strict TDD; see the `tdd` skill)

- **Domain:** table-driven unit tests, no database. They cover §4.14, the §4.10 invariant, installment
  sums, month-end anchoring, the midnight forecast cut-off, rounding and cross-rate conversion.
- **Use cases through HTTP:** `httptest` against a real Postgres, using a throwaway database per test run.
- **MCP parity:** an SDK client calls the same queries as HTTP and compares the results.
- **BCB contract test:** hits the live PTAX API and skips when offline. Recorded fixtures cover parsing.
- **Security:** cross-user access attempts return 404; the DTO leak test from §6.7.
- **OAuth:** the full flow runs end to end with a test client: discovery, registration (CIMD and DCR),
  PKCE, consent, token, refresh rotation and reuse detection, revocation. The MCP endpoint rejects tokens
  minted for another audience.
- **Email:** integration tests send through a Mailpit container and read the message from its API.
- **MCP evaluations** (mcp-builder method): at least 10 realistic, read-only questions with a single
  verifiable answer each, run against a seeded demo dataset, kept in `apps/api/evals/`. Example: "In which
  November day does the balance first drop below 1,300?". They measure whether an agent can actually use
  the tools. They need a model API key, so they run on demand and in an optional CI job, never as a
  required gate.
- **Demo dataset:** a seed command loads a fictional user with a few months of data. It serves the evals,
  contributors and screenshots.

### 6.11 Packaging

- Multi-stage Docker build into a minimal static image, built for amd64 and arm64.
- `docker compose up` starts Postgres and the API. Migrations run on start.
- A `dev` profile adds Mailpit, which catches outgoing email and shows it in a web inbox.
- The binary also carries admin commands: `horizon-money users reset-password <email>`.

## 7. Design decisions worth knowing

1. Timezone, main currency and savings-rate band are user settings. Divisor and low-balance threshold are workspace settings.
2. Tags are per user, shared across workspaces, with no 100-tag limit.
3. One-off movements before D0 are rejected, but series may start before D0 (only occurrences from D0
   count). D0 can move earlier, not later past existing data. A workspace's currency is frozen once it
   has data.
4. Monthly series anchored on day 29–31 fall on the month's last day in shorter months. An installment remainder goes to the last part.
5. Editing a series with `all` keeps the occurrences that were edited individually.
6. Transfers have no repetition and no tags in v1. The PTAX selling rate is used for conversion.
7. Amounts are decimal strings at the API edge, minor units inside.
8. MCP lives inside the API binary. It takes OAuth 2.1 (CIMD + DCR) and personal access tokens.
   The account pages are rendered by Go, not by the future Next.js app, so sign-in and consent live in one place.
9. Manual constructor injection, no DI library.
10. Yearly views show all 12 months, and future months include the forecast.
11. Sign-up emails are not verified. Google does not auto-merge into an existing password account.
    Password reset works by email when SMTP is set, and always works through a server command.

## 8. Left to the builder

Exact endpoint paths and tool names, pagination, id type, package names inside `internal/`, test
database strategy (testcontainers or compose), the rate-limiter algorithm, and the docs UI library.
