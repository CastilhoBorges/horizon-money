# Domain core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the pure rules in `apps/api/internal/domain` for issues #5, #6 and #7: money and dates,
series and installments, the daily forecast, the day-by-day balance, and the monthly and yearly
calculations. The §4.14 acceptance fixture proves them.

**Architecture:** One Go package, `internal/domain`, with no I/O and no wall clock. "Today" comes in as a
`Date`, which `domain.Today` computes from the `port.Clock` instant and the user's timezone. A `Ledger`
holds one workspace's settings plus a flat list of dated entries and transfer legs. It answers every read
(`Days`, `Horizon`, `MonthSummary`, `YearSummary`). Series are expanded on read (§6.3). The use cases (#9)
will build the ledger from stored one-offs, expanded occurrences and transfers.

**Tech Stack:** Go 1.27, standard `testing` (no testify), `golang.org/x/text/currency` for ISO 4217 scales.

**Spec:** `docs/specs/2026-11-27-product-and-backend-design.md` §4.3, §4.5, §4.7–§4.11, §4.14, §6.2, §6.3, §6.10.

**Skills consulted:** golang-how-to, golang-testing, golang-naming, golang-error-handling,
golang-design-patterns, tdd, superpowers:writing-plans.

**Delivery:** branch `feat/domain-core` in `.claude/worktrees/feat-domain-core`. Each task is one commit.
One PR titled `feat(domain): money, dates, recurrence, forecast and summaries`, with `Closes #5`, `Closes #6`
and `Closes #7`.

**Out of scope (owner):**
- per-tag totals and checked days (#9);
- persisting series groups and moving a split's exceptions to the new half (#8, #9);
- description and tag validation (#9);
- the system `Clock` adapter and its wiring (#9);
- cross-rate conversion (#15).

## Global Constraints

- `internal/domain` imports only the standard library and `golang.org/x/text/currency`. It never calls
  `time.Now` (forbidigo) and never imports `net/http`, `database/sql`, pgx, `app` or adapters (depguard).
- Money is `domain.Amount`: `int64` minor units. No `float32` or `float64` in the package, tests included.
- Every rounding rule lives in `internal/domain/rounding.go`: half-up for divisions, and floor plus
  remainder for installments.
- Errors are `*domain.Error` sentinels with stable codes (`area.snake_case`), matched with `errors.Is`.
  Each task adds its own sentinels to `errors.go`.
- Validation limits are exported constants in `internal/domain`, with these exact values:
  `MaxAmount = 9_999_999_999_999`, `MinInstallments = 2`, `MaxInstallments = 360`,
  `MinForecastDivisor = 28`, `MaxForecastDivisor = 31`, `DefaultHorizonMonths = 3`, `MaxHorizonMonths = 12`.
- Tests:
  - black-box (`package domain_test`), table-driven, with named lowercase subtests;
  - `t.Parallel()` in every test and subtest (paralleltest);
  - one `_test.go` per source file, and shared helpers in `domain_test.go` call `t.Helper()`;
  - amounts in tests are minor-unit literals (`22660` is 226.60). Expected values come from the spec or are
    worked out by hand, never recomputed the way the code computes them.
- Before every commit, run from `apps/api` in the worktree: `go test -race ./...` and
  `golangci-lint run ./...`, both clean.

## Decisions the spec leaves open (confirm before Task 1; recorded as ADR-0012 in Task 7)

1. **`low` means `0 ≤ end < low_balance_threshold`.** With the default threshold of 0, a day is never `low`.
2. **The savings rate is in basis points, rounded half-up** (27.14 % = `2714`). The band is compared on the
   rounded value, so the label always matches the number shown.
3. **The average daily is rounded half-up to the minor unit** (18420 ÷ 27 = 682.2 → `682`).
4. **`Days` never returns days before D0.** The range is clamped to D0. Any balance before D0 is
   `initial_balance` (§4.9).
5. **An exception override is a full snapshot** (date and amount) of one occurrence, keyed by the date the
   series produced. A later `all` edit never touches it (§4.5).
6. **Monthly series and installments keep an explicit anchor day.** A `following` split of a series anchored
   on the 31st still falls on the 31st after a short month.
7. **Splitting at a series' first occurrence is refused** (`series.split_at_first_occurrence`). In that case
   the use case edits the template.
8. **The horizon spans 1–12 months**, 3 by default.
9. **An installment total below one minor unit per part is refused** (`series.installment_part_too_small`),
   so no part is ever zero.

## Test seams (confirm before Task 1)

Tests touch only the exported API of `internal/domain`. The unexported rounding helpers are covered only
through these seams.

| Seam | Catches | Leaves to later issues |
|---|---|---|
| `ParseCurrency`, `ParseAmount`, `Amount.Format` | scale errors, malformed decimal strings, overflow | the HTTP/MCP edges themselves (#9, #14) |
| `NewDate`, `ParseDate`, `Today`, `Date` and `Month` methods | calendar arithmetic, the midnight in the user's timezone | — |
| `NewRecurring`, `NewInstallments`, `Series.Expand`, `SplitAt`, `WithAmount` | month-end anchoring, remainder, partial view from D0, exceptions, split numbering | how series rows are stored and grouped (#8) |
| `DailyValue`, `Ledger.Days`, `Horizon`, `MonthSummary`, `YearSummary` | §4.14, the §4.10 invariant, forecast cut-off, statuses, labels, rounding | how use cases assemble the ledger from the database (#9) |

## Review Focus

1. **A `following` split of a series anchored on the 29th–31st, after a short month,** must keep the anchor
   (Mar 31, not Mar 28). Pinned in Task 3.
2. **The midnight cut-off happens in the user's timezone.** 23:59 in São Paulo is still "today" even when UTC
   has already moved to the next day. Pinned in Tasks 2 and 5.
3. **Large sums must not overflow the savings rate.** `saving × 10 000` passes `int64` long before the
   amounts look absurd, so the code uses a 128-bit intermediate (`math/bits`). Pinned in Task 6.
4. **Agents send loose decimal strings.** `"26"` and `"26.5"` are accepted. `"26,00"`, `"1,000.00"`,
   `" 26"`, `"+26"` and `"1e3"` are refused, never misread. Pinned in Task 1.
5. **An override can move an occurrence across the requested range.** Moved in, it appears; moved out, it
   disappears. Pinned in Task 3.

---

## File structure (`apps/api/internal/domain/` unless noted)

| File | Holds |
|---|---|
| `doc.go` | package doc: pure rules, no I/O, no wall clock |
| `errors.go` | `Error` and every sentinel |
| `currency.go` | `Currency`, `ParseCurrency` |
| `amount.go` | `Amount`, `MaxAmount`, `ParseAmount`, `Format` |
| `date.go` | `Date`, `NewDate`, `ParseDate`, `Today` |
| `month.go` | `Month`, `NewMonth`, `MonthOf`, `ParseMonth` |
| `rounding.go` | `splitParts`, `divHalfUp`, `mulDivHalfUp`: unexported, and the only rounding code |
| `movement.go` | `MovementType` and its five values |
| `series.go` | `Frequency`, `Series`, `Occurrence`, `Exception`, constructors, `Expand`, `SplitAt`, `WithAmount` |
| `forecast.go` | `DailyValue` |
| `ledger.go` | `Ledger`, `Entry`, `TransferLeg`, `Totals`, `Day`, `Status`, `Days`, `Horizon` |
| `summary.go` | `BasisPoints`, `SavingsTarget`, labels, `Calculations`, `MonthSummary`, `YearSummary` |
| `*_test.go` | one per source file; `domain_test.go` holds the `date(t, "2026-11-27")` and `month(t, "2026-11")` helpers |
| `apps/api/internal/port/clock.go` | `Clock` port and the `port` package doc |

---

### Task 1: errors, currencies and amounts (#5)

**Files:** create `doc.go`, `errors.go`, `currency.go`, `amount.go`, `domain_test.go`, `currency_test.go`,
`amount_test.go`. Modify `apps/api/go.mod` and `go.sum` with `go get golang.org/x/text@v0.44.0`.

**Interfaces — produces:**
- `type Error struct{ Code string }`, where `(*Error).Error()` returns `Code`.
- `ErrUnknownCurrency` (`currency.unknown`), `ErrInvalidAmount` (`amount.invalid`), `ErrTooManyDecimals`
  (`amount.too_many_decimals`), `ErrAmountTooLarge` (`amount.too_large`).
- `type Currency struct` with unexported code and scale; `ParseCurrency(code string) (Currency, error)`;
  `(Currency) Code() string`; `(Currency) Scale() int`.
- `type Amount int64`; `const MaxAmount Amount = 9_999_999_999_999`;
  `ParseAmount(s string, c Currency) (Amount, error)`; `(Amount) Format(c Currency) string`.

- [ ] **Step 1: write `TestParseCurrency`.** Expect these code and scale pairs: `"BRL"` → BRL, 2;
  `"brl"` → BRL, 2; `"USD"` → USD, 2; `"CAD"` → CAD, 2; `"JPY"` → JPY, 0; `"KWD"` → KWD, 3. Expect
  `ErrUnknownCurrency` for `""`, `"XXX"`, `"ABC"`, `"BRLX"` and `"R$"`.
- [ ] **Step 2:** `go test ./internal/domain/ -run TestParseCurrency` fails (undefined).
- [ ] **Step 3: implement `ParseCurrency`.** Call `currency.ParseISO`, which is case-insensitive. Then reject
  the zero `currency.Unit{}`, because x/text returns it for `"XXX"` with a nil error. The scale comes from
  `currency.Standard.Rounding(u)`.
- [ ] **Step 4:** the test passes.
- [ ] **Step 5: write `TestParseAmount`** (BRL unless noted):
  - accepted: `"26"` → 2600; `"26.5"` → 2650; `"26.00"` → 2600; `"0.01"` → 1; `"-41.67"` → -4167;
    `"-0.00"` → 0; `"99999999999.99"` → 9999999999999; JPY `"1500"` → 1500; KWD `"1.234"` → 1234;
  - `ErrInvalidAmount`: `""`, `"abc"`, `"26,00"`, `"1,000.00"`, `".5"`, `"5."`, `"+5"`, `" 5"`, `"5 "`,
    `"1e3"`, `"--1"`;
  - `ErrTooManyDecimals`: `"26.001"`, and JPY `"1500.0"`;
  - `ErrAmountTooLarge`: `"100000000000.00"`, `"-100000000000.00"`, `"99999999999999999999999"`.
- [ ] **Step 6: write `TestAmount_Format`.** BRL: 142245 → `"1422.45"`, -4167 → `"-41.67"`, 5 → `"0.05"`,
  -5 → `"-0.05"`, 0 → `"0.00"`. JPY: 1500 → `"1500"`. KWD: 1234 → `"1.234"`. Add a round-trip subtest:
  `ParseAmount(a.Format(c), c) == a` for each of those values.
- [ ] **Step 7:** both tests fail.
- [ ] **Step 8: implement `ParseAmount` and `Format`.**
  - Match against the pattern `^-?[0-9]+(\.[0-9]+)?$`, compiled once at package level.
  - More fraction digits than the scale returns `ErrTooManyDecimals`.
  - Bound the digit count before multiplying, so no input can overflow. Any absolute value above
    `MaxAmount` returns `ErrAmountTooLarge`.
- [ ] **Step 9:** tests and lint are clean. Commit `feat(domain): currencies and decimal-string amounts`.

### Task 2: dates, months, today and the Clock port (#5)

**Files:** create `date.go`, `month.go`, `date_test.go`, `month_test.go`, `apps/api/internal/port/clock.go`.
Modify `errors.go` and `domain_test.go`.

**Interfaces — produces:**
- `ErrInvalidDate` (`date.invalid`), `ErrInvalidMonth` (`month.invalid`).
- `type Date struct`: a `time.Time` at midnight UTC, so `==` and map keys work. The zero value means "no date".
  - Constructors: `NewDate(year int, month time.Month, day int) (Date, error)` (years 1–9999, no
    normalisation), `ParseDate(s string) (Date, error)` (exactly `YYYY-MM-DD`), and
    `Today(now time.Time, loc *time.Location) Date`.
  - Methods: `Year() int`, `Month() time.Month`, `Day() int`, `String() string`, `IsZero() bool`,
    `AddDays(n int) Date`, `DaysSince(o Date) int`, `Compare(o Date) int`, `Before(o Date) bool`,
    `After(o Date) bool`, `IsWeekend() bool`.
- `type Month struct`.
  - Constructors: `NewMonth(year int, month time.Month) (Month, error)`, `MonthOf(d Date) Month`, and
    `ParseMonth(s string) (Month, error)` (exactly `YYYY-MM`).
  - Methods: `Year() int`, `Month() time.Month`, `First() Date`, `Last() Date`, `Days() int`,
    `Add(n int) Month`, `Contains(d Date) bool`, `String() string`.
- In `port`: `type Clock interface { Now() time.Time }`, documented as the only source of time for `app`.

- [ ] **Step 1: write `TestNewDate`.** `(2028, 2, 29)` is accepted. `(2026, 2, 29)`, `(2026, 4, 31)`,
  `(2026, 13, 1)`, `(0, 1, 1)` and `(10000, 1, 1)` return `ErrInvalidDate`.
- [ ] **Step 2: write `TestParseDate`.** `"2026-11-27"` round-trips through `String`. `""`, `"2026-11-7"`,
  `"27/11/2026"`, `"2026-13-01"`, `"2026-02-30"` and `"2026-11-27T00:00:00Z"` return `ErrInvalidDate`.
- [ ] **Step 3: write `TestToday`.** The test file blank-imports `time/tzdata`, so it is hermetic.
  - America/Sao_Paulo: `2026-11-28T02:59:59Z` → 2026-11-27, and `2026-11-28T03:00:00Z` → 2026-11-28;
  - UTC: `2026-11-28T02:59:59Z` → 2026-11-28.
- [ ] **Step 4: write `TestDate_AddDays`, `TestDate_DaysSince` and `TestDate_IsWeekend`.**
  - `AddDays`: 2026-11-30 +1 → 2026-12-01; 2026-12-31 +1 → 2027-01-01; 2028-02-28 +1 → 2028-02-29;
    2026-03-01 −1 → 2026-02-28;
  - `DaysSince`: 2026-11-27 since 2026-11-01 = 26, and the reverse = −26;
  - `IsWeekend`: 2026-11-27 (Fri) false, 11-28 true, 11-29 true, 11-30 (Mon) false.
- [ ] **Step 5: write `TestParseMonth` and the `TestMonth_*` tests.**
  - `"2026-02"`: First 2026-02-01, Last 2026-02-28, Days 28. `"2028-02"`: Days 29. `"2026-11"`: Days 30;
  - `Add`: 2026-12 +1 → 2027-01; 2026-01 −1 → 2025-12; 2026-11 +14 → 2028-01;
  - `Contains`: 2026-11 contains 2026-11-30 but not 2026-12-01;
  - `ErrInvalidMonth`: `"2026-13"`, `"2026-1"`, `"26-01"`, `""`, `"2026-11-01"`.
- [ ] **Step 6:** the tests fail. Implement on `time.Date(..., time.UTC)`. `NewDate` refuses any input that
  `time.Date` would normalise. The tests pass.
- [ ] **Step 7:** tests and lint are clean. Commit `feat(domain): calendar dates, months and the Clock port`.

### Task 3: recurring series, exceptions and following splits (#6)

**Files:** create `movement.go`, `series.go`, `series_test.go`. Modify `errors.go`.

**Interfaces — produces:**
- `type MovementType string`: `TypeIncome` `"income"`, `TypeExpense` `"expense"`, `TypeDaily` `"daily"`,
  `TypeSaving` `"saving"`, `TypeCard` `"card"`.
- `type Frequency string`: `FrequencyDaily` `"daily"`, `FrequencyWeekly` `"weekly"`, `FrequencyMonthly`
  `"monthly"`, `FrequencyInstallments` `"installments"`.
- `ErrAmountNotPositive` (`movement.amount_not_positive`), `ErrUnknownMovementType` (`movement.type_unknown`),
  `ErrUnknownFrequency` (`series.frequency_unknown`), `ErrEndBeforeStart` (`series.end_before_start`),
  `ErrNotAnOccurrence` (`series.not_an_occurrence`), `ErrSplitAtFirstOccurrence`
  (`series.split_at_first_occurrence`).
- `type Series struct`, with exported fields so #8 can rebuild it from a row:
  - `Type MovementType`, `Frequency Frequency`;
  - `Start Date`: the first occurrence;
  - `End Date`: the last day an occurrence may fall on, inclusive; zero means no end;
  - `AnchorDay int`: for monthly series and installments, 1–31, clamped to shorter months;
  - `Amount Amount`: every occurrence, or for installments every part but the last;
  - `LastAmount Amount`, `Parts int`, `FirstPart int`: installments only (Task 4).
- `NewRecurring(t MovementType, f Frequency, amount Amount, start, end Date) (Series, error)`. It accepts only
  daily, weekly or monthly, and sets `AnchorDay = start.Day()`.
- `type Occurrence struct { Date, Original Date; Amount Amount; Part, Parts int }`. `Date` is where the
  occurrence counts. `Original` is the date the series produced, which keys exceptions. `Part` and `Parts`
  are 0 outside installments.
- `type Exception struct { Deleted bool; Date Date; Amount Amount }`. When `Deleted` is false, it is a full
  override.
- `(s Series) Expand(from, to Date, exceptions map[Date]Exception) []Occurrence`, sorted by `Date`, then
  by `Original`.
- `(s Series) SplitAt(at Date) (head, tail Series, err error)`. `head.End = at − 1`; `tail.Start = at`. Every
  other field is kept, `AnchorDay` included.
- `(s Series) WithAmount(a Amount) (Series, error)`: the new per-occurrence amount for a following or all edit.

- [ ] **Step 1: write `TestNewRecurring`.** Each case returns its error:
  - end 2026-11-01 before start 2026-11-02 → `ErrEndBeforeStart`;
  - amount 0 or −1 → `ErrAmountNotPositive`;
  - frequency `"yearly"` or `FrequencyInstallments` → `ErrUnknownFrequency`;
  - type `"transfer"` → `ErrUnknownMovementType`.
- [ ] **Step 2: write `TestSeries_Expand`.** Expected dates per case:
  - monthly from 2026-01-31, expanded over Jan–Jun 2026 → Jan 31, Feb 28, Mar 31, Apr 30, May 31, Jun 30;
  - monthly from 2028-01-30 → Feb 29 2028;
  - monthly from 2026-01-29 → Feb 28, Mar 29;
  - weekly from 2026-11-02 over Nov 2026 → 2, 9, 16, 23, 30;
  - weekly from 2020-01-06 over Nov 2026 → 2, 9, 16, 23, 30;
  - daily from 2026-11-28 with end 2026-11-30, over Nov–Dec → 28, 29, 30;
  - monthly rent from 2020-01-05 over Nov 2026 → only 2026-11-05;
  - a range entirely before the start, or entirely after the end → empty.
- [ ] **Step 3: write `TestSeries_ExpandWithExceptions`.** Monthly 100000 from 2026-01-10, expanded over
  Feb 1–Apr 30:
  - with no exceptions → Feb 10, Mar 10, Apr 10;
  - Mar 10 deleted → Feb 10, Apr 10;
  - Feb 10 overridden to {Feb 12, 120000} → that occurrence has Date Feb 12, Original Feb 10, Amount 120000;
  - Jan 10 (outside the range) overridden to {Feb 1, 100000} → it appears first;
  - Apr 10 overridden to {May 2, 100000} → it disappears;
  - a key on Feb 11, which the series never produces, is ignored.
- [ ] **Step 4: write `TestSeries_SplitAt` and `TestSeries_WithAmount`.**
  - Monthly from 2026-01-31, `SplitAt(2026-02-28)`: head over 2026 → [Jan 31]; tail over Feb–Apr →
    Feb 28, Mar 31, Apr 30 (Review Focus 1);
  - `SplitAt(2026-02-27)` → `ErrNotAnOccurrence`; `SplitAt(2026-01-31)` → `ErrSplitAtFirstOccurrence`;
  - `WithAmount(0)` → `ErrAmountNotPositive`. `WithAmount(5000)` expands to amounts of 5000.
- [ ] **Step 5:** the tests fail. Implement them. One unexported `isRuleDate(d)` serves both the override
  pass of `Expand` and `SplitAt`. `Expand` jumps straight to `from` instead of walking from `Start`. A
  monthly rule date is `min(AnchorDay, month.Days())`. Use an exhaustive `switch` on `Frequency`. The tests
  pass.
- [ ] **Step 6:** tests and lint are clean. Commit `feat(domain): recurring series, exceptions and following splits`.

### Task 4: installments (#6)

**Files:** create `rounding.go`. Modify `series.go`, `series_test.go` and `errors.go`.

**Interfaces — produces:**
- `ErrInstallmentsOutOfRange` (`series.installments_out_of_range`), `ErrInstallmentPartTooSmall`
  (`series.installment_part_too_small`). `const MinInstallments = 2`, `MaxInstallments = 360`.
- `NewInstallments(t MovementType, total Amount, parts int, first Date) (Series, error)`. It sets
  `Frequency = FrequencyInstallments`, `FirstPart = 1`, `AnchorDay = first.Day()`,
  `Amount = floor(total ÷ parts)` and `LastAmount = total − Amount × (parts − 1)`.
- `splitParts(total Amount, parts int) (part, last Amount)` in `rounding.go`.
- Changes to Task 3's functions:
  - `Expand` numbers the parts as `FirstPart` plus the months since `Start`, stops after part `Parts`, and
    gives `LastAmount` to part `Parts`;
  - `SplitAt` sets `tail.FirstPart` to the part number of `at`;
  - `WithAmount` also sets `LastAmount`, so after an edit every part from there on has the edited value (§4.5).

- [ ] **Step 1: write `TestNewInstallments`.** Expect these part and last values:
  - (10000, 3) → 3333, 3334; (100000, 12) → 8333, 8337; (1000, 360) → 2, 282; (2, 2) → 1, 1;
    (700, 7) → 100, 100;
  - in every case, the expanded parts sum exactly to the total;
  - errors: parts 1 or 361 → `ErrInstallmentsOutOfRange`; total 5 in 12 parts → `ErrInstallmentPartTooSmall`;
    total 0 → `ErrAmountNotPositive`.
- [ ] **Step 2: write `TestSeries_ExpandInstallments`.**
  - 10000 in 3 from 2026-01-10, over all of 2026 → Jan 10 1/3 3333, Feb 10 2/3 3333, Mar 10 3/3 3334, and
    nothing in April;
  - the §4.5 example, 100000 in 12 from 2026-08-15, over 2026-11-01..2027-12-31 → nine occurrences: parts
    4/12 (2026-11-15) to 12/12 (2027-07-15), eight of 8333 and the last 8337;
  - 3 parts from 2026-01-31 → Jan 31, Feb 28, Mar 31.
- [ ] **Step 3: write `TestSeries_SplitAtInstallments`.** 10000 in 3 from 2026-01-10:
  - `SplitAt(2026-02-10)`: head → [1/3 3333]; tail → [2/3 3333 Feb 10, 3/3 3334 Mar 10];
  - `tail.WithAmount(4000)` → [2/3 4000, 3/3 4000];
  - `SplitAt(2026-04-10)` → `ErrNotAnOccurrence`.
- [ ] **Step 4:** the tests fail. Implement them. The tests pass.
- [ ] **Step 5:** tests and lint are clean. Commit `feat(domain): installments with the remainder on the last part`.

### Task 5: daily forecast, day-by-day balance and horizon (#7)

**Files:** create `forecast.go`, `ledger.go`, `forecast_test.go`, `ledger_test.go`. Modify `rounding.go`
and `errors.go`.

**Interfaces — produces:**
- `ErrForecastDivisorOutOfRange` (`workspace.forecast_divisor_out_of_range`), `ErrForecastAmountNotPositive`
  (`forecast.amount_not_positive`), `ErrHorizonOutOfRange` (`horizon.months_out_of_range`).
  `const MinForecastDivisor = 28`, `MaxForecastDivisor = 31`, `DefaultHorizonMonths = 3`, `MaxHorizonMonths = 12`.
- `divHalfUp(n, d int64) int64` in `rounding.go`, for `n ≥ 0` and `d > 0`.
- `DailyValue(monthlyAmounts []Amount, divisor int) (Amount, error)`: the half-up value of the sum divided
  by the divisor.
- Value types:
  - `type Entry struct { Date Date; Type MovementType; Amount Amount }`;
  - `type TransferLeg struct { Date Date; Amount Amount; Incoming bool }`;
  - `type Totals struct { Income, Expense, Daily, Saving, Card Amount }`;
  - `type Status string`: `StatusOK` `"ok"`, `StatusLow` `"low"`, `StatusNegative` `"negative"`;
  - `type Day struct { Date Date; Totals Totals; TransfersIn, TransfersOut, Forecast, End Amount; IsWeekend bool; Status Status }`;
  - `type Ledger struct { Start Date; InitialBalance, DailyForecast, LowThreshold Amount; Entries []Entry; Transfers []TransferLeg }`.
- `(l Ledger) Days(from, to, today Date) []Day`:
  - the range is clamped to `l.Start`, and an empty range returns nil;
  - entries and legs dated before `Start` are ignored;
  - `Forecast` is `DailyForecast` when `d > today` and `d ≥ Start`;
  - `End` follows §4.9;
  - `Status` follows Decision 1.
- `(l Ledger) Horizon(first Month, months int, today Date) ([]Day, error)`.

- [ ] **Step 1: write `TestDailyValue`.**
  - ([100000], 30) → 3333 (§4.7); ([35000, 30000, 60000], 30) → 4167; ([105], 30) → 4 (exactly half
    rounds up); ([104], 30) → 3; ([100000], 28) → 3571; ([100000], 31) → 3226; (nil, 30) → 0;
  - divisor 27 or 32 → `ErrForecastDivisorOutOfRange`; an amount of 0 or −1 → `ErrForecastAmountNotPositive`.
- [ ] **Step 2: write `TestLedger_Days_AcceptanceFixture`.** Add an `acceptanceLedger(t)` builder in
  `ledger_test.go`: §4.14 with Start 2026-11-01, InitialBalance 25000, DailyForecast from
  `DailyValue([35000, 30000, 60000], 30)`, and the §4.14 entries as `Entry` values. With today 2026-11-27,
  `Days(2026-11-01, 2026-12-02)` gives:
  - `End` on each of the 32 days: 1 → 22660, 2 → 21770, 3 → 66015, 4 → 62495, 5 → 57285, 6 → 148705,
    7…27 → 142245, 28 → 138078, 29 → 133911, 30 → 129744; Dec 1 → 125577, Dec 2 → 121410;
  - `Forecast` 0 on Nov 27 and 4167 on Nov 28;
  - `Totals` on Nov 3 = {Income 60000, Expense 15755}, and on Nov 6 = {Income 640000, Expense 145000,
    Saving 190000, Card 213580}.
- [ ] **Step 3: write `TestLedger_Days_ForecastCutOff`** (Review Focus 2). Use Start 2026-11-01,
  InitialBalance 10000 and DailyForecast 1000, with today taken from `Today(…, America/Sao_Paulo)`:
  - at `2026-11-28T02:59:59Z`, Nov 27 has forecast 0 and Nov 28 has 1000;
  - at `2026-11-28T03:00:00Z`, Nov 28 has 0 and Nov 29 has 1000.
- [ ] **Step 4: write `TestLedger_Days_StartsAtD0`.** Use Start 2026-12-05, InitialBalance 10000,
  DailyForecast 1000, today 2026-11-27, and an entry dated 2026-12-04. `Days(2026-12-01, 2026-12-06)`
  returns exactly Dec 5 (End 9000) and Dec 6 (End 8000).
- [ ] **Step 5: write `TestLedger_Days_Status`.** Each case is one day, with Start = today and no forecast.
  - With LowThreshold 10000, these initial balances give these statuses: −1 → negative, 0 → low, 9999 → low,
    10000 → ok;
  - with LowThreshold 0, a balance of 0 → ok.
- [ ] **Step 6: write `TestLedger_Days_TransfersAndWeekend`.** Use Start 2026-11-27, InitialBalance 10000,
  legs {11-28, 5000, in} and {11-28, 2000, out}, and today 11-27.
  - Nov 28: TransfersIn 5000, TransfersOut 2000, End 13000, IsWeekend true;
  - Nov 27: IsWeekend false.
- [ ] **Step 7: write `TestLedger_Horizon`.** `Horizon(2026-11, 3, today)` returns 92 days, from 2026-11-01
  to 2027-01-31. `months` 0 or 13 returns `ErrHorizonOutOfRange`.
- [ ] **Step 8:** the tests fail. Implement them, building one per-date map of totals and walking the days
  from `Start`. The tests pass.
- [ ] **Step 9:** tests and lint are clean. Commit `feat(domain): daily forecast, day-by-day balance and horizon`.

### Task 6: monthly and yearly calculations (#7)

**Files:** create `summary.go` and `summary_test.go`. Modify `rounding.go`.

**Interfaces — produces:**
- `mulDivHalfUp(a, b, d int64) int64` in `rounding.go`: half-up `a × b ÷ d` for `a, b ≥ 0` and `d > 0`,
  through `bits.Mul64` and `bits.Div64`.
- `type BasisPoints int64`; `type SavingsTarget struct { Min, Max BasisPoints }` (inclusive bounds).
- Labels:
  - `PerformanceLabel`: `PerformanceMoneyLeft` `"money_left"`, `PerformanceShort` `"short"`;
  - `SavingsBand`: `BandBelow` `"below"`, `BandWithin` `"within"`, `BandAbove` `"above"`, and `""` when
    there is no rate;
  - `CostLabel`: `CostWithinIncome` `"within_income"`, `CostAboveIncome` `"above_income"`.
- `type Calculations struct` with these fields: `Totals Totals`, `TransfersIn`, `TransfersOut`,
  `ForecastRemaining Amount`, `ForecastDays`, `ElapsedDays int`, `Performance Amount`,
  `PerformanceLabel PerformanceLabel`, `SavingsRate *BasisPoints`, `SavingsBand SavingsBand`,
  `CostOfLiving Amount`, `CostLabel CostLabel`, `AverageDaily *Amount`.
- `type MonthSummary struct { Month Month; DailyForecast Amount; Calculations }` and
  `(l Ledger) MonthSummary(m Month, today Date, target SavingsTarget) MonthSummary`.
- `type YearSummary struct { Year int; Months []MonthSummary; Calculations }` and
  `(l Ledger) YearSummary(year int, today Date, target SavingsTarget) (YearSummary, error)`. A year outside
  1–9999 returns `ErrInvalidMonth`.
- One unexported function derives every computed field from the raw sums (totals, transfers, forecast days,
  forecast remaining, elapsed days). It serves both a month and the year, whose raw sums are the 12 months'
  sums added up, and that is how §4.11's year totals come out. Elapsed days follow §4.10 against `l.Start`.

Every case uses target {2000, 3000}.

- [ ] **Step 1: write `TestLedger_MonthSummary_AcceptanceFixture`.** For Nov 2026 with today 11-27, expect:
  - Totals {Income 700000, Expense 160755, Daily 18420, Saving 190000, Card 213580};
  - ForecastDays 3, ForecastRemaining 12501, DailyForecast 4167;
  - Performance 104744 money_left;
  - SavingsRate 2714 within;
  - CostOfLiving 405256 within_income;
  - AverageDaily 682, ElapsedDays 27.
- [ ] **Step 2: write `TestLedger_MonthSummary_Periods`** on the fixture:
  - Dec 2026, a future month: ForecastDays 31, ForecastRemaining 129177, Performance −129177 short,
    SavingsRate nil, band `""`, CostOfLiving 129177 above_income, AverageDaily nil, ElapsedDays 0;
  - Oct 2026, before D0: every amount 0, money_left, within_income, both pointers nil;
  - Nov seen from today 2026-12-10, a past month: ForecastDays 0, ElapsedDays 30, AverageDaily 614,
    Performance 117245.
- [ ] **Step 3: write `TestLedger_MonthSummary_D0MidMonth`.** Use Start 2026-11-10, one daily entry of 10000
  on 11-10 and today 11-27. Expect ElapsedDays 18 and AverageDaily 556.
- [ ] **Step 4: write `TestLedger_MonthSummary_SavingsBand`.**
  - With income 10000, savings of 1999, 2000, 3000 and 3001 give below, within, within and above;
  - with income 100000, savings of 30004 and 30005 give 3000 within and 3001 above (Decision 2);
  - income 0 with a saving of 5000 gives nil and `""`.
- [ ] **Step 5: write `TestLedger_MonthSummary_LargeSums`** (Review Focus 3). In Nov 2026, add 1000 income
  entries of `MaxAmount` and 1000 saving entries of 4999999999999. Expect SavingsRate 5000.
- [ ] **Step 6: write `TestLedger_MonthSummary_BalanceIdentity`** (§4.10).
  - Scenario: the fixture, plus a monthly expense of 120000 from 2020-01-05 expanded over
    2026-11-01..2027-02-28, plus legs {2026-11-10, 50000, in} and {2026-12-03, 20000, out}. Today is 2026-11-27;
  - for each month from 2026-10 to 2027-02, assert `end(M.Last) − end(previous M.Last) == Performance +
    TransfersIn − TransfersOut`;
  - `end(d)` comes from `Days`, and is InitialBalance for any `d` before Start.
- [ ] **Step 7: write `TestLedger_YearSummary_AcceptanceFixture`.** For 2026 with today 11-27, expect:
  - 12 months, and `Months[10]` deep-equal to `MonthSummary(2026-11)`;
  - year Performance −24433 short;
  - SavingsRate 2714 within;
  - CostOfLiving 534433 within_income;
  - AverageDaily 682, ElapsedDays 27;
  - ForecastDays 34, ForecastRemaining 141678;
  - years 0 and 10000 return `ErrInvalidMonth`.
- [ ] **Step 8:** the tests fail. Implement them. The tests pass.
- [ ] **Step 9:** tests and lint are clean. Commit `feat(domain): monthly and yearly calculations`.

### Task 7: docs, gates and PR

**Files:** create `docs/architecture/domain.md`, `docs/decisions/0012-domain-calculation-details.md` and
`docs/devlog/2026/2026-11-27-domain-core.md`. Modify `docs/architecture/README.md`, `docs/GLOSSARY.md`,
`docs/ROADMAP.md`, and the generated indexes.

- [ ] **Step 1: write `docs/architecture/domain.md`** (under 120 lines): the file map, how a `Ledger` is
  built and read, the series model (anchor day, first part, overrides), and where rounding lives. Link it
  from `architecture/README.md`, and set the "Current state" row for the domain to Built.
- [ ] **Step 2: update `GLOSSARY.md`.** Add three rows: `parcela` → `part` (one installment, numbered "3/12");
  `dia âncora` → `anchor day` (the day of month a monthly series falls on, clamped in shorter months);
  `ledger` (a workspace's settings, movements and transfers as the domain reads them).
- [ ] **Step 3: write ADR-0012** from the nine decisions above, and the devlog entry (what shipped, what
  was learned).
- [ ] **Step 4: update `ROADMAP.md`.** Phase 0 → Done, and Phase 1 → In progress, since this PR starts it.
- [ ] **Step 5:** run `pnpm docs:index`. Then, from the worktree root, `pnpm test`, `pnpm lint` and
  `pnpm docs:check` all pass.
- [ ] **Step 6:** commit `docs: domain architecture, ADR-0012 and devlog`. Push, and open the PR.
  - The body lists what was reused: `golang.org/x/text/currency` for ISO 4217 scales. Nothing was extracted,
    because no earlier copy existed;
  - the body ends with `Closes #5`, `Closes #6` and `Closes #7`;
  - wait for green CI. Merging is the maintainer's call.
