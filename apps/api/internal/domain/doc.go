// Package domain holds Horizon Money's pure business rules: money, calendar dates, series and
// installments, the daily forecast, the day-by-day balance and the monthly and yearly calculations.
//
// The package does no I/O and never reads the wall clock. "Today" is always passed in, computed by
// Today from the instant the Clock port returns. Amounts are minor units of one currency; every
// rounding rule lives in rounding.go.
package domain
