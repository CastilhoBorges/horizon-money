# Domain core

**Date:** 2026-11-27

## What happened

- Built `internal/domain`, closing #5, #6 and #7:
  - money as minor units, with decimal strings at the edge;
  - ISO 4217 scales;
  - calendar dates and months, and the user's "today";
  - recurring series, installments, exceptions and following splits;
  - the daily forecast, the day-by-day balance, the horizon, and the monthly and yearly calculations.
- Added the `Clock` port in `internal/port`.
- The §4.14 acceptance fixture passes. It checks every end balance from Nov 1 to Dec 2, the whole November
  summary and the year totals.
- The §4.10 balance identity is checked month by month, from before D0 into future months. The scenario
  has a recurring expense and transfers in both directions.
- ADR-0012 records the calculation details the spec left open.

## Learned

- `golang.org/x/text/currency` returns the zero unit with a nil error for `"XXX"`, so the parser checks for
  it explicitly.
- A monthly series needs an explicit anchor day. Deriving the day from the start date breaks a series on
  the 31st as soon as a following split lands on February 28.
- The savings rate overflows `int64` long before amounts look unrealistic (`saving × 10 000`). A 128-bit
  multiply-divide (`math/bits`) keeps it exact.
- `goconst` counts string literals across test files. Dates shared by several tests live as constants in
  `domain_test.go`.
