package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func TestParseMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input         string
		expectedFirst string
		expectedLast  string
		expectedDays  int
	}{
		{input: "2026-02", expectedFirst: "2026-02-01", expectedLast: "2026-02-28", expectedDays: 28},
		{input: "2028-02", expectedFirst: "2028-02-01", expectedLast: leapDay, expectedDays: 29},
		{input: "2026-11", expectedFirst: nov01, expectedLast: nov30, expectedDays: 30},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			m := month(t, tt.input)
			if m.String() != tt.input {
				t.Errorf("String() = %q, want %q", m.String(), tt.input)
			}
			if m.First() != date(t, tt.expectedFirst) || m.Last() != date(t, tt.expectedLast) || m.Days() != tt.expectedDays {
				t.Errorf("%s = %s..%s (%d days), want %s..%s (%d days)",
					tt.input, m.First(), m.Last(), m.Days(), tt.expectedFirst, tt.expectedLast, tt.expectedDays)
			}
		})
	}
}

func TestParseMonthRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"2026-13", "2026-1", "26-01", "", nov01} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.ParseMonth(input); !errors.Is(err, domain.ErrInvalidMonth) {
				t.Errorf("ParseMonth(%q) error = %v, want %v", input, err, domain.ErrInvalidMonth)
			}
		})
	}
}

func TestNewMonth(t *testing.T) {
	t.Parallel()

	got, err := domain.NewMonth(2026, time.November)
	if err != nil || got != month(t, "2026-11") || got.Year() != 2026 || got.Month() != time.November {
		t.Errorf("NewMonth(2026, November) = %s, %v", got, err)
	}
	for _, year := range []int{0, 10000} {
		if _, err := domain.NewMonth(year, time.January); !errors.Is(err, domain.ErrInvalidMonth) {
			t.Errorf("NewMonth(%d, January) error = %v, want %v", year, err, domain.ErrInvalidMonth)
		}
	}
	if _, err := domain.NewMonth(2026, 13); !errors.Is(err, domain.ErrInvalidMonth) {
		t.Errorf("NewMonth(2026, 13) error = %v, want %v", err, domain.ErrInvalidMonth)
	}
}

func TestMonthOf(t *testing.T) {
	t.Parallel()

	if got := domain.MonthOf(date(t, "2026-11-27")); got != month(t, "2026-11") {
		t.Errorf("MonthOf(2026-11-27) = %s, want 2026-11", got)
	}
}

func TestMonth_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from     string
		months   int
		expected string
	}{
		{from: "2026-12", months: 1, expected: "2027-01"},
		{from: "2026-01", months: -1, expected: "2025-12"},
		{from: "2026-11", months: 14, expected: "2028-01"},
	}
	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			t.Parallel()

			if got := month(t, tt.from).Add(tt.months); got != month(t, tt.expected) {
				t.Errorf("%s.Add(%d) = %s, want %s", tt.from, tt.months, got, tt.expected)
			}
		})
	}
}

func TestMonth_Contains(t *testing.T) {
	t.Parallel()

	m := month(t, "2026-11")
	if !m.Contains(date(t, nov30)) || m.Contains(date(t, "2026-12-01")) {
		t.Error("2026-11 must contain 2026-11-30 and not 2026-12-01")
	}
}
