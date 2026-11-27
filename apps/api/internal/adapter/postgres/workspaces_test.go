//go:build integration

package postgres_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

func mustDate(t *testing.T, s string) domain.Date {
	t.Helper()
	d, err := domain.ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", s, err)
	}
	return d
}

func brl(t *testing.T) domain.Currency {
	t.Helper()
	c, err := domain.ParseCurrency("BRL")
	if err != nil {
		t.Fatalf("ParseCurrency error = %v", err)
	}
	return c
}

// newWorkspace creates a BRL workspace with D0 = 2026-11-01 and an initial balance of 250.00.
func newWorkspace(t *testing.T, repos port.Repositories, userID domain.ID, name string) domain.Workspace {
	t.Helper()
	w, err := repos.Workspaces.Create(t.Context(), userID, port.NewWorkspace{
		Name: name, Currency: brl(t), InitialBalance: 25000, Start: mustDate(t, "2026-11-01"),
	})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	return w
}

func TestWorkspaces_CreateAppliesTheSeededDefaults(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	created := newWorkspace(t, repos, userID, "Main")

	got, err := repos.Workspaces.Get(t.Context(), userID, created.ID)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	want := domain.Workspace{
		ID: created.ID, Name: "Main", Currency: brl(t), InitialBalance: 25000, Start: mustDate(t, "2026-11-01"),
		LowThreshold: 0, ForecastDivisor: 30,
	}
	if created.ID.IsZero() || got != want || created != want {
		t.Errorf("Create = %+v and Get = %+v, want %+v", created, got, want)
	}
}

func TestWorkspaces_UpdateAndArchive(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	main := newWorkspace(t, repos, userID, "Main")
	travel := newWorkspace(t, repos, userID, "Travel")

	main.Name, main.LowThreshold, main.ForecastDivisor = "Everyday", 10000, 31
	main.InitialBalance, main.Start = 30000, mustDate(t, "2026-10-01")
	if err := repos.Workspaces.Update(t.Context(), userID, main); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if got, err := repos.Workspaces.Get(t.Context(), userID, main.ID); err != nil || got != main {
		t.Errorf("Get after Update = %+v, %v; want %+v", got, err, main)
	}

	if err := repos.Workspaces.SetArchived(t.Context(), userID, travel.ID, true); err != nil {
		t.Fatalf("SetArchived error = %v", err)
	}
	active, err := repos.Workspaces.List(t.Context(), userID, false)
	if err != nil || !slices.Equal(names(active), []string{"Everyday"}) {
		t.Errorf("List(active) = %v, %v; want [Everyday]", names(active), err)
	}
	all, err := repos.Workspaces.List(t.Context(), userID, true)
	if err != nil || !slices.Equal(names(all), []string{"Everyday", "Travel"}) || !all[1].Archived {
		t.Errorf("List(all) = %+v, %v; want Everyday then an archived Travel", all, err)
	}
}

func names(workspaces []domain.Workspace) []string {
	out := make([]string, 0, len(workspaces))
	for _, w := range workspaces {
		out = append(out, w.Name)
	}
	return out
}

func TestWorkspaces_ForecastCategoriesAreReplacedAsAWhole(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")

	first := []domain.ForecastCategory{{Name: "Food", MonthlyAmount: 35000}, {Name: "Transport", MonthlyAmount: 30000}, {Name: "Leisure", MonthlyAmount: 60000}}
	if err := repos.Workspaces.ReplaceForecastCategories(t.Context(), userID, w.ID, first); err != nil {
		t.Fatalf("ReplaceForecastCategories error = %v", err)
	}
	if got, err := repos.Workspaces.ForecastCategories(t.Context(), userID, w.ID); err != nil || !slices.Equal(amounts(got), []domain.Amount{35000, 30000, 60000}) {
		t.Errorf("ForecastCategories = %+v, %v; want 350, 300, 600 in order", got, err)
	}

	if err := repos.Workspaces.ReplaceForecastCategories(t.Context(), userID, w.ID, []domain.ForecastCategory{{Name: "All", MonthlyAmount: 100000}}); err != nil {
		t.Fatalf("ReplaceForecastCategories error = %v", err)
	}
	if got, err := repos.Workspaces.ForecastCategories(t.Context(), userID, w.ID); err != nil || len(got) != 1 || got[0].Name != "All" || got[0].ID.IsZero() {
		t.Errorf("ForecastCategories = %+v, %v; want only All", got, err)
	}
}

func amounts(categories []domain.ForecastCategory) []domain.Amount {
	out := make([]domain.Amount, 0, len(categories))
	for _, c := range categories {
		out = append(out, c.MonthlyAmount)
	}
	return out
}

func TestWorkspaces_DeleteTakesItsCategories(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	if err := repos.Workspaces.ReplaceForecastCategories(t.Context(), userID, w.ID, []domain.ForecastCategory{{Name: "Food", MonthlyAmount: 1000}}); err != nil {
		t.Fatalf("ReplaceForecastCategories error = %v", err)
	}

	if err := repos.Workspaces.Delete(t.Context(), userID, w.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := repos.Workspaces.Get(t.Context(), userID, w.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Get after Delete error = %v, want %v", err, port.ErrNotFound)
	}
	if _, err := repos.Workspaces.ForecastCategories(t.Context(), userID, w.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("ForecastCategories after Delete error = %v, want %v", err, port.ErrNotFound)
	}
}

func TestWorkspaces_TheDatabaseRefusesAnInvalidDivisor(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")

	invalid := w
	invalid.ForecastDivisor = 27
	if err := repos.Workspaces.Update(t.Context(), userID, invalid); err == nil {
		t.Fatal("Update(divisor 27) error = nil, want the check constraint to refuse it")
	}
	if got, err := repos.Workspaces.Get(t.Context(), userID, w.ID); err != nil || got.ForecastDivisor != 30 {
		t.Errorf("divisor after the refused update = %d, %v; want 30", got.ForecastDivisor, err)
	}
}

func TestWorkspaces_AnotherUserCannotReachThem(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	w := newWorkspace(t, repos, owner, "Main")
	ctx := t.Context()

	stolen := w
	stolen.Name = "Mine now"
	attempts := map[string]error{
		"get":            func() error { _, err := repos.Workspaces.Get(ctx, intruder, w.ID); return err }(),
		"update":         repos.Workspaces.Update(ctx, intruder, stolen),
		"archive":        repos.Workspaces.SetArchived(ctx, intruder, w.ID, true),
		"delete":         repos.Workspaces.Delete(ctx, intruder, w.ID),
		"categories":     func() error { _, err := repos.Workspaces.ForecastCategories(ctx, intruder, w.ID); return err }(),
		"set categories": repos.Workspaces.ReplaceForecastCategories(ctx, intruder, w.ID, []domain.ForecastCategory{{Name: "X", MonthlyAmount: 1}}),
	}
	for name, err := range attempts {
		if !errors.Is(err, port.ErrNotFound) {
			t.Errorf("%s by another user: error = %v, want %v", name, err, port.ErrNotFound)
		}
	}
	if list, err := repos.Workspaces.List(ctx, intruder, true); err != nil || len(list) != 0 {
		t.Errorf("List by another user = %+v, %v; want nothing", list, err)
	}
	if got, err := repos.Workspaces.Get(ctx, owner, w.ID); err != nil || got != w {
		t.Errorf("owner's workspace after the attempts = %+v, %v; want it untouched", got, err)
	}
	if cats, err := repos.Workspaces.ForecastCategories(ctx, owner, w.ID); err != nil || len(cats) != 0 {
		t.Errorf("owner's categories after the attempts = %+v, %v; want none", cats, err)
	}
}

func TestWorkspaces_ADivisorBeyondSmallintIsRefusedNotWrapped(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")

	huge := w
	huge.ForecastDivisor = 65566 // wraps to 30 as an int16
	if err := repos.Workspaces.Update(t.Context(), userID, huge); !errors.Is(err, domain.ErrForecastDivisorOutOfRange) {
		t.Errorf("Update(divisor 65566) error = %v, want %v", err, domain.ErrForecastDivisorOutOfRange)
	}
	if got, err := repos.Workspaces.Get(t.Context(), userID, w.ID); err != nil || got != w {
		t.Errorf("workspace after the refused update = %+v, %v; want it unchanged", got, err)
	}
}
