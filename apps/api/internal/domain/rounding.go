package domain

// Every rounding rule of the domain lives in this file (§6.2): floor plus remainder for installments,
// and half-up for divisions.

// splitParts splits total into parts. Every part is total ÷ parts rounded down to the minor unit, and the
// last part takes the remainder, so the parts always sum exactly to the total.
func splitParts(total Amount, parts int) (part, last Amount) {
	part = total / Amount(parts)
	return part, total - part*Amount(parts-1)
}
