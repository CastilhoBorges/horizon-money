package domain

import "slices"

// Frequency is how a series repeats.
type Frequency string

// The ways a movement repeats (§4.5). A one-off movement is not a series.
const (
	FrequencyDaily        Frequency = "daily"
	FrequencyWeekly       Frequency = "weekly"
	FrequencyMonthly      Frequency = "monthly"
	FrequencyInstallments Frequency = "installments"
)

// Installment counts accepted at creation (§4.5).
const (
	MinInstallments = 2
	MaxInstallments = 360
)

// Series is the rule that produces a repeated movement's occurrences. Occurrences are expanded on read
// for the range asked (§6.3). A following split turns one series into two that share a series group.
type Series struct {
	Type      MovementType
	Frequency Frequency
	// Start is the first occurrence.
	Start Date
	// End is the last day an occurrence may fall on, inclusive. The zero Date means no end.
	End Date
	// AnchorDay is the day of the month a monthly series or an installment falls on (1–31). In shorter
	// months it falls on the last day. A split keeps it, so the 31st survives February.
	AnchorDay int
	// Amount is every occurrence's amount; for installments, every part but the last.
	Amount Amount
	// LastAmount is the last installment, which takes the remainder of the total.
	LastAmount Amount
	// Parts is the installment count N.
	Parts int
	// FirstPart is the part number of Start: 1 for a new purchase, later for the second half of a split.
	FirstPart int
}

// Occurrence is one dated instance of a series.
type Occurrence struct {
	// Date is where the occurrence counts, after any override.
	Date Date
	// Original is the date the series produced. Exceptions are keyed by it.
	Original Date
	Amount   Amount
	// Part and Parts number an installment ("3/12"). They are 0 outside installments.
	Part  int
	Parts int
}

// Exception changes one occurrence of a series, keyed by the date the series produced. It either
// deletes the occurrence or overrides it with a full snapshot of its date and amount, which a later
// edit of the whole series never touches.
type Exception struct {
	Deleted bool
	Date    Date
	Amount  Amount
}

// NewRecurring builds a daily, weekly or monthly series with the same amount every time. The zero end
// Date means it never ends. A monthly series falls on the start's day of the month.
func NewRecurring(t MovementType, f Frequency, amount Amount, start, end Date) (Series, error) {
	if !t.isValid() {
		return Series{}, ErrUnknownMovementType
	}
	if f != FrequencyDaily && f != FrequencyWeekly && f != FrequencyMonthly {
		return Series{}, ErrUnknownFrequency
	}
	if amount <= 0 {
		return Series{}, ErrAmountNotPositive
	}
	if !end.IsZero() && end.Before(start) {
		return Series{}, ErrEndBeforeStart
	}
	return Series{Type: t, Frequency: f, Start: start, End: end, AnchorDay: start.Day(), Amount: amount}, nil
}

// NewInstallments splits total into parts monthly parts from first, numbered 1 to parts. Each part is the
// total ÷ parts rounded down to the minor unit and the last part takes the remainder, so the parts sum
// exactly to the total. A total below one minor unit per part is refused.
func NewInstallments(t MovementType, total Amount, parts int, first Date) (Series, error) {
	if !t.isValid() {
		return Series{}, ErrUnknownMovementType
	}
	if total <= 0 {
		return Series{}, ErrAmountNotPositive
	}
	if parts < MinInstallments || parts > MaxInstallments {
		return Series{}, ErrInstallmentsOutOfRange
	}
	part, last := splitParts(total, parts)
	if part == 0 {
		return Series{}, ErrInstallmentPartTooSmall
	}
	return Series{
		Type: t, Frequency: FrequencyInstallments, Start: first, AnchorDay: first.Day(),
		Amount: part, LastAmount: last, Parts: parts, FirstPart: 1,
	}, nil
}

// Validate checks a series rebuilt from storage, so a corrupt row surfaces as an error instead of a
// wrong balance. The constructors and edit methods always return valid series.
func (s Series) Validate() error {
	switch {
	case !s.Type.isValid():
		return ErrUnknownMovementType
	case s.Start.IsZero():
		return ErrInvalidDate
	case s.Amount <= 0:
		return ErrAmountNotPositive
	case !s.End.IsZero() && s.End.Before(s.Start):
		return ErrEndBeforeStart
	case s.AnchorDay < 1 || s.AnchorDay > 31:
		return ErrInvalidSeries
	}
	switch s.Frequency {
	case FrequencyDaily, FrequencyWeekly:
		return s.validateNoParts()
	case FrequencyMonthly:
		if err := s.validateNoParts(); err != nil {
			return err
		}
		return s.validateAnchor()
	case FrequencyInstallments:
		return s.validateParts()
	}
	return ErrUnknownFrequency
}

func (s Series) validateNoParts() error {
	if s.LastAmount != 0 || s.Parts != 0 || s.FirstPart != 0 {
		return ErrInvalidSeries
	}
	return nil
}

// validateAnchor checks that a monthly series or an installment starts on its anchor day.
func (s Series) validateAnchor() error {
	if s.Start != anchoredDate(MonthOf(s.Start), s.AnchorDay) {
		return ErrInvalidSeries
	}
	return nil
}

func (s Series) validateParts() error {
	switch {
	case s.Parts < MinInstallments || s.Parts > MaxInstallments:
		return ErrInstallmentsOutOfRange
	case s.LastAmount <= 0:
		return ErrAmountNotPositive
	case s.FirstPart < 1 || s.FirstPart > s.Parts:
		return ErrInvalidSeries
	}
	return s.validateAnchor()
}

// Expand lists the occurrences that count between from and to, inclusive, sorted by date. Exceptions
// apply by the date the series produced: an override moved into the range appears and one moved out of
// it disappears. Keys the series never produces are ignored.
func (s Series) Expand(from, to Date, exceptions map[Date]Exception) []Occurrence {
	var out []Occurrence
	for _, original := range s.ruleDates(from, to) {
		e, ok := exceptions[original]
		switch {
		case !ok:
			out = append(out, s.occurrence(original, original, s.amountOf(original)))
		case !e.Deleted && inRange(e.Date, from, to):
			out = append(out, s.occurrence(e.Date, original, e.Amount))
		}
	}
	for original, e := range exceptions {
		if e.Deleted || inRange(original, from, to) || !inRange(e.Date, from, to) || !s.isRuleDate(original) {
			continue
		}
		out = append(out, s.occurrence(e.Date, original, e.Amount))
	}
	slices.SortFunc(out, func(a, b Occurrence) int {
		if c := a.Date.Compare(b.Date); c != 0 {
			return c
		}
		return a.Original.Compare(b.Original)
	})
	return out
}

// SplitAt cuts the series for a following edit: head ends the day before at, and tail starts on that
// date with every other field kept, the anchor day included. The caller then applies the new values to tail.
// Splitting at the first occurrence is refused; edit the series itself instead.
func (s Series) SplitAt(at Date) (head, tail Series, err error) {
	if !s.isRuleDate(at) {
		return Series{}, Series{}, ErrNotAnOccurrence
	}
	if at == s.Start {
		return Series{}, Series{}, ErrSplitAtFirstOccurrence
	}
	head, tail = s, s
	head.End = at.AddDays(-1)
	tail.Start = at
	if s.Frequency == FrequencyInstallments {
		tail.FirstPart = s.partOf(at)
	}
	return head, tail, nil
}

// WithAmount sets the amount of every occurrence, for a following or all edit. For installments it is
// the value per part from there on, the last part included; the total is whatever the parts now sum to.
func (s Series) WithAmount(a Amount) (Series, error) {
	if a <= 0 {
		return Series{}, ErrAmountNotPositive
	}
	s.Amount = a
	if s.Frequency == FrequencyInstallments {
		s.LastAmount = a
	}
	return s, nil
}

// WithStart moves the series so its first occurrence falls on d, for a following or all edit that changes
// the date. A monthly series or an installment then falls on d's day of the month. Installments keep their
// part numbers: the part that fell on Start now falls on d.
func (s Series) WithStart(d Date) (Series, error) {
	if d.IsZero() {
		return Series{}, ErrInvalidDate
	}
	if !s.End.IsZero() && s.End.Before(d) {
		return Series{}, ErrEndBeforeStart
	}
	s.Start, s.AnchorDay = d, d.Day()
	return s, nil
}

func (s Series) occurrence(at, original Date, amount Amount) Occurrence {
	o := Occurrence{Date: at, Original: original, Amount: amount}
	if s.Frequency == FrequencyInstallments {
		o.Part, o.Parts = s.partOf(original), s.Parts
	}
	return o
}

// amountOf is the template amount of the occurrence the series produces on original.
func (s Series) amountOf(original Date) Amount {
	if s.Frequency == FrequencyInstallments && s.partOf(original) == s.Parts {
		return s.LastAmount
	}
	return s.Amount
}

// partOf is the installment number of a date the series produces.
func (s Series) partOf(original Date) int {
	return s.FirstPart + monthsBetween(MonthOf(s.Start), MonthOf(original))
}

// lastDate is the last day the series may produce: End, or the last part's date for installments,
// whichever comes first. ok is false for a series with no end.
func (s Series) lastDate() (last Date, ok bool) {
	last = s.End
	if s.Frequency == FrequencyInstallments {
		lastPart := anchoredDate(MonthOf(s.Start).Add(s.Parts-s.FirstPart), s.AnchorDay)
		if last.IsZero() || lastPart.Before(last) {
			last = lastPart
		}
	}
	return last, !last.IsZero()
}

// ruleDates lists the dates the series produces between from and to, inclusive, without walking the
// days before from.
func (s Series) ruleDates(from, to Date) []Date {
	lo, hi := later(s.Start, from), to
	if last, ok := s.lastDate(); ok && last.Before(hi) {
		hi = last
	}
	if hi.Before(lo) {
		return nil
	}

	var out []Date
	switch s.Frequency {
	case FrequencyDaily:
		for d := lo; !d.After(hi); d = d.AddDays(1) {
			out = append(out, d)
		}
	case FrequencyWeekly:
		first := s.Start.AddDays((lo.DaysSince(s.Start) + 6) / 7 * 7)
		for d := first; !d.After(hi); d = d.AddDays(7) {
			out = append(out, d)
		}
	case FrequencyMonthly, FrequencyInstallments:
		for m := MonthOf(lo); !m.First().After(hi); m = m.Add(1) {
			if d := anchoredDate(m, s.AnchorDay); inRange(d, lo, hi) {
				out = append(out, d)
			}
		}
	}
	return out
}

// isRuleDate reports whether the series produces d.
func (s Series) isRuleDate(d Date) bool {
	if last, ok := s.lastDate(); d.Before(s.Start) || (ok && d.After(last)) {
		return false
	}
	switch s.Frequency {
	case FrequencyDaily:
		return true
	case FrequencyWeekly:
		return d.DaysSince(s.Start)%7 == 0
	case FrequencyMonthly, FrequencyInstallments:
		return d == anchoredDate(MonthOf(d), s.AnchorDay)
	}
	return false
}

// anchoredDate is the anchor day in month m, or the month's last day when the month is shorter.
func anchoredDate(m Month, anchorDay int) Date {
	return m.First().AddDays(min(anchorDay, m.Days()) - 1)
}

func inRange(d, from, to Date) bool { return !d.Before(from) && !d.After(to) }
