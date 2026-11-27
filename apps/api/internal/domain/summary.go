package domain

import "time"

// BasisPoints is a rate in hundredths of a percent: 2714 is 27.14%.
type BasisPoints int64

// fullRate is 100% in basis points.
const fullRate = 10_000

// SavingsTarget is the user's ideal savings-rate band, bounds included.
type SavingsTarget struct {
	Min, Max BasisPoints
}

// PerformanceLabel says whether a period left money or fell short.
type PerformanceLabel string

// The performance labels (§4.10).
const (
	PerformanceMoneyLeft PerformanceLabel = "money_left"
	PerformanceShort     PerformanceLabel = "short"
)

// SavingsBand places the savings rate against the user's target. It is empty when there is no rate.
type SavingsBand string

// The savings bands (§4.10).
const (
	BandBelow  SavingsBand = "below"
	BandWithin SavingsBand = "within"
	BandAbove  SavingsBand = "above"
)

// CostLabel compares the cost of living with the income.
type CostLabel string

// The cost-of-living labels (§4.10).
const (
	CostWithinIncome CostLabel = "within_income"
	CostAboveIncome  CostLabel = "above_income"
)

// Calculations are the four monthly calculations and the sums behind them (§4.10). A year has the same
// shape, computed from the sums of its months (§4.11).
type Calculations struct {
	Totals            Totals
	TransfersIn       Amount
	TransfersOut      Amount
	ForecastDays      int
	ForecastRemaining Amount
	ElapsedDays       int
	Performance       Amount
	PerformanceLabel  PerformanceLabel
	// SavingsRate is nil when there is no income.
	SavingsRate  *BasisPoints
	SavingsBand  SavingsBand
	CostOfLiving Amount
	CostLabel    CostLabel
	// AverageDaily is nil when no day has elapsed, as in a future month.
	AverageDaily *Amount
}

// MonthSummary is the monthly view of one workspace. DailyForecast is there to compare with AverageDaily.
type MonthSummary struct {
	Month         Month
	DailyForecast Amount
	Calculations
}

// YearSummary is the yearly view: the 12 months and the year's totals.
type YearSummary struct {
	Year   int
	Months []MonthSummary
	Calculations
}

// MonthSummary computes the monthly calculations of m. Sums are over the entries and legs dated in m from
// D0 on, and future days of m carry the forecast.
func (l Ledger) MonthSummary(m Month, today Date, target SavingsTarget) MonthSummary {
	return l.summarize(m, l.monthSums(m, today), target)
}

// YearSummary computes the 12 months of year and the year's totals. Future months are projections that
// include the forecast.
func (l Ledger) YearSummary(year int, today Date, target SavingsTarget) (YearSummary, error) {
	january, err := NewMonth(year, time.January)
	if err != nil {
		return YearSummary{}, err
	}
	summary := YearSummary{Year: year, Months: make([]MonthSummary, 0, 12)}
	var total sums
	for i := range 12 {
		m := january.Add(i)
		s := l.monthSums(m, today)
		total.add(s)
		summary.Months = append(summary.Months, l.summarize(m, s, target))
	}
	summary.Calculations = total.derive(target)
	return summary, nil
}

func (l Ledger) summarize(m Month, s sums, target SavingsTarget) MonthSummary {
	return MonthSummary{Month: m, DailyForecast: l.DailyForecast, Calculations: s.derive(target)}
}

// sums are the raw figures of a period, which every calculation derives from.
type sums struct {
	totals            Totals
	in, out           Amount
	forecastDays      int
	forecastRemaining Amount
	elapsedDays       int
}

func (s *sums) add(o sums) {
	s.totals.sum(o.totals)
	s.in += o.in
	s.out += o.out
	s.forecastDays += o.forecastDays
	s.forecastRemaining += o.forecastRemaining
	s.elapsedDays += o.elapsedDays
}

func (l Ledger) monthSums(m Month, today Date) sums {
	var s sums
	for _, e := range l.Entries {
		if m.Contains(e.Date) && !e.Date.Before(l.Start) {
			s.totals.add(e.Type, e.Amount)
		}
	}
	for _, leg := range l.Transfers {
		if !m.Contains(leg.Date) || leg.Date.Before(l.Start) {
			continue
		}
		if leg.Incoming {
			s.in += leg.Amount
		} else {
			s.out += leg.Amount
		}
	}
	for d := m.First(); !d.After(m.Last()); d = d.AddDays(1) {
		if l.hasForecast(d, today) {
			s.forecastDays++
		}
	}
	s.forecastRemaining = l.DailyForecast * Amount(s.forecastDays)
	s.elapsedDays = l.elapsedDays(m, today)
	return s
}

// elapsedDays are the days average daily divides by (§4.10): from max(month start, D0) through today in
// the current month, the days on or after D0 in a past month, and none in a future month.
func (l Ledger) elapsedDays(m Month, today Date) int {
	if m.First().After(today) {
		return 0
	}
	first, last := m.First(), m.Last()
	if l.Start.After(first) {
		first = l.Start
	}
	if m.Contains(today) {
		last = today
	}
	if last.Before(first) {
		return 0
	}
	return last.DaysSince(first) + 1
}

func (s sums) derive(target SavingsTarget) Calculations {
	t := s.totals
	c := Calculations{
		Totals: t, TransfersIn: s.in, TransfersOut: s.out,
		ForecastDays: s.forecastDays, ForecastRemaining: s.forecastRemaining, ElapsedDays: s.elapsedDays,
		Performance:      t.Income - (t.Expense + t.Daily + t.Saving + t.Card + s.forecastRemaining),
		PerformanceLabel: PerformanceMoneyLeft,
		CostOfLiving:     t.Expense + t.Daily + t.Card + s.forecastRemaining,
		CostLabel:        CostWithinIncome,
	}
	if c.Performance < 0 {
		c.PerformanceLabel = PerformanceShort
	}
	if c.CostOfLiving > t.Income {
		c.CostLabel = CostAboveIncome
	}
	if t.Income > 0 {
		rate := BasisPoints(mulDivHalfUp(int64(t.Saving), fullRate, int64(t.Income)))
		c.SavingsRate, c.SavingsBand = &rate, target.band(rate)
	}
	if s.elapsedDays > 0 {
		average := Amount(divHalfUp(int64(t.Daily), int64(s.elapsedDays)))
		c.AverageDaily = &average
	}
	return c
}

// band compares the rate as shown (rounded), so the label always matches the number.
func (t SavingsTarget) band(rate BasisPoints) SavingsBand {
	switch {
	case rate < t.Min:
		return BandBelow
	case rate > t.Max:
		return BandAbove
	default:
		return BandWithin
	}
}
