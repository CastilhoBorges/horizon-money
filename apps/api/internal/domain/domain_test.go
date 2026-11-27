package domain_test

import (
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

// Dates many tests share.
const (
	nov01   = "2026-11-01"
	nov02   = "2026-11-02"
	nov27   = "2026-11-27"
	nov28   = "2026-11-28"
	nov30   = "2026-11-30"
	dec31   = "2026-12-31"
	leapDay = "2028-02-29"
)

// date parses a "YYYY-MM-DD" literal or fails the test.
func date(t *testing.T, s string) domain.Date {
	t.Helper()
	d, err := domain.ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", s, err)
	}
	return d
}

// month parses a "YYYY-MM" literal or fails the test.
func month(t *testing.T, s string) domain.Month {
	t.Helper()
	m, err := domain.ParseMonth(s)
	if err != nil {
		t.Fatalf("ParseMonth(%q) error = %v", s, err)
	}
	return m
}
