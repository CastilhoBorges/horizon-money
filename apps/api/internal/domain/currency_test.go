package domain_test

import (
	"errors"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

const (
	codeBRL = "BRL"
	codeJPY = "JPY"
	codeKWD = "KWD"
)

func TestParseCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expectedCode  string
		expectedScale int
	}{
		{name: "brazilian real", input: codeBRL, expectedCode: codeBRL, expectedScale: 2},
		{name: "lowercase code", input: "brl", expectedCode: codeBRL, expectedScale: 2},
		{name: "us dollar", input: "USD", expectedCode: "USD", expectedScale: 2},
		{name: "canadian dollar", input: "CAD", expectedCode: "CAD", expectedScale: 2},
		{name: "yen has no minor unit", input: codeJPY, expectedCode: codeJPY, expectedScale: 0},
		{name: "dinar has three decimals", input: codeKWD, expectedCode: codeKWD, expectedScale: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseCurrency(tt.input)
			if err != nil {
				t.Fatalf("ParseCurrency(%q) error = %v", tt.input, err)
			}
			if got.Code() != tt.expectedCode || got.Scale() != tt.expectedScale {
				t.Errorf("ParseCurrency(%q) = %s/%d, want %s/%d",
					tt.input, got.Code(), got.Scale(), tt.expectedCode, tt.expectedScale)
			}
		})
	}
}

func TestParseCurrencyRejectsUnknownCodes(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "XXX", "ABC", "BRLX", "R$"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.ParseCurrency(input); !errors.Is(err, domain.ErrUnknownCurrency) {
				t.Errorf("ParseCurrency(%q) error = %v, want %v", input, err, domain.ErrUnknownCurrency)
			}
		})
	}
}
