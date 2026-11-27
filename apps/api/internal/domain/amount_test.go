package domain_test

import (
	"errors"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func currency(t *testing.T, code string) domain.Currency {
	t.Helper()
	c, err := domain.ParseCurrency(code)
	if err != nil {
		t.Fatalf("ParseCurrency(%q) error = %v", code, err)
	}
	return c
}

func TestParseAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		currency string
		expected domain.Amount
	}{
		{name: "whole units", input: "26", currency: codeBRL, expected: 2600},
		{name: "one decimal", input: "26.5", currency: codeBRL, expected: 2650},
		{name: "full scale", input: "26.00", currency: codeBRL, expected: 2600},
		{name: "one cent", input: "0.01", currency: codeBRL, expected: 1},
		{name: "negative", input: "-41.67", currency: codeBRL, expected: -4167},
		{name: "negative zero", input: "-0.00", currency: codeBRL, expected: 0},
		{name: "largest amount", input: "99999999999.99", currency: codeBRL, expected: 9999999999999},
		{name: "yen", input: "1500", currency: codeJPY, expected: 1500},
		{name: "dinar", input: "1.234", currency: codeKWD, expected: 1234},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseAmount(tt.input, currency(t, tt.currency))
			if err != nil {
				t.Fatalf("ParseAmount(%q) error = %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("ParseAmount(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseAmountRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		currency string
		expected error
	}{
		{name: "empty", input: "", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "letters", input: "abc", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "decimal comma", input: "26,00", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "thousands separator", input: "1,000.00", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "no integer part", input: ".5", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "no fraction part", input: "5.", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "plus sign", input: "+5", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "leading space", input: " 5", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "trailing space", input: "5 ", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "exponent", input: "1e3", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "double minus", input: "--1", currency: codeBRL, expected: domain.ErrInvalidAmount},
		{name: "beyond the cent", input: "26.001", currency: codeBRL, expected: domain.ErrTooManyDecimals},
		{name: "decimals on yen", input: "1500.0", currency: codeJPY, expected: domain.ErrTooManyDecimals},
		{name: "above the maximum", input: "100000000000.00", currency: codeBRL, expected: domain.ErrAmountTooLarge},
		{name: "below the negative maximum", input: "-100000000000.00", currency: codeBRL, expected: domain.ErrAmountTooLarge},
		{name: "beyond int64", input: "99999999999999999999999", currency: codeBRL, expected: domain.ErrAmountTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseAmount(tt.input, currency(t, tt.currency))
			if !errors.Is(err, tt.expected) {
				t.Errorf("ParseAmount(%q) = %d, %v; want error %v", tt.input, got, err, tt.expected)
			}
		})
	}
}

func TestAmount_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   domain.Amount
		currency string
		expected string
	}{
		{name: "reais and cents", amount: 142245, currency: codeBRL, expected: "1422.45"},
		{name: "negative", amount: -4167, currency: codeBRL, expected: "-41.67"},
		{name: "cents only", amount: 5, currency: codeBRL, expected: "0.05"},
		{name: "negative cents only", amount: -5, currency: codeBRL, expected: "-0.05"},
		{name: "zero", amount: 0, currency: codeBRL, expected: "0.00"},
		{name: "yen", amount: 1500, currency: codeJPY, expected: "1500"},
		{name: "dinar", amount: 1234, currency: codeKWD, expected: "1.234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := currency(t, tt.currency)
			got := tt.amount.Format(c)
			if got != tt.expected {
				t.Errorf("Amount(%d).Format(%s) = %q, want %q", tt.amount, tt.currency, got, tt.expected)
			}
			back, err := domain.ParseAmount(got, c)
			if err != nil || back != tt.amount {
				t.Errorf("ParseAmount(%q) = %d, %v; want the round trip to give %d", got, back, err, tt.amount)
			}
		})
	}
}
