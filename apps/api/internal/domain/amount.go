package domain

import (
	"regexp"
	"strconv"
	"strings"
)

// Amount is money in minor units of one currency (cents for BRL). It is never a float.
type Amount int64

// MaxAmount is the largest absolute amount accepted at the edge. It keeps every sum the domain makes far
// from the int64 limit.
const MaxAmount Amount = 9_999_999_999_999

// maxSafeDigits is how many decimal digits always fit in an int64.
const maxSafeDigits = 18

var amountPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// ParseAmount reads a decimal string such as "26", "26.5" or "-41.67" in the currency's scale. It refuses
// signs other than a leading minus, separators, spaces, exponents and more decimals than the currency has.
func ParseAmount(s string, c Currency) (Amount, error) {
	if !amountPattern.MatchString(s) {
		return 0, ErrInvalidAmount
	}
	unsigned, negative := strings.CutPrefix(s, "-")
	whole, fraction, _ := strings.Cut(unsigned, ".")
	if len(fraction) > c.scale {
		return 0, ErrTooManyDecimals
	}

	digits := strings.TrimLeft(whole+fraction+strings.Repeat("0", c.scale-len(fraction)), "0")
	if digits == "" {
		return 0, nil
	}
	if len(digits) > maxSafeDigits {
		return 0, ErrAmountTooLarge
	}
	value, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || Amount(value) > MaxAmount {
		return 0, ErrAmountTooLarge
	}
	if negative {
		value = -value
	}
	return Amount(value), nil
}

// Format writes the amount as a decimal string in the currency's scale, such as "1422.45" or "-0.05".
// It is the inverse of ParseAmount.
func (a Amount) Format(c Currency) string {
	digits, negative := strings.CutPrefix(strconv.FormatInt(int64(a), 10), "-")
	sign := ""
	if negative {
		sign = "-"
	}
	if c.scale == 0 {
		return sign + digits
	}
	if len(digits) <= c.scale {
		digits = strings.Repeat("0", c.scale-len(digits)+1) + digits
	}
	cut := len(digits) - c.scale
	return sign + digits[:cut] + "." + digits[cut:]
}
