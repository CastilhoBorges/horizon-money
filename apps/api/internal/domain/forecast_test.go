package domain_test

import (
	"errors"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func TestDailyValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		monthly  []domain.Amount
		divisor  int
		expected domain.Amount
	}{
		{name: "the spec example", monthly: []domain.Amount{100000}, divisor: 30, expected: 3333},
		{name: "the acceptance fixture", monthly: []domain.Amount{35000, 30000, 60000}, divisor: 30, expected: 4167},
		{name: "exactly half rounds up", monthly: []domain.Amount{105}, divisor: 30, expected: 4},
		{name: "below half rounds down", monthly: []domain.Amount{104}, divisor: 30, expected: 3},
		{name: "smallest divisor", monthly: []domain.Amount{100000}, divisor: 28, expected: 3571},
		{name: "largest divisor", monthly: []domain.Amount{100000}, divisor: 31, expected: 3226},
		{name: "no categories", monthly: nil, divisor: 30, expected: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.DailyValue(tt.monthly, tt.divisor)
			if err != nil {
				t.Fatalf("DailyValue(%v, %d) error = %v", tt.monthly, tt.divisor, err)
			}
			if got != tt.expected {
				t.Errorf("DailyValue(%v, %d) = %d, want %d", tt.monthly, tt.divisor, got, tt.expected)
			}
		})
	}
}

func TestDailyValueRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		monthly  []domain.Amount
		divisor  int
		expected error
	}{
		{name: "divisor below 28", monthly: []domain.Amount{100000}, divisor: 27, expected: domain.ErrForecastDivisorOutOfRange},
		{name: "divisor above 31", monthly: []domain.Amount{100000}, divisor: 32, expected: domain.ErrForecastDivisorOutOfRange},
		{name: "zero category", monthly: []domain.Amount{100000, 0}, divisor: 30, expected: domain.ErrForecastAmountNotPositive},
		{name: "negative category", monthly: []domain.Amount{-1}, divisor: 30, expected: domain.ErrForecastAmountNotPositive},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.DailyValue(tt.monthly, tt.divisor); !errors.Is(err, tt.expected) {
				t.Errorf("DailyValue(%v, %d) error = %v, want %v", tt.monthly, tt.divisor, err, tt.expected)
			}
		})
	}
}
