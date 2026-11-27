package domain_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func entry(t *testing.T, day string, kind domain.MovementType, amount domain.Amount) domain.Entry {
	t.Helper()
	return domain.Entry{Date: date(t, day), Type: kind, Amount: amount}
}

// acceptanceLedger is the §4.14 fixture: a BRL workspace with D0 = 2026-11-01, an initial balance of
// 250.00, forecast categories of 350 + 300 + 600 with divisor 30, and the first week of November logged.
func acceptanceLedger(t *testing.T) domain.Ledger {
	t.Helper()
	daily, err := domain.DailyValue([]domain.Amount{35000, 30000, 60000}, 30)
	if err != nil {
		t.Fatalf("DailyValue error = %v", err)
	}
	return domain.Ledger{
		Start:          date(t, nov01),
		InitialBalance: 25000,
		DailyForecast:  daily,
		Entries: []domain.Entry{
			entry(t, nov01, domain.TypeDaily, 2340),
			entry(t, nov02, domain.TypeDaily, 890),
			entry(t, "2026-11-03", domain.TypeIncome, 60000),
			entry(t, "2026-11-03", domain.TypeExpense, 1850),
			entry(t, "2026-11-03", domain.TypeExpense, 4275),
			entry(t, "2026-11-03", domain.TypeExpense, 9630),
			entry(t, "2026-11-04", domain.TypeDaily, 3520),
			entry(t, "2026-11-05", domain.TypeDaily, 5210),
			entry(t, "2026-11-06", domain.TypeIncome, 640000),
			entry(t, "2026-11-06", domain.TypeSaving, 190000),
			entry(t, "2026-11-06", domain.TypeCard, 213580),
			entry(t, "2026-11-06", domain.TypeExpense, 145000),
			entry(t, "2026-11-07", domain.TypeDaily, 6460),
		},
	}
}

// byDate indexes days by their "YYYY-MM-DD" date.
func byDate(days []domain.Day) map[string]domain.Day {
	out := make(map[string]domain.Day, len(days))
	for _, d := range days {
		out[d.Date.String()] = d
	}
	return out
}

func TestLedger_Days_AcceptanceFixture(t *testing.T) {
	t.Parallel()

	days := acceptanceLedger(t).Days(date(t, nov01), date(t, "2026-12-02"), date(t, nov27))
	if len(days) != 32 {
		t.Fatalf("Days returned %d days, want 32", len(days))
	}

	expected := map[int]domain.Amount{1: 22660, 2: 21770, 3: 66015, 4: 62495, 5: 57285, 6: 148705}
	for day := 7; day <= 27; day++ {
		expected[day] = 142245
	}
	expected[28], expected[29], expected[30] = 138078, 133911, 129744
	for i, d := range days[:30] {
		if d.End != expected[i+1] {
			t.Errorf("end of %s = %d, want %d", d.Date, d.End, expected[i+1])
		}
	}
	if days[30].End != 125577 || days[31].End != 121410 {
		t.Errorf("end of Dec 1 and 2 = %d and %d, want 125577 and 121410", days[30].End, days[31].End)
	}

	got := byDate(days)
	if got[nov27].Forecast != 0 || got[nov28].Forecast != 4167 {
		t.Errorf("forecast on Nov 27 and 28 = %d and %d, want 0 (today) and 4167", got[nov27].Forecast, got[nov28].Forecast)
	}
	if want := (domain.Totals{Income: 60000, Expense: 15755}); got["2026-11-03"].Totals != want {
		t.Errorf("totals on Nov 3 = %+v, want %+v", got["2026-11-03"].Totals, want)
	}
	if want := (domain.Totals{Income: 640000, Expense: 145000, Saving: 190000, Card: 213580}); got["2026-11-06"].Totals != want {
		t.Errorf("totals on Nov 6 = %+v, want %+v", got["2026-11-06"].Totals, want)
	}
}

func TestLedger_Days_FromAfterD0(t *testing.T) {
	t.Parallel()

	days := acceptanceLedger(t).Days(date(t, "2026-12-01"), date(t, "2026-12-02"), date(t, nov27))
	if len(days) != 2 || days[0].End != 125577 || days[1].End != 121410 {
		t.Errorf("Days(Dec 1..2) = %+v, want ends 125577 and 121410, built up from D0", days)
	}
}

func TestLedger_Days_ForecastCutOff(t *testing.T) {
	t.Parallel()

	saoPaulo, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("LoadLocation error = %v", err)
	}
	ledger := domain.Ledger{Start: date(t, nov01), InitialBalance: 10000, DailyForecast: 1000}
	tests := []struct {
		name       string
		now        time.Time
		noForecast string
		forecast   string
	}{
		{
			name: "one second before midnight in sao paulo", now: time.Date(2026, 11, 28, 2, 59, 59, 0, time.UTC),
			noForecast: nov27, forecast: nov28,
		},
		{
			name: "midnight in sao paulo", now: time.Date(2026, 11, 28, 3, 0, 0, 0, time.UTC),
			noForecast: nov28, forecast: "2026-11-29",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			today := domain.Today(tt.now, saoPaulo)
			days := byDate(ledger.Days(date(t, "2026-11-26"), date(t, "2026-11-29"), today))
			if days[tt.noForecast].Forecast != 0 || days[tt.forecast].Forecast != 1000 {
				t.Errorf("forecast on %s and %s = %d and %d, want 0 and 1000",
					tt.noForecast, tt.forecast, days[tt.noForecast].Forecast, days[tt.forecast].Forecast)
			}
		})
	}
}

func TestLedger_Days_StartsAtD0(t *testing.T) {
	t.Parallel()

	ledger := domain.Ledger{
		Start: date(t, "2026-12-05"), InitialBalance: 10000, DailyForecast: 1000,
		Entries: []domain.Entry{entry(t, "2026-12-04", domain.TypeExpense, 5000)},
	}
	days := ledger.Days(date(t, "2026-12-01"), date(t, "2026-12-06"), date(t, nov27))
	if len(days) != 2 || days[0].Date != date(t, "2026-12-05") || days[1].Date != date(t, "2026-12-06") {
		t.Fatalf("Days = %+v, want exactly Dec 5 and Dec 6", days)
	}
	if days[0].End != 9000 || days[1].End != 8000 {
		t.Errorf("ends = %d and %d, want 9000 and 8000 (forecast only, the entry before D0 ignored)", days[0].End, days[1].End)
	}
}

func TestLedger_Days_Status(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		balance   domain.Amount
		threshold domain.Amount
		expected  domain.Status
	}{
		{name: "below zero", balance: -1, threshold: 10000, expected: domain.StatusNegative},
		{name: "zero", balance: 0, threshold: 10000, expected: domain.StatusLow},
		{name: "just below the threshold", balance: 9999, threshold: 10000, expected: domain.StatusLow},
		{name: "at the threshold", balance: 10000, threshold: 10000, expected: domain.StatusOK},
		{name: "zero with the default threshold", balance: 0, threshold: 0, expected: domain.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			today := date(t, nov27)
			ledger := domain.Ledger{Start: today, InitialBalance: tt.balance, LowThreshold: tt.threshold}
			days := ledger.Days(today, today, today)
			if len(days) != 1 || days[0].Status != tt.expected {
				t.Errorf("Days = %+v, want one day with status %s", days, tt.expected)
			}
		})
	}
}

func TestLedger_Days_TransfersAndWeekend(t *testing.T) {
	t.Parallel()

	ledger := domain.Ledger{
		Start: date(t, nov27), InitialBalance: 10000,
		Transfers: []domain.TransferLeg{
			{Date: date(t, nov28), Amount: 5000, Incoming: true},
			{Date: date(t, nov28), Amount: 2000, Incoming: false},
		},
	}
	days := byDate(ledger.Days(date(t, nov27), date(t, nov28), date(t, nov27)))
	friday, saturday := days[nov27], days[nov28]
	if saturday.TransfersIn != 5000 || saturday.TransfersOut != 2000 || saturday.End != 13000 {
		t.Errorf("Nov 28 = in %d, out %d, end %d; want 5000, 2000, 13000",
			saturday.TransfersIn, saturday.TransfersOut, saturday.End)
	}
	if friday.IsWeekend || !saturday.IsWeekend {
		t.Errorf("IsWeekend = %v (Fri) and %v (Sat), want false and true", friday.IsWeekend, saturday.IsWeekend)
	}
}

func TestLedger_Horizon(t *testing.T) {
	t.Parallel()

	ledger := acceptanceLedger(t)
	days, err := ledger.Horizon(month(t, "2026-11"), domain.DefaultHorizonMonths, date(t, nov27))
	if err != nil {
		t.Fatalf("Horizon error = %v", err)
	}
	if len(days) != 92 || days[0].Date != date(t, nov01) || days[91].Date != date(t, "2027-01-31") {
		t.Errorf("Horizon = %d days from %s, want 92 days from 2026-11-01 to 2027-01-31", len(days), days[0].Date)
	}
	for _, months := range []int{0, domain.MaxHorizonMonths + 1} {
		if _, err := ledger.Horizon(month(t, "2026-11"), months, date(t, nov27)); !errors.Is(err, domain.ErrHorizonOutOfRange) {
			t.Errorf("Horizon(%d months) error = %v, want %v", months, err, domain.ErrHorizonOutOfRange)
		}
	}
}

func mustID(t *testing.T, s string) domain.ID {
	t.Helper()
	id, err := domain.ParseID(s)
	if err != nil {
		t.Fatalf("ParseID(%q) error = %v", s, err)
	}
	return id
}

func TestNewLedger_AcceptanceFixtureFromRecords(t *testing.T) {
	t.Parallel()

	workspace := domain.Workspace{
		ID: mustID(t, "01a21ec4-e9fa-7aa5-8edc-d14ef635e779"), Start: date(t, nov01), InitialBalance: 25000, ForecastDivisor: 30,
	}
	categories := []domain.ForecastCategory{{MonthlyAmount: 35000}, {MonthlyAmount: 30000}, {MonthlyAmount: 60000}}
	fixture := acceptanceLedger(t)
	movements := make([]domain.Movement, 0, len(fixture.Entries))
	for _, e := range fixture.Entries {
		movements = append(movements, domain.Movement{WorkspaceID: workspace.ID, Entry: e})
	}

	ledger, err := domain.NewLedger(workspace, categories, movements, nil, nil, date(t, "2026-12-02"))
	if err != nil {
		t.Fatalf("NewLedger error = %v", err)
	}
	from, to, today := date(t, nov01), date(t, "2026-12-02"), date(t, nov27)
	if got, want := ledger.Days(from, to, today), fixture.Days(from, to, today); !reflect.DeepEqual(got, want) {
		t.Errorf("Days from the stored records differ from the §4.14 fixture:\n got %+v\nwant %+v", got, want)
	}
}

func TestNewLedger_SeriesEditsAndTransfers(t *testing.T) {
	t.Parallel()

	workspace := domain.Workspace{
		ID: mustID(t, "01a21ec4-e9fa-7aa5-8edc-d14ef635e779"), Start: date(t, nov01), InitialBalance: 100000,
		LowThreshold: 5000, ForecastDivisor: 30,
	}
	other := mustID(t, "01a21ec4-e9fa-7aa5-8edc-d14ef635e780")
	rentID := mustID(t, "01a21ec4-e9fa-7aa5-8edc-d14ef635e781")
	rent, err := domain.NewRecurring(domain.TypeExpense, domain.FrequencyMonthly, 120000, date(t, "2020-01-05"), domain.Date{})
	if err != nil {
		t.Fatalf("NewRecurring error = %v", err)
	}
	movements := []domain.Movement{
		{ID: rentID, WorkspaceID: workspace.ID, Series: &rent, GroupID: rentID},
		{WorkspaceID: workspace.ID, Entry: entry(t, "2026-10-31", domain.TypeIncome, 999)}, // before D0
		{WorkspaceID: other, Entry: entry(t, nov02, domain.TypeIncome, 999)},               // another workspace
	}
	edits := map[domain.ID][]domain.OccurrenceEdit{rentID: {
		{Original: date(t, "2026-11-05"), Exception: domain.Exception{Date: date(t, "2026-11-07"), Amount: 130000}},
		{Original: date(t, "2026-12-05"), Exception: domain.Exception{Deleted: true}},
	}}
	transfers := []domain.Transfer{
		{FromWorkspaceID: workspace.ID, ToWorkspaceID: other, AmountSent: 20000, AmountReceived: 4000, Date: date(t, "2026-11-10")},
		{FromWorkspaceID: other, ToWorkspaceID: workspace.ID, AmountSent: 1000, AmountReceived: 5000, Date: date(t, "2026-11-12")},
	}

	got, err := domain.NewLedger(workspace, nil, movements, edits, transfers, date(t, dec31))
	if err != nil {
		t.Fatalf("NewLedger error = %v", err)
	}
	want := domain.Ledger{
		Start: date(t, nov01), InitialBalance: 100000, DailyForecast: 0, LowThreshold: 5000,
		Entries: []domain.Entry{entry(t, "2026-11-07", domain.TypeExpense, 130000)},
		Transfers: []domain.TransferLeg{
			{Date: date(t, "2026-11-10"), Amount: 20000, Incoming: false},
			{Date: date(t, "2026-11-12"), Amount: 5000, Incoming: true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewLedger =\n %+v\nwant\n %+v", got, want)
	}
}

func TestNewLedgerRejectsBadRecords(t *testing.T) {
	t.Parallel()

	workspace := domain.Workspace{ID: mustID(t, "01a21ec4-e9fa-7aa5-8edc-d14ef635e779"), Start: date(t, nov01), ForecastDivisor: 30}
	corrupt := domain.Series{Type: domain.TypeExpense, Frequency: domain.FrequencyMonthly, Start: date(t, nov01), Amount: 100}
	tests := []struct {
		name      string
		workspace domain.Workspace
		movements []domain.Movement
		expected  error
	}{
		{name: "divisor out of range", workspace: domain.Workspace{ID: workspace.ID, Start: workspace.Start, ForecastDivisor: 27}, expected: domain.ErrForecastDivisorOutOfRange},
		{name: "a corrupt series", workspace: workspace, movements: []domain.Movement{{WorkspaceID: workspace.ID, Series: &corrupt}}, expected: domain.ErrInvalidSeries},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewLedger(tt.workspace, nil, tt.movements, nil, nil, date(t, dec31)); !errors.Is(err, tt.expected) {
				t.Errorf("NewLedger error = %v, want %v", err, tt.expected)
			}
		})
	}
}
