package domain

import "time"

// Date is a calendar date with no time of day, such as a movement's date or a workspace's D0. It wraps
// midnight UTC, so == and map keys compare dates. The zero value means "no date".
type Date struct {
	t time.Time
}

const (
	minYear    = 1
	maxYear    = 9999
	dateLayout = "2006-01-02"
	secsPerDay = 24 * 60 * 60
)

// NewDate builds a date, refusing days the month does not have (February 30) and years outside 1–9999.
func NewDate(year int, month time.Month, day int) (Date, error) {
	if year < minYear || year > maxYear {
		return Date{}, ErrInvalidDate
	}
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || t.Month() != month || t.Day() != day {
		return Date{}, ErrInvalidDate
	}
	return Date{t: t}, nil
}

// ParseDate reads exactly "YYYY-MM-DD".
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, ErrInvalidDate
	}
	return NewDate(t.Year(), t.Month(), t.Day())
}

// Today is the calendar date of the instant now in the user's timezone. The day changes at the user's
// midnight, which is when that day's forecast disappears.
func Today(now time.Time, loc *time.Location) Date {
	local := now.In(loc)
	return Date{t: time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)}
}

// Year is the date's year.
func (d Date) Year() int { return d.t.Year() }

// Month is the date's month.
func (d Date) Month() time.Month { return d.t.Month() }

// Day is the date's day of the month.
func (d Date) Day() int { return d.t.Day() }

// String formats the date as "YYYY-MM-DD".
func (d Date) String() string { return d.t.Format(dateLayout) }

// IsZero reports whether d is the zero Date, which means "no date".
func (d Date) IsZero() bool { return d.t.IsZero() }

// AddDays moves the date n days, backwards when n is negative.
func (d Date) AddDays(n int) Date { return Date{t: d.t.AddDate(0, 0, n)} }

// DaysSince counts the days from o to d: positive when d is later.
func (d Date) DaysSince(o Date) int { return int((d.t.Unix() - o.t.Unix()) / secsPerDay) }

// Compare returns -1, 0 or +1 as d is before, equal to or after o.
func (d Date) Compare(o Date) int { return d.t.Compare(o.t) }

// Before reports whether d is earlier than o.
func (d Date) Before(o Date) bool { return d.t.Before(o.t) }

// After reports whether d is later than o.
func (d Date) After(o Date) bool { return d.t.After(o.t) }

// later is the later of two dates.
func later(a, b Date) Date {
	if a.After(b) {
		return a
	}
	return b
}

// IsWeekend reports whether the date is a Saturday or a Sunday.
func (d Date) IsWeekend() bool {
	weekday := d.t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}
