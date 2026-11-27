package domain_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

// target is the default savings-rate band, 20% to 30%.
var target = domain.SavingsTarget{Min: 2000, Max: 3000}

func ptr[T any](v T) *T { return new(v) }

// show prints calculations with their optional values dereferenced.
func show(c domain.Calculations) string {
	rate, average := "nil", "nil"
	if c.SavingsRate != nil {
		rate = fmt.Sprint(*c.SavingsRate)
	}
	if c.AverageDaily != nil {
		average = fmt.Sprint(*c.AverageDaily)
	}
	c.SavingsRate, c.AverageDaily = nil, nil
	return fmt.Sprintf("%+v SavingsRate=%s AverageDaily=%s", c, rate, average)
}

func assertCalculations(t *testing.T, got, want domain.Calculations) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("calculations\n got: %s\nwant: %s", show(got), show(want))
	}
}

// fixtureNovember is the §4.14 expected November summary, seen from today 2026-11-27.
var fixtureNovember = domain.Calculations{
	Totals:            domain.Totals{Income: 700000, Expense: 160755, Daily: 18420, Saving: 190000, Card: 213580},
	ForecastDays:      3,
	ForecastRemaining: 12501,
	ElapsedDays:       27,
	Performance:       104744,
	PerformanceLabel:  domain.PerformanceMoneyLeft,
	SavingsRate:       ptr(domain.BasisPoints(2714)),
	SavingsBand:       domain.BandWithin,
	CostOfLiving:      405256,
	CostLabel:         domain.CostWithinIncome,
	AverageDaily:      ptr(domain.Amount(682)),
}

func TestLedger_MonthSummary_AcceptanceFixture(t *testing.T) {
	t.Parallel()

	got := acceptanceLedger(t).MonthSummary(month(t, "2026-11"), date(t, nov27), target)
	if got.Month != month(t, "2026-11") || got.DailyForecast != 4167 {
		t.Errorf("month %s with daily forecast %d, want 2026-11 with 4167", got.Month, got.DailyForecast)
	}
	assertCalculations(t, got.Calculations, fixtureNovember)
}

func TestLedger_MonthSummary_Periods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		month    string
		today    string
		expected domain.Calculations
	}{
		{name: "a future month is all forecast", month: "2026-12", today: nov27, expected: domain.Calculations{
			ForecastDays: 31, ForecastRemaining: 129177,
			Performance: -129177, PerformanceLabel: domain.PerformanceShort,
			CostOfLiving: 129177, CostLabel: domain.CostAboveIncome,
		}},
		{name: "a month before D0 is empty", month: "2026-10", today: nov27, expected: domain.Calculations{
			PerformanceLabel: domain.PerformanceMoneyLeft, CostLabel: domain.CostWithinIncome,
		}},
		{
			name: "a past month has no forecast and counts every day from D0", month: "2026-11", today: "2026-12-10",
			expected: domain.Calculations{
				Totals:      fixtureNovember.Totals,
				ElapsedDays: 30,
				Performance: 117245, PerformanceLabel: domain.PerformanceMoneyLeft,
				SavingsRate: ptr(domain.BasisPoints(2714)), SavingsBand: domain.BandWithin,
				CostOfLiving: 392755, CostLabel: domain.CostWithinIncome,
				AverageDaily: ptr(domain.Amount(614)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := acceptanceLedger(t).MonthSummary(month(t, tt.month), date(t, tt.today), target)
			assertCalculations(t, got.Calculations, tt.expected)
		})
	}
}

func TestLedger_MonthSummary_D0MidMonth(t *testing.T) {
	t.Parallel()

	ledger := domain.Ledger{
		Start:   date(t, "2026-11-10"),
		Entries: []domain.Entry{entry(t, "2026-11-10", domain.TypeDaily, 10000)},
	}
	got := ledger.MonthSummary(month(t, "2026-11"), date(t, nov27), target)
	if got.ElapsedDays != 18 || got.AverageDaily == nil || *got.AverageDaily != 556 {
		t.Errorf("elapsed %d days, average %s; want 18 days from D0 through today and 556", got.ElapsedDays, show(got.Calculations))
	}
}

func TestLedger_MonthSummary_SavingsBand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		income       domain.Amount
		saving       domain.Amount
		expectedRate *domain.BasisPoints
		expectedBand domain.SavingsBand
	}{
		{name: "below the band", income: 10000, saving: 1999, expectedRate: ptr(domain.BasisPoints(1999)), expectedBand: domain.BandBelow},
		{name: "on the lower bound", income: 10000, saving: 2000, expectedRate: ptr(domain.BasisPoints(2000)), expectedBand: domain.BandWithin},
		{name: "on the upper bound", income: 10000, saving: 3000, expectedRate: ptr(domain.BasisPoints(3000)), expectedBand: domain.BandWithin},
		{name: "above the band", income: 10000, saving: 3001, expectedRate: ptr(domain.BasisPoints(3001)), expectedBand: domain.BandAbove},
		{name: "rounded down onto the bound", income: 100000, saving: 30004, expectedRate: ptr(domain.BasisPoints(3000)), expectedBand: domain.BandWithin},
		{name: "a half rounds up past the bound", income: 100000, saving: 30005, expectedRate: ptr(domain.BasisPoints(3001)), expectedBand: domain.BandAbove},
		{name: "no income means no rate", income: 0, saving: 5000, expectedRate: nil, expectedBand: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := []domain.Entry{entry(t, nov01, domain.TypeSaving, tt.saving)}
			if tt.income > 0 {
				entries = append(entries, entry(t, nov01, domain.TypeIncome, tt.income))
			}
			ledger := domain.Ledger{Start: date(t, nov01), Entries: entries}
			got := ledger.MonthSummary(month(t, "2026-11"), date(t, nov01), target)
			if !reflect.DeepEqual(got.SavingsRate, tt.expectedRate) || got.SavingsBand != tt.expectedBand {
				t.Errorf("savings %s, want rate %v band %q", show(got.Calculations), tt.expectedRate, tt.expectedBand)
			}
		})
	}
}

func TestLedger_MonthSummary_LargeSums(t *testing.T) {
	t.Parallel()

	var entries []domain.Entry
	for range 1000 {
		entries = append(entries,
			entry(t, nov01, domain.TypeIncome, domain.MaxAmount),
			entry(t, nov01, domain.TypeSaving, 4999999999999))
	}
	ledger := domain.Ledger{Start: date(t, nov01), Entries: entries}
	got := ledger.MonthSummary(month(t, "2026-11"), date(t, nov01), target)
	if got.SavingsRate == nil || *got.SavingsRate != 5000 {
		t.Errorf("savings rate %s, want 5000 without overflowing", show(got.Calculations))
	}
}

func TestLedger_MonthSummary_BalanceIdentity(t *testing.T) {
	t.Parallel()

	ledger := acceptanceLedger(t)
	rent, err := domain.NewRecurring(domain.TypeExpense, domain.FrequencyMonthly, 120000, date(t, "2020-01-05"), domain.Date{})
	if err != nil {
		t.Fatalf("NewRecurring error = %v", err)
	}
	for _, o := range rent.Expand(date(t, nov01), date(t, "2027-02-28"), nil) {
		ledger.Entries = append(ledger.Entries, domain.Entry{Date: o.Date, Type: domain.TypeExpense, Amount: o.Amount})
	}
	ledger.Transfers = []domain.TransferLeg{
		{Date: date(t, "2026-11-10"), Amount: 50000, Incoming: true},
		{Date: date(t, "2026-12-03"), Amount: 20000, Incoming: false},
	}
	today := date(t, nov27)
	ends := byDate(ledger.Days(date(t, nov01), date(t, "2027-02-28"), today))
	endOf := func(d domain.Date) domain.Amount {
		if d.Before(ledger.Start) {
			return ledger.InitialBalance
		}
		return ends[d.String()].End
	}

	for m := month(t, "2026-10"); m != month(t, "2027-03"); m = m.Add(1) {
		summary := ledger.MonthSummary(m, today, target)
		change := endOf(m.Last()) - endOf(m.Add(-1).Last())
		if want := summary.Performance + summary.TransfersIn - summary.TransfersOut; change != want {
			t.Errorf("%s: balance changed by %d, but performance + transfers in - out = %d", m, change, want)
		}
	}
}

func TestLedger_YearSummary_AcceptanceFixture(t *testing.T) {
	t.Parallel()

	ledger := acceptanceLedger(t)
	today := date(t, nov27)
	got, err := ledger.YearSummary(2026, today, target)
	if err != nil {
		t.Fatalf("YearSummary error = %v", err)
	}
	if got.Year != 2026 || len(got.Months) != 12 {
		t.Fatalf("year %d with %d months, want 2026 with 12", got.Year, len(got.Months))
	}
	if november := ledger.MonthSummary(month(t, "2026-11"), today, target); !reflect.DeepEqual(got.Months[10], november) {
		t.Errorf("Months[10] = %s, want the November summary %s", show(got.Months[10].Calculations), show(november.Calculations))
	}
	assertCalculations(t, got.Calculations, domain.Calculations{
		Totals:            fixtureNovember.Totals,
		ForecastDays:      34,
		ForecastRemaining: 141678,
		ElapsedDays:       27,
		Performance:       -24433,
		PerformanceLabel:  domain.PerformanceShort,
		SavingsRate:       ptr(domain.BasisPoints(2714)),
		SavingsBand:       domain.BandWithin,
		CostOfLiving:      534433,
		CostLabel:         domain.CostWithinIncome,
		AverageDaily:      ptr(domain.Amount(682)),
	})

	for _, year := range []int{0, 10000} {
		if _, err := ledger.YearSummary(year, today, target); !errors.Is(err, domain.ErrInvalidMonth) {
			t.Errorf("YearSummary(%d) error = %v, want %v", year, err, domain.ErrInvalidMonth)
		}
	}
}
