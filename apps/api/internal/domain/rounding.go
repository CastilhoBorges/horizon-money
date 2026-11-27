package domain

// Every rounding rule of the domain lives in this file (§6.2): floor plus remainder for installments,
// and half-up for divisions.

// splitParts splits total into parts. Every part is total ÷ parts rounded down to the minor unit, and the
// last part takes the remainder, so the parts always sum exactly to the total.
func splitParts(total Amount, parts int) (part, last Amount) {
	part = total / Amount(parts)
	return part, total - part*Amount(parts-1)
}

// divHalfUp divides n by d rounding half-up to the nearest integer, for n ≥ 0 and d > 0:
// 41.666… becomes 41.67 and an exact half goes up.
func divHalfUp(n, d int64) int64 {
	quotient, remainder := n/d, n%d
	if remainder*2 >= d {
		quotient++
	}
	return quotient
}
