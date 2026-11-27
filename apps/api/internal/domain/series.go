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

// Expand lists the occurrences that count between from and to, inclusive, sorted by date. Exceptions
// apply by the date the series produced: an override moved into the range appears and one moved out of
// it disappears. Keys the series never produces are ignored.
func (s Series) Expand(from, to Date, exceptions map[Date]Exception) []Occurrence {
	var out []Occurrence
	for _, original := range s.ruleDates(from, to) {
		e, ok := exceptions[original]
		switch {
		case !ok:
			out = append(out, s.occurrence(original, original, s.Amount))
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
	return head, tail, nil
}

// WithAmount sets the amount of every occurrence, for a following or all edit.
func (s Series) WithAmount(a Amount) (Series, error) {
	if a <= 0 {
		return Series{}, ErrAmountNotPositive
	}
	s.Amount = a
	return s, nil
}

func (s Series) occurrence(at, original Date, amount Amount) Occurrence {
	return Occurrence{Date: at, Original: original, Amount: amount}
}

// ruleDates lists the dates the series produces between from and to, inclusive, without walking the
// days before from.
func (s Series) ruleDates(from, to Date) []Date {
	lo, hi := s.Start, to
	if from.After(lo) {
		lo = from
	}
	if !s.End.IsZero() && s.End.Before(hi) {
		hi = s.End
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
	if d.Before(s.Start) || (!s.End.IsZero() && d.After(s.End)) {
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
