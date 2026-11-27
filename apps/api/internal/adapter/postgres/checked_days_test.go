//go:build integration

package postgres_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

func TestCheckedDays_SetUnsetAndList(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	ctx := t.Context()

	for _, day := range []string{nov03, "2026-11-05", "2026-11-05", "2026-12-01"} {
		if err := repos.CheckedDays.Set(ctx, userID, w.ID, mustDate(t, day)); err != nil {
			t.Fatalf("Set(%s) error = %v", day, err)
		}
	}
	if err := repos.CheckedDays.Unset(ctx, userID, w.ID, mustDate(t, nov03)); err != nil {
		t.Fatalf("Unset error = %v", err)
	}
	if err := repos.CheckedDays.Unset(ctx, userID, w.ID, mustDate(t, "2026-11-20")); err != nil {
		t.Errorf("Unset(a day never checked) error = %v, want nil", err)
	}
	got, err := repos.CheckedDays.List(ctx, userID, w.ID, mustDate(t, "2026-11-01"), mustDate(t, "2026-11-30"))
	if err != nil || !slices.Equal(got, []domain.Date{mustDate(t, "2026-11-05")}) {
		t.Errorf("List(November) = %v, %v; want [2026-11-05] once", got, err)
	}
}

func TestCheckedDays_AnotherUserCannotReachThem(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	w := newWorkspace(t, repos, owner, "Main")
	day := mustDate(t, "2026-11-05")
	ctx := t.Context()
	if err := repos.CheckedDays.Set(ctx, owner, w.ID, day); err != nil {
		t.Fatalf("Set error = %v", err)
	}

	if err := repos.CheckedDays.Set(ctx, intruder, w.ID, day); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Set by another user error = %v, want %v", err, port.ErrNotFound)
	}
	if err := repos.CheckedDays.Unset(ctx, intruder, w.ID, day); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Unset by another user error = %v, want %v", err, port.ErrNotFound)
	}
	if _, err := repos.CheckedDays.List(ctx, intruder, w.ID, day, day); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("List by another user error = %v, want %v", err, port.ErrNotFound)
	}
	if got, err := repos.CheckedDays.List(ctx, owner, w.ID, day, day); err != nil || len(got) != 1 {
		t.Errorf("owner's checked days after the attempts = %v, %v; want the day still checked", got, err)
	}
}
