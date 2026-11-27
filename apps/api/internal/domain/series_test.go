package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

const (
	jan10 = "2026-01-10"
	feb10 = "2026-02-10"
	mar10 = "2026-03-10"
	apr10 = "2026-04-10"
	jan31 = "2026-01-31"
	feb28 = "2026-02-28"
	mar31 = "2026-03-31"
	apr30 = "2026-04-30"
)

// recurring builds a series or fails the test. An empty end means no end date.
func recurring(t *testing.T, f domain.Frequency, amount domain.Amount, start, end string) domain.Series {
	t.Helper()
	var endDate domain.Date
	if end != "" {
		endDate = date(t, end)
	}
	s, err := domain.NewRecurring(domain.TypeExpense, f, amount, date(t, start), endDate)
	if err != nil {
		t.Fatalf("NewRecurring(%s, %s..%s) error = %v", f, start, end, err)
	}
	return s
}

// dates lists the dates where the occurrences count, as "YYYY-MM-DD".
func dates(occurrences []domain.Occurrence) []string {
	out := make([]string, 0, len(occurrences))
	for _, o := range occurrences {
		out = append(out, o.Date.String())
	}
	return out
}

func TestNewRecurring(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		movement  domain.MovementType
		frequency domain.Frequency
		amount    domain.Amount
		start     string
		end       string
		expected  error
	}{
		{
			name: "end before start", movement: domain.TypeExpense, frequency: domain.FrequencyMonthly,
			amount: 1000, start: nov02, end: nov01, expected: domain.ErrEndBeforeStart,
		},
		{
			name: "zero amount", movement: domain.TypeExpense, frequency: domain.FrequencyMonthly,
			amount: 0, start: nov02, expected: domain.ErrAmountNotPositive,
		},
		{
			name: "negative amount", movement: domain.TypeExpense, frequency: domain.FrequencyMonthly,
			amount: -1, start: nov02, expected: domain.ErrAmountNotPositive,
		},
		{
			name: "unknown frequency", movement: domain.TypeExpense, frequency: "yearly",
			amount: 1000, start: nov02, expected: domain.ErrUnknownFrequency,
		},
		{
			name: "installments need their own constructor", movement: domain.TypeExpense,
			frequency: domain.FrequencyInstallments, amount: 1000, start: nov02, expected: domain.ErrUnknownFrequency,
		},
		{
			name: "a transfer is not a movement type", movement: "transfer", frequency: domain.FrequencyMonthly,
			amount: 1000, start: nov02, expected: domain.ErrUnknownMovementType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var end domain.Date
			if tt.end != "" {
				end = date(t, tt.end)
			}
			_, err := domain.NewRecurring(tt.movement, tt.frequency, tt.amount, date(t, tt.start), end)
			if !errors.Is(err, tt.expected) {
				t.Errorf("NewRecurring error = %v, want %v", err, tt.expected)
			}
		})
	}
}

func TestSeries_Expand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		frequency domain.Frequency
		start     string
		end       string
		from      string
		to        string
		expected  []string
	}{
		{
			name: "monthly on the 31st falls on the last day of shorter months", frequency: domain.FrequencyMonthly,
			start: jan31, from: "2026-01-01", to: "2026-06-30",
			expected: []string{jan31, feb28, mar31, apr30, "2026-05-31", "2026-06-30"},
		},
		{
			name: "monthly on the 30th reaches february 29 in a leap year", frequency: domain.FrequencyMonthly,
			start: "2028-01-30", from: "2028-02-01", to: leapDay, expected: []string{leapDay},
		},
		{
			name: "monthly on the 29th comes back to the 29th", frequency: domain.FrequencyMonthly,
			start: "2026-01-29", from: "2026-02-01", to: "2026-03-31", expected: []string{feb28, "2026-03-29"},
		},
		{
			name: "weekly", frequency: domain.FrequencyWeekly, start: nov02, from: nov01, to: nov30,
			expected: []string{nov02, "2026-11-09", "2026-11-16", "2026-11-23", nov30},
		},
		{
			name: "weekly that began years ago", frequency: domain.FrequencyWeekly, start: "2020-01-06",
			from: nov01, to: nov30,
			expected: []string{nov02, "2026-11-09", "2026-11-16", "2026-11-23", nov30},
		},
		{
			name: "daily with an end date", frequency: domain.FrequencyDaily, start: "2026-11-28", end: nov30,
			from: nov01, to: dec31, expected: []string{"2026-11-28", "2026-11-29", nov30},
		},
		{
			name: "rent that began years ago", frequency: domain.FrequencyMonthly, start: "2020-01-05",
			from: nov01, to: nov30, expected: []string{"2026-11-05"},
		},
		{
			name: "range before the start", frequency: domain.FrequencyMonthly, start: "2026-11-05",
			from: "2026-10-01", to: "2026-10-31", expected: []string{},
		},
		{
			name: "range after the end", frequency: domain.FrequencyDaily, start: nov01, end: "2026-11-03",
			from: "2026-11-04", to: nov30, expected: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := recurring(t, tt.frequency, 1000, tt.start, tt.end).Expand(date(t, tt.from), date(t, tt.to), nil)
			if !slices.Equal(dates(got), tt.expected) {
				t.Fatalf("Expand(%s..%s) = %v, want %v", tt.from, tt.to, dates(got), tt.expected)
			}
			for _, o := range got {
				if o.Amount != 1000 || o.Original != o.Date || o.Part != 0 || o.Parts != 0 {
					t.Errorf("occurrence %+v: want amount 1000, original = date, no part numbers", o)
				}
			}
		})
	}
}

func TestSeries_ExpandWithExceptions(t *testing.T) {
	t.Parallel()

	type occurrence struct {
		date, original string
		amount         domain.Amount
	}
	tests := []struct {
		name       string
		exceptions map[string]domain.Exception
		expected   []occurrence
	}{
		{name: "no exceptions", expected: []occurrence{
			{feb10, feb10, 100000}, {mar10, mar10, 100000}, {apr10, apr10, 100000},
		}},
		{
			name: "a deleted occurrence", exceptions: map[string]domain.Exception{mar10: {Deleted: true}},
			expected: []occurrence{{feb10, feb10, 100000}, {apr10, apr10, 100000}},
		},
		{
			name:       "an override moves and changes one occurrence",
			exceptions: map[string]domain.Exception{feb10: {Date: date(t, "2026-02-12"), Amount: 120000}},
			expected:   []occurrence{{"2026-02-12", feb10, 120000}, {mar10, mar10, 100000}, {apr10, apr10, 100000}},
		},
		{
			name:       "an override moved into the range appears",
			exceptions: map[string]domain.Exception{jan10: {Date: date(t, "2026-02-01"), Amount: 100000}},
			expected: []occurrence{
				{"2026-02-01", jan10, 100000}, {feb10, feb10, 100000}, {mar10, mar10, 100000}, {apr10, apr10, 100000},
			},
		},
		{
			name:       "an override moved out of the range disappears",
			exceptions: map[string]domain.Exception{apr10: {Date: date(t, "2026-05-02"), Amount: 100000}},
			expected:   []occurrence{{feb10, feb10, 100000}, {mar10, mar10, 100000}},
		},
		{
			name:       "a key the series never produces is ignored",
			exceptions: map[string]domain.Exception{"2026-02-11": {Deleted: true}},
			expected:   []occurrence{{feb10, feb10, 100000}, {mar10, mar10, 100000}, {apr10, apr10, 100000}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exceptions := map[domain.Date]domain.Exception{}
			for key, e := range tt.exceptions {
				exceptions[date(t, key)] = e
			}
			series := recurring(t, domain.FrequencyMonthly, 100000, jan10, "")
			got := series.Expand(date(t, "2026-02-01"), date(t, apr30), exceptions)

			want := make([]domain.Occurrence, 0, len(tt.expected))
			for _, o := range tt.expected {
				want = append(want, domain.Occurrence{Date: date(t, o.date), Original: date(t, o.original), Amount: o.amount})
			}
			if !slices.Equal(got, want) {
				t.Errorf("Expand = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSeries_SplitAt(t *testing.T) {
	t.Parallel()

	series := recurring(t, domain.FrequencyMonthly, 1000, jan31, "")
	head, tail, err := series.SplitAt(date(t, feb28))
	if err != nil {
		t.Fatalf("SplitAt(%s) error = %v", feb28, err)
	}
	if got := dates(head.Expand(date(t, "2026-01-01"), date(t, dec31), nil)); !slices.Equal(got, []string{jan31}) {
		t.Errorf("head = %v, want [%s]", got, jan31)
	}
	if got := dates(tail.Expand(date(t, "2026-02-01"), date(t, apr30), nil)); !slices.Equal(got, []string{feb28, mar31, apr30}) {
		t.Errorf("tail = %v, want the 31st anchor kept: [%s %s %s]", got, feb28, mar31, apr30)
	}

	if _, _, err := series.SplitAt(date(t, "2026-02-27")); !errors.Is(err, domain.ErrNotAnOccurrence) {
		t.Errorf("SplitAt(a date the series never produces) error = %v, want %v", err, domain.ErrNotAnOccurrence)
	}
	if _, _, err := series.SplitAt(date(t, jan31)); !errors.Is(err, domain.ErrSplitAtFirstOccurrence) {
		t.Errorf("SplitAt(the first occurrence) error = %v, want %v", err, domain.ErrSplitAtFirstOccurrence)
	}
}

func TestSeries_WithAmount(t *testing.T) {
	t.Parallel()

	series := recurring(t, domain.FrequencyMonthly, 1000, jan10, "")
	if _, err := series.WithAmount(0); !errors.Is(err, domain.ErrAmountNotPositive) {
		t.Errorf("WithAmount(0) error = %v, want %v", err, domain.ErrAmountNotPositive)
	}
	edited, err := series.WithAmount(5000)
	if err != nil {
		t.Fatalf("WithAmount(5000) error = %v", err)
	}
	for _, o := range edited.Expand(date(t, jan10), date(t, mar10), nil) {
		if o.Amount != 5000 {
			t.Errorf("occurrence on %s has amount %d, want 5000", o.Date, o.Amount)
		}
	}
}

// installments builds an installment series of card spending or fails the test.
func installments(t *testing.T, total domain.Amount, parts int, first string) domain.Series {
	t.Helper()
	s, err := domain.NewInstallments(domain.TypeCard, total, parts, date(t, first))
	if err != nil {
		t.Fatalf("NewInstallments(%d, %d, %s) error = %v", total, parts, first, err)
	}
	return s
}

// part is the date, part number and amount of one installment.
type part struct {
	date   string
	number int
	amount domain.Amount
}

func parts(occurrences []domain.Occurrence) []part {
	out := make([]part, 0, len(occurrences))
	for _, o := range occurrences {
		out = append(out, part{date: o.Date.String(), number: o.Part, amount: o.Amount})
	}
	return out
}

func TestNewInstallments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		total        domain.Amount
		parts        int
		expectedPart domain.Amount
		expectedLast domain.Amount
	}{
		{name: "100.00 in 3", total: 10000, parts: 3, expectedPart: 3333, expectedLast: 3334},
		{name: "1000.00 in 12", total: 100000, parts: 12, expectedPart: 8333, expectedLast: 8337},
		{name: "10.00 in 360", total: 1000, parts: 360, expectedPart: 2, expectedLast: 282},
		{name: "0.02 in 2", total: 2, parts: 2, expectedPart: 1, expectedLast: 1},
		{name: "7.00 in 7", total: 700, parts: 7, expectedPart: 100, expectedLast: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := installments(t, tt.total, tt.parts, jan10).Expand(date(t, jan10), date(t, "2060-12-31"), nil)
			if len(got) != tt.parts {
				t.Fatalf("expanded %d parts, want %d", len(got), tt.parts)
			}
			var sum domain.Amount
			for i, o := range got {
				expected := tt.expectedPart
				if i == tt.parts-1 {
					expected = tt.expectedLast
				}
				if o.Amount != expected || o.Part != i+1 || o.Parts != tt.parts {
					t.Errorf("part %d = %d (%d/%d), want %d (%d/%d)", i+1, o.Amount, o.Part, o.Parts, expected, i+1, tt.parts)
				}
				sum += o.Amount
			}
			if sum != tt.total {
				t.Errorf("parts sum to %d, want the total %d", sum, tt.total)
			}
		})
	}
}

func TestNewInstallmentsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		movement domain.MovementType
		total    domain.Amount
		parts    int
		expected error
	}{
		{name: "one part", movement: domain.TypeCard, total: 10000, parts: 1, expected: domain.ErrInstallmentsOutOfRange},
		{name: "361 parts", movement: domain.TypeCard, total: 10000, parts: 361, expected: domain.ErrInstallmentsOutOfRange},
		{name: "less than a cent per part", movement: domain.TypeCard, total: 5, parts: 12, expected: domain.ErrInstallmentPartTooSmall},
		{name: "zero total", movement: domain.TypeCard, total: 0, parts: 3, expected: domain.ErrAmountNotPositive},
		{name: "a transfer is not a movement type", movement: "transfer", total: 10000, parts: 3, expected: domain.ErrUnknownMovementType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewInstallments(tt.movement, tt.total, tt.parts, date(t, jan10)); !errors.Is(err, tt.expected) {
				t.Errorf("NewInstallments(%d, %d) error = %v, want %v", tt.total, tt.parts, err, tt.expected)
			}
		})
	}
}

func TestSeries_ExpandInstallments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		series   domain.Series
		from, to string
		expected []part
	}{
		{
			name: "the last part takes the remainder", series: installments(t, 10000, 3, jan10),
			from: "2026-01-01", to: dec31,
			expected: []part{{jan10, 1, 3333}, {feb10, 2, 3333}, {mar10, 3, 3334}},
		},
		{
			name: "a purchase from august seen from a november D0", series: installments(t, 100000, 12, "2026-08-15"),
			from: nov01, to: "2027-12-31",
			expected: []part{
				{"2026-11-15", 4, 8333},
				{"2026-12-15", 5, 8333},
				{"2027-01-15", 6, 8333},
				{"2027-02-15", 7, 8333},
				{"2027-03-15", 8, 8333},
				{"2027-04-15", 9, 8333},
				{"2027-05-15", 10, 8333},
				{"2027-06-15", 11, 8333},
				{"2027-07-15", 12, 8337},
			},
		},
		{
			name: "parts anchored on the 31st", series: installments(t, 10000, 3, jan31),
			from: "2026-01-01", to: dec31,
			expected: []part{{jan31, 1, 3333}, {feb28, 2, 3333}, {mar31, 3, 3334}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parts(tt.series.Expand(date(t, tt.from), date(t, tt.to), nil))
			if !slices.Equal(got, tt.expected) {
				t.Errorf("Expand = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSeries_SplitAtInstallments(t *testing.T) {
	t.Parallel()

	from, to := date(t, "2026-01-01"), date(t, dec31)
	series := installments(t, 10000, 3, jan10)
	head, tail, err := series.SplitAt(date(t, feb10))
	if err != nil {
		t.Fatalf("SplitAt(%s) error = %v", feb10, err)
	}
	if got := parts(head.Expand(from, to, nil)); !slices.Equal(got, []part{{jan10, 1, 3333}}) {
		t.Errorf("head = %v, want [1/3]", got)
	}
	if got := parts(tail.Expand(from, to, nil)); !slices.Equal(got, []part{{feb10, 2, 3333}, {mar10, 3, 3334}}) {
		t.Errorf("tail = %v, want the numbering to continue at 2/3", got)
	}

	edited, err := tail.WithAmount(4000)
	if err != nil {
		t.Fatalf("WithAmount(4000) error = %v", err)
	}
	if got := parts(edited.Expand(from, to, nil)); !slices.Equal(got, []part{{feb10, 2, 4000}, {mar10, 3, 4000}}) {
		t.Errorf("edited tail = %v, want every part from there on at the edited amount", got)
	}

	if _, _, err := series.SplitAt(date(t, apr10)); !errors.Is(err, domain.ErrNotAnOccurrence) {
		t.Errorf("SplitAt(after the last part) error = %v, want %v", err, domain.ErrNotAnOccurrence)
	}
}

func TestSeries_WithStart(t *testing.T) {
	t.Parallel()

	rent := recurring(t, domain.FrequencyMonthly, 1000, "2026-01-05", "")
	_, tail, err := rent.SplitAt(date(t, "2026-12-05"))
	if err != nil {
		t.Fatalf("SplitAt error = %v", err)
	}
	moved, err := tail.WithStart(date(t, "2026-12-10"))
	if err != nil {
		t.Fatalf("WithStart error = %v", err)
	}
	got := dates(moved.Expand(date(t, "2026-12-01"), date(t, "2027-02-28"), nil))
	if want := []string{"2026-12-10", "2027-01-10", "2027-02-10"}; !slices.Equal(got, want) {
		t.Errorf("rent moved to the 10th from December = %v, want %v", got, want)
	}

	_, purchase, err := installments(t, 10000, 3, jan10).SplitAt(date(t, feb10))
	if err != nil {
		t.Fatalf("SplitAt error = %v", err)
	}
	movedParts, err := purchase.WithStart(date(t, "2026-02-20"))
	if err != nil {
		t.Fatalf("WithStart error = %v", err)
	}
	gotParts := parts(movedParts.Expand(date(t, "2026-01-01"), date(t, dec31), nil))
	if want := []part{{"2026-02-20", 2, 3333}, {"2026-03-20", 3, 3334}}; !slices.Equal(gotParts, want) {
		t.Errorf("installments moved to the 20th = %v, want the numbering kept: %v", gotParts, want)
	}

	toMonthEnd, err := recurring(t, domain.FrequencyMonthly, 1000, jan10, "").WithStart(date(t, jan31))
	if err != nil {
		t.Fatalf("WithStart error = %v", err)
	}
	if got := dates(toMonthEnd.Expand(date(t, "2026-01-01"), date(t, mar31), nil)); !slices.Equal(got, []string{jan31, feb28, mar31}) {
		t.Errorf("series moved to the 31st = %v, want [%s %s %s]", got, jan31, feb28, mar31)
	}

	ending := recurring(t, domain.FrequencyDaily, 1000, nov01, nov30)
	if _, err := ending.WithStart(date(t, "2026-12-01")); !errors.Is(err, domain.ErrEndBeforeStart) {
		t.Errorf("WithStart after the end error = %v, want %v", err, domain.ErrEndBeforeStart)
	}
	if _, err := ending.WithStart(domain.Date{}); !errors.Is(err, domain.ErrInvalidDate) {
		t.Errorf("WithStart(zero date) error = %v, want %v", err, domain.ErrInvalidDate)
	}
}

func TestSeries_ExpandInstallmentsWithOverride(t *testing.T) {
	t.Parallel()

	exceptions := map[domain.Date]domain.Exception{date(t, mar10): {Date: date(t, "2026-04-02"), Amount: 5000}}
	got := parts(installments(t, 10000, 3, jan10).Expand(date(t, "2026-01-01"), date(t, dec31), exceptions))
	if want := []part{{jan10, 1, 3333}, {feb10, 2, 3333}, {"2026-04-02", 3, 5000}}; !slices.Equal(got, want) {
		t.Errorf("Expand = %v, want the moved last part still numbered 3/3 with the override's amount: %v", got, want)
	}
}

func TestSeries_WithAmountKeepsOverrides(t *testing.T) {
	t.Parallel()

	exceptions := map[domain.Date]domain.Exception{date(t, feb10): {Date: date(t, "2026-02-12"), Amount: 120000}}
	edited, err := recurring(t, domain.FrequencyMonthly, 100000, jan10, "").WithAmount(5000)
	if err != nil {
		t.Fatalf("WithAmount error = %v", err)
	}
	got := edited.Expand(date(t, "2026-02-01"), date(t, apr30), exceptions)
	want := []domain.Occurrence{
		{Date: date(t, "2026-02-12"), Original: date(t, feb10), Amount: 120000},
		{Date: date(t, mar10), Original: date(t, mar10), Amount: 5000},
		{Date: date(t, apr10), Original: date(t, apr10), Amount: 5000},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Expand after an all edit = %+v, want the individually edited occurrence kept: %+v", got, want)
	}
}

func TestSeries_Validate(t *testing.T) {
	t.Parallel()

	monthly := recurring(t, domain.FrequencyMonthly, 1000, jan31, "")
	daily := recurring(t, domain.FrequencyDaily, 1000, nov01, "")
	_, dailyTail, err := daily.SplitAt(date(t, "2026-11-05"))
	if err != nil {
		t.Fatalf("SplitAt error = %v", err)
	}
	purchase := installments(t, 10000, 3, jan10)
	_, purchaseTail, err := purchase.SplitAt(date(t, feb10))
	if err != nil {
		t.Fatalf("SplitAt error = %v", err)
	}
	for name, s := range map[string]domain.Series{
		"monthly": monthly, "daily": daily, "a daily tail keeps its first anchor day": dailyTail,
		"installments": purchase, "an installments tail": purchaseTail,
	} {
		if err := s.Validate(); err != nil {
			t.Errorf("%s: Validate() error = %v, want nil", name, err)
		}
	}

	tests := []struct {
		name     string
		change   func(s *domain.Series)
		base     domain.Series
		expected error
	}{
		{name: "unknown type", base: monthly, change: func(s *domain.Series) { s.Type = "transfer" }, expected: domain.ErrUnknownMovementType},
		{name: "no frequency", base: monthly, change: func(s *domain.Series) { s.Frequency = "" }, expected: domain.ErrUnknownFrequency},
		{name: "no start", base: monthly, change: func(s *domain.Series) { s.Start = domain.Date{} }, expected: domain.ErrInvalidDate},
		{name: "zero amount", base: monthly, change: func(s *domain.Series) { s.Amount = 0 }, expected: domain.ErrAmountNotPositive},
		{name: "end before start", base: monthly, change: func(s *domain.Series) { s.End = date(t, "2026-01-30") }, expected: domain.ErrEndBeforeStart},
		{name: "anchor day 0", base: daily, change: func(s *domain.Series) { s.AnchorDay = 0 }, expected: domain.ErrInvalidSeries},
		{name: "anchor day 32", base: daily, change: func(s *domain.Series) { s.AnchorDay = 32 }, expected: domain.ErrInvalidSeries},
		{name: "monthly start off its anchor day", base: monthly, change: func(s *domain.Series) { s.AnchorDay = 15 }, expected: domain.ErrInvalidSeries},
		{name: "monthly with installment fields", base: monthly, change: func(s *domain.Series) { s.Parts = 3 }, expected: domain.ErrInvalidSeries},
		{name: "one part", base: purchase, change: func(s *domain.Series) { s.Parts = 1 }, expected: domain.ErrInstallmentsOutOfRange},
		{name: "first part 0", base: purchase, change: func(s *domain.Series) { s.FirstPart = 0 }, expected: domain.ErrInvalidSeries},
		{name: "first part past the last", base: purchase, change: func(s *domain.Series) { s.FirstPart = 4 }, expected: domain.ErrInvalidSeries},
		{name: "zero last part", base: purchase, change: func(s *domain.Series) { s.LastAmount = 0 }, expected: domain.ErrAmountNotPositive},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := tt.base
			tt.change(&s)
			if err := s.Validate(); !errors.Is(err, tt.expected) {
				t.Errorf("Validate() error = %v, want %v", err, tt.expected)
			}
		})
	}
}
