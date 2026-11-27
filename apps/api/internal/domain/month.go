package domain

import "time"

// Month is a calendar month, the unit of the monthly view.
type Month struct {
	year  int
	month time.Month
}

const monthLayout = "2006-01"

// NewMonth builds a month, refusing months outside 1–12 and years outside 1–9999.
func NewMonth(year int, month time.Month) (Month, error) {
	if year < minYear || year > maxYear || month < time.January || month > time.December {
		return Month{}, ErrInvalidMonth
	}
	return Month{year: year, month: month}, nil
}

// MonthOf is the month a date falls in.
func MonthOf(d Date) Month { return Month{year: d.Year(), month: d.Month()} }

// ParseMonth reads exactly "YYYY-MM".
func ParseMonth(s string) (Month, error) {
	t, err := time.Parse(monthLayout, s)
	if err != nil {
		return Month{}, ErrInvalidMonth
	}
	return NewMonth(t.Year(), t.Month())
}

// Year is the month's year.
func (m Month) Year() int { return m.year }

// Month is the month of the year.
func (m Month) Month() time.Month { return m.month }

// First is the month's first day.
func (m Month) First() Date { return Date{t: time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.UTC)} }

// Last is the month's last day.
func (m Month) Last() Date { return Date{t: m.First().t.AddDate(0, 1, -1)} }

// Days is the number of days in the month.
func (m Month) Days() int { return m.Last().Day() }

// Add moves the month n months, backwards when n is negative.
func (m Month) Add(n int) Month { return MonthOf(Date{t: m.First().t.AddDate(0, n, 0)}) }

// Contains reports whether d falls in the month.
func (m Month) Contains(d Date) bool { return d.Year() == m.year && d.Month() == m.month }

// monthsBetween counts the months from a to b: positive when b is later.
func monthsBetween(a, b Month) int { return (b.year-a.year)*12 + int(b.month) - int(a.month) }

// String formats the month as "YYYY-MM".
func (m Month) String() string { return m.First().t.Format(monthLayout) }
