package domain_test

import (
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
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
