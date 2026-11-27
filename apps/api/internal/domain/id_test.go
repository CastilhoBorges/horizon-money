package domain_test

import (
	"errors"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func TestParseID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "canonical", input: "01a21ec4-e9fa-7aa5-8edc-d14ef635e779", expected: "01a21ec4-e9fa-7aa5-8edc-d14ef635e779"},
		{name: "upper case is normalised", input: "01A21EC4-E9FA-7AA5-8EDC-D14EF635E779", expected: "01a21ec4-e9fa-7aa5-8edc-d14ef635e779"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseID(tt.input)
			if err != nil {
				t.Fatalf("ParseID(%q) error = %v", tt.input, err)
			}
			if got.String() != tt.expected || got.IsZero() {
				t.Errorf("ParseID(%q) = %s (zero %v), want %s", tt.input, got, got.IsZero(), tt.expected)
			}
		})
	}
}

func TestParseIDRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"",
		"42",
		"01a21ec4e9fa7aa58edcd14ef635e779",
		"01a21ec4-e9fa-7aa5-8edc-d14ef635e77",
		"01a21ec4-e9fa-7aa5-8edc-d14ef635e7799",
		"01a21ec4-e9fa-7aa5-8edc_d14ef635e779",
		"01a21ec4-e9fa-7aa5-8edc-d14ef635e77g",
		"{01a21ec4-e9fa-7aa5-8edc-d14ef635e779}",
		"00000000-0000-0000-0000-000000000000",
		"01a21ec4-e9fa-7aa5-8edc-d14ef635--79",
		"01a21ec4-e9fa-7aa5-8edc-------------",
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.ParseID(input); !errors.Is(err, domain.ErrInvalidID) {
				t.Errorf("ParseID(%q) error = %v, want %v", input, err, domain.ErrInvalidID)
			}
		})
	}
}

func TestID_IsZero(t *testing.T) {
	t.Parallel()

	if !(domain.ID{}).IsZero() {
		t.Error("the zero ID must report IsZero")
	}
}
