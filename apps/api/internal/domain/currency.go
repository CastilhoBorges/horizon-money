package domain

import "golang.org/x/text/currency"

// Currency is an ISO 4217 currency and the number of decimals of its minor unit.
type Currency struct {
	code  string
	scale int
}

// ParseCurrency reads an ISO 4217 code, in any letter case. The minor-unit scale comes from the ISO 4217
// data in golang.org/x/text: BRL, USD and CAD have 2 decimals, JPY 0 and KWD 3.
func ParseCurrency(code string) (Currency, error) {
	unit, err := currency.ParseISO(code)
	// x/text maps "XXX" (no currency) to the zero Unit without an error.
	if err != nil || unit == (currency.Unit{}) {
		return Currency{}, ErrUnknownCurrency
	}
	scale, _ := currency.Standard.Rounding(unit)
	return Currency{code: unit.String(), scale: scale}, nil
}

// Code is the upper-case ISO 4217 code, such as "BRL".
func (c Currency) Code() string { return c.code }

// Scale is the number of decimals of the minor unit.
func (c Currency) Scale() int { return c.scale }
