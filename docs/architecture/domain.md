# Domain (`apps/api/internal/domain`)

The pure business rules. The package does no I/O and never reads the wall clock (`depguard` and
`forbidigo` enforce both). Tests are black-box (`package domain_test`) and need no database.

## Files

| File | Holds |
|---|---|
| `errors.go` | `Error{Code}` and every sentinel; the code is what HTTP and MCP return |
| `currency.go` | `Currency` and `ParseCurrency`; the minor-unit scale comes from ISO 4217 data in `golang.org/x/text` |
| `amount.go` | `Amount` (`int64` minor units), `MaxAmount`, `ParseAmount` and `Format` for decimal strings |
| `date.go`, `month.go` | `Date` (a calendar date, midnight UTC inside), `Month`, and `Today(now, timezone)` |
| `rounding.go` | every rounding rule: floor plus remainder, half-up, and a 128-bit multiply-divide |
| `movement.go` | the five `MovementType` values |
| `series.go` | `Series`, `Occurrence`, `Exception`: recurrence, installments and following splits |
| `forecast.go` | `DailyValue`, the daily forecast |
| `ledger.go` | `Ledger`, `Day` and `Status`: the day-by-day balance and the horizon |
| `summary.go` | `Calculations`, `MonthSummary` and `YearSummary`: the four calculations and their labels |

The `Clock` port lives in `internal/port/clock.go`. A use case asks it for the instant, and `domain.Today`
turns that instant into the user's calendar date.

## How a read works

1. The use case loads one workspace and builds a `Ledger`:
   - settings: D0 (`Start`), the initial balance, `DailyValue(categories, divisor)` and the low-balance
     threshold;
   - `Entries`: one-off movements plus each series expanded with `Series.Expand(from, to, exceptions)`;
   - `Transfers`: that workspace's legs, incoming or outgoing.
2. It computes `today := domain.Today(clock.Now(), user.Timezone)`.
3. It asks the ledger: `Days`, `Horizon`, `MonthSummary` or `YearSummary`.

`Days` walks from D0 and never returns a day before it. The forecast counts only on days after today, from
D0 on. A monthly and a yearly summary derive every field from the same sums (totals, transfers, forecast
days, elapsed days). That is what keeps the §4.10 balance identity true, and a test checks it month by month.

## Series model

- A `Series` stores its template: type, amount, start, an optional end and an **anchor day**. Monthly
  series and installments fall on the anchor day, or on the last day of a shorter month.
- Installments also store `Parts`, `FirstPart`, and a `LastAmount` that takes the remainder. A part is
  numbered from `FirstPart` plus the months since `Start`.
- An `Exception` is keyed by the date the series produced. It either deletes that occurrence or overrides
  it with a full snapshot (date and amount). An override moved into a range appears there, and one moved
  out disappears.
- `SplitAt(at)` serves a `following` edit. The head ends the day before, and the tail starts at `at`
  keeping the anchor day and continuing the part numbers. `WithAmount` then sets the new value per part.

How series rows, series groups and exceptions are stored belongs to the persistence adapter (#8).

## Rounding and limits

- Installments: floor plus remainder on the last part.
- The daily value, the average daily and the savings rate (in basis points) round half-up.
- The savings band is compared on the rounded rate.
- Limits are constants: `MaxAmount`, `Min/MaxInstallments`, `Min/MaxForecastDivisor`,
  `Default/MaxHorizonMonths`.
- ADR-0012 records the calculation details the spec left open.
