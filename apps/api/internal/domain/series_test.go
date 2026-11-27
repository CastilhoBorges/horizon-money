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
			from: nov01, to: "2026-12-31", expected: []string{"2026-11-28", "2026-11-29", nov30},
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
	if got := dates(head.Expand(date(t, "2026-01-01"), date(t, "2026-12-31"), nil)); !slices.Equal(got, []string{jan31}) {
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
