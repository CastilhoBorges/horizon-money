package domain_test

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata" // the timezone tests must not depend on the machine's zoneinfo

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func TestNewDate(t *testing.T) {
	t.Parallel()

	got, err := domain.NewDate(2028, time.February, 29)
	if err != nil {
		t.Fatalf("NewDate(2028-02-29) error = %v", err)
	}
	if got.Year() != 2028 || got.Month() != time.February || got.Day() != 29 {
		t.Errorf("NewDate(2028-02-29) = %d-%d-%d", got.Year(), got.Month(), got.Day())
	}

	invalid := []struct {
		name  string
		year  int
		month time.Month
		day   int
	}{
		{name: "february 29 outside a leap year", year: 2026, month: time.February, day: 29},
		{name: "april 31", year: 2026, month: time.April, day: 31},
		{name: "month 13", year: 2026, month: 13, day: 1},
		{name: "year 0", year: 0, month: time.January, day: 1},
		{name: "year 10000", year: 10000, month: time.January, day: 1},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewDate(tt.year, tt.month, tt.day); !errors.Is(err, domain.ErrInvalidDate) {
				t.Errorf("NewDate(%d, %d, %d) error = %v, want %v", tt.year, tt.month, tt.day, err, domain.ErrInvalidDate)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	got, err := domain.ParseDate(nov27)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", nov27, err)
	}
	if got.String() != nov27 {
		t.Errorf("ParseDate(%q).String() = %q", nov27, got.String())
	}

	for _, input := range []string{"", "2026-11-7", "27/11/2026", "2026-13-01", "2026-02-30", "2026-11-27T00:00:00Z"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.ParseDate(input); !errors.Is(err, domain.ErrInvalidDate) {
				t.Errorf("ParseDate(%q) error = %v, want %v", input, err, domain.ErrInvalidDate)
			}
		})
	}
}

func TestToday(t *testing.T) {
	t.Parallel()

	saoPaulo, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("LoadLocation error = %v", err)
	}
	tests := []struct {
		name     string
		now      string
		location *time.Location
		expected string
	}{
		{name: "one second before midnight in sao paulo", now: "2026-11-28T02:59:59Z", location: saoPaulo, expected: nov27},
		{name: "midnight in sao paulo", now: "2026-11-28T03:00:00Z", location: saoPaulo, expected: nov28},
		{name: "the same instant in utc", now: "2026-11-28T02:59:59Z", location: time.UTC, expected: nov28},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			now, err := time.Parse(time.RFC3339, tt.now)
			if err != nil {
				t.Fatalf("time.Parse(%q) error = %v", tt.now, err)
			}
			if got := domain.Today(now, tt.location); got != date(t, tt.expected) {
				t.Errorf("Today(%s, %s) = %s, want %s", tt.now, tt.location, got, tt.expected)
			}
		})
	}
}

func TestDate_AddDays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		from     string
		days     int
		expected string
	}{
		{name: "into the next month", from: nov30, days: 1, expected: "2026-12-01"},
		{name: "into the next year", from: dec31, days: 1, expected: "2027-01-01"},
		{name: "onto a leap day", from: "2028-02-28", days: 1, expected: leapDay},
		{name: "backwards", from: "2026-03-01", days: -1, expected: "2026-02-28"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := date(t, tt.from).AddDays(tt.days); got != date(t, tt.expected) {
				t.Errorf("%s.AddDays(%d) = %s, want %s", tt.from, tt.days, got, tt.expected)
			}
		})
	}
}

func TestDate_DaysSince(t *testing.T) {
	t.Parallel()

	if got := date(t, nov27).DaysSince(date(t, nov01)); got != 26 {
		t.Errorf("DaysSince = %d, want 26", got)
	}
	if got := date(t, nov01).DaysSince(date(t, nov27)); got != -26 {
		t.Errorf("DaysSince = %d, want -26", got)
	}
}

func TestDate_Compare(t *testing.T) {
	t.Parallel()

	earlier, same, later := date(t, nov27), date(t, nov27), date(t, nov28)
	if earlier.Compare(later) != -1 || later.Compare(earlier) != 1 || earlier.Compare(same) != 0 {
		t.Errorf("Compare is not ordering %s before %s", earlier, later)
	}
	if !earlier.Before(later) || earlier.After(later) || !later.After(earlier) || earlier.Before(same) {
		t.Errorf("Before/After disagree with the calendar for %s and %s", earlier, later)
	}
	if !(domain.Date{}).IsZero() || earlier.IsZero() {
		t.Error("IsZero is true only for the zero Date")
	}
}

func TestDate_IsWeekend(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		nov27:        false, // Friday
		nov28:        true,  // Saturday
		"2026-11-29": true,  // Sunday
		nov30:        false, // Monday
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if got := date(t, input).IsWeekend(); got != expected {
				t.Errorf("%s.IsWeekend() = %v, want %v", input, got, expected)
			}
		})
	}
}
