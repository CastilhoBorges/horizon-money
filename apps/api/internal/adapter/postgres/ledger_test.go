//go:build integration

package postgres_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

// sortedIDs is ids in the order the repositories return tag ids: ascending.
func sortedIDs(ids ...domain.ID) []domain.ID {
	slices.SortFunc(ids, func(a, b domain.ID) int { return slices.Compare(a[:], b[:]) })
	return ids
}

func TestMovements_ForLedgerPicksWhatTheRangeNeeds(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main") // D0 = 2026-11-01
	ctx := t.Context()

	inRange := create(t, repos, userID, oneOff(t, w, nov03, domain.TypeDaily, 100))
	create(t, repos, userID, oneOff(t, w, "2027-01-03", domain.TypeDaily, 100)) // after the range
	rent := create(t, repos, userID, monthlySeries(t, w, "2020-01-05", 120000))

	endedEarly := monthlySeries(t, w, "2026-01-10", 5000)
	endedEarly.Series.End = mustDate(t, "2026-09-30")
	create(t, repos, userID, endedEarly) // ended before D0
	movedIn := monthlySeries(t, w, "2026-01-20", 7000)
	movedIn.Series.End = mustDate(t, "2026-10-31")
	movedIn = create(t, repos, userID, movedIn)
	moved := domain.OccurrenceEdit{Original: mustDate(t, "2026-10-20"), Exception: domain.Exception{Date: mustDate(t, "2026-11-02"), Amount: 7000}}
	if err := repos.Movements.PutEdit(ctx, userID, movedIn.ID, moved); err != nil {
		t.Fatalf("PutEdit error = %v", err)
	}
	create(t, repos, userID, monthlySeries(t, w, "2027-02-01", 100)) // starts after the range

	movements, edits, err := repos.Movements.ForLedger(ctx, userID, w.ID, mustDate(t, "2026-12-31"))
	if err != nil {
		t.Fatalf("ForLedger error = %v", err)
	}
	if want := []domain.Movement{rent, movedIn, inRange}; !reflect.DeepEqual(movements, want) {
		t.Errorf("ForLedger movements =\n %+v\nwant (by date) the rent, the series with an occurrence moved into the range, the in-range one-off:\n %+v", movements, want)
	}
	if !reflect.DeepEqual(edits, map[domain.ID][]domain.OccurrenceEdit{movedIn.ID: {moved}}) {
		t.Errorf("ForLedger edits = %+v, want only the moved occurrence", edits)
	}
}

func TestMovements_ForLedgerAlwaysReadsFromD0(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main") // D0 = 2026-11-01, initial 250.00
	november := create(t, repos, userID, oneOff(t, w, nov03, domain.TypeExpense, 10000))

	// A December balance still needs November's movements: the balance builds up from D0.
	movements, _, err := repos.Movements.ForLedger(t.Context(), userID, w.ID, mustDate(t, "2026-12-31"))
	if err != nil || !reflect.DeepEqual(movements, []domain.Movement{november}) {
		t.Fatalf("ForLedger = %+v, %v; want the November one-off", movements, err)
	}
	ledger, err := domain.NewLedger(w, nil, movements, nil, nil, mustDate(t, "2026-12-31"))
	if err != nil {
		t.Fatalf("NewLedger error = %v", err)
	}
	dec1 := mustDate(t, "2026-12-01")
	if days := ledger.Days(dec1, dec1, mustDate(t, "2026-11-27")); len(days) != 1 || days[0].End != 15000 {
		t.Errorf("Dec 1 = %+v, want an end balance of 150.00 (250.00 - 100.00)", days)
	}
}

func TestLedger_TheAcceptanceFixtureStoredAndReadBack(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main") // BRL, D0 = 2026-11-01, initial 250.00
	ctx := t.Context()

	categories := []domain.ForecastCategory{{Name: "Food", MonthlyAmount: 35000}, {Name: "Transport", MonthlyAmount: 30000}, {Name: "Leisure", MonthlyAmount: 60000}}
	if err := repos.Workspaces.ReplaceForecastCategories(ctx, userID, w.ID, categories); err != nil {
		t.Fatalf("ReplaceForecastCategories error = %v", err)
	}
	for _, e := range []struct {
		day    string
		kind   domain.MovementType
		amount domain.Amount
	}{
		{"2026-11-01", domain.TypeDaily, 2340},
		{"2026-11-02", domain.TypeDaily, 890},
		{nov03, domain.TypeIncome, 60000},
		{nov03, domain.TypeExpense, 1850},
		{nov03, domain.TypeExpense, 4275},
		{nov03, domain.TypeExpense, 9630},
		{"2026-11-04", domain.TypeDaily, 3520},
		{"2026-11-05", domain.TypeDaily, 5210},
		{nov06, domain.TypeIncome, 640000},
		{nov06, domain.TypeSaving, 190000},
		{nov06, domain.TypeCard, 213580},
		{nov06, domain.TypeExpense, 145000},
		{"2026-11-07", domain.TypeDaily, 6460},
	} {
		create(t, repos, userID, oneOff(t, w, e.day, e.kind, e.amount))
	}

	to := mustDate(t, "2026-12-02")
	stored, err := repos.Workspaces.ForecastCategories(ctx, userID, w.ID)
	if err != nil {
		t.Fatalf("ForecastCategories error = %v", err)
	}
	movements, edits, err := repos.Movements.ForLedger(ctx, userID, w.ID, to)
	if err != nil {
		t.Fatalf("ForLedger error = %v", err)
	}
	transfers, err := repos.Transfers.ListForWorkspace(ctx, userID, w.ID, w.Start, to)
	if err != nil {
		t.Fatalf("ListForWorkspace error = %v", err)
	}
	ledger, err := domain.NewLedger(w, stored, movements, edits, transfers, to)
	if err != nil {
		t.Fatalf("NewLedger error = %v", err)
	}

	days := ledger.Days(w.Start, to, mustDate(t, "2026-11-27"))
	expected := map[int]domain.Amount{1: 22660, 2: 21770, 3: 66015, 4: 62495, 5: 57285, 6: 148705, 28: 138078, 29: 133911, 30: 129744}
	for day := 7; day <= 27; day++ {
		expected[day] = 142245
	}
	if len(days) != 32 || days[30].End != 125577 || days[31].End != 121410 {
		t.Fatalf("Days = %d days ending %v, want 32 days ending 125577 and 121410 on Dec 1 and 2", len(days), days[len(days)-1].End)
	}
	for i, d := range days[:30] {
		if d.End != expected[i+1] {
			t.Errorf("end of %s = %d, want %d (§4.14)", d.Date, d.End, expected[i+1])
		}
	}
}
