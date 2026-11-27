//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

func oneOff(t *testing.T, w domain.Workspace, day string, kind domain.MovementType, amount domain.Amount) domain.Movement {
	t.Helper()
	return domain.Movement{WorkspaceID: w.ID, Entry: domain.Entry{Date: mustDate(t, day), Type: kind, Amount: amount}}
}

func monthlySeries(t *testing.T, w domain.Workspace, start string, amount domain.Amount) domain.Movement {
	t.Helper()
	s, err := domain.NewRecurring(domain.TypeExpense, domain.FrequencyMonthly, amount, mustDate(t, start), domain.Date{})
	if err != nil {
		t.Fatalf("NewRecurring error = %v", err)
	}
	return domain.Movement{WorkspaceID: w.ID, Description: "rent", Series: &s}
}

func create(t *testing.T, repos port.Repositories, userID domain.ID, m domain.Movement) domain.Movement {
	t.Helper()
	created, err := repos.Movements.Create(t.Context(), userID, m)
	if err != nil {
		t.Fatalf("create movement: %v", err)
	}
	return created
}

func TestMovements_OneOffWithTagsRoundTrips(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	market, lunch := newTag(t, repos, userID, "market"), newTag(t, repos, userID, "lunch")

	m := oneOff(t, w, nov03, domain.TypeDaily, 2340)
	m.Description, m.TagIDs = "groceries", sortedIDs(market.ID, lunch.ID)
	created := create(t, repos, userID, m)
	if created.ID.IsZero() || !created.GroupID.IsZero() {
		t.Fatalf("Create = %+v, want an id and no series group", created)
	}
	m.ID = created.ID
	if got, err := repos.Movements.Get(t.Context(), userID, created.ID); err != nil || !reflect.DeepEqual(got, m) || !reflect.DeepEqual(created, m) {
		t.Errorf("Get = %+v, %v; want %+v", got, err, m)
	}
}

func TestMovements_SeriesStartsItsOwnGroupAndASplitTailJoinsIt(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")

	purchase, err := domain.NewInstallments(domain.TypeCard, 100000, 12, mustDate(t, "2026-08-15"))
	if err != nil {
		t.Fatalf("NewInstallments error = %v", err)
	}
	head := create(t, repos, userID, domain.Movement{WorkspaceID: w.ID, Series: &purchase})
	if head.GroupID != head.ID {
		t.Fatalf("a new series has group %s, want its own id %s", head.GroupID, head.ID)
	}
	if got, err := repos.Movements.Get(t.Context(), userID, head.ID); err != nil || !reflect.DeepEqual(got, head) {
		t.Errorf("Get = %+v, %v; want the installments back field for field: %+v", got, err, head)
	}

	before, after, err := purchase.SplitAt(mustDate(t, "2026-12-15"))
	if err != nil {
		t.Fatalf("SplitAt error = %v", err)
	}
	head.Series = &before
	if err := repos.Movements.Update(t.Context(), userID, head); err != nil {
		t.Fatalf("Update head error = %v", err)
	}
	tail := create(t, repos, userID, domain.Movement{WorkspaceID: w.ID, Series: &after, GroupID: head.GroupID})

	group, err := repos.Movements.Group(t.Context(), userID, head.GroupID)
	if err != nil || len(group) != 2 || !reflect.DeepEqual(group[0], head) || !reflect.DeepEqual(group[1], tail) {
		t.Errorf("Group = %+v, %v; want the head then the tail", group, err)
	}
}

func TestMovements_UpdateReplacesTheTemplateAndTags(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	old, current := newTag(t, repos, userID, "old"), newTag(t, repos, userID, "current")

	m := oneOff(t, w, nov03, domain.TypeDaily, 2340)
	m.TagIDs = []domain.ID{old.ID}
	m = create(t, repos, userID, m)

	weekly, err := domain.NewRecurring(domain.TypeDaily, domain.FrequencyWeekly, 1500, mustDate(t, "2026-11-02"), mustDate(t, "2026-12-28"))
	if err != nil {
		t.Fatalf("NewRecurring error = %v", err)
	}
	m.Series, m.Entry, m.GroupID, m.Description, m.TagIDs = &weekly, domain.Entry{}, m.ID, "coffee", []domain.ID{current.ID}
	if err := repos.Movements.Update(t.Context(), userID, m); err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if got, err := repos.Movements.Get(t.Context(), userID, m.ID); err != nil || !reflect.DeepEqual(got, m) {
		t.Errorf("Get after turning a one-off into a weekly series = %+v, %v; want %+v", got, err, m)
	}
}

func TestMovements_OccurrenceEditsAreUpsertedAndDeletedWithTheMovement(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	late := newTag(t, repos, userID, "late")
	rent := create(t, repos, userID, monthlySeries(t, w, "2026-01-05", 120000))
	ctx := t.Context()

	nov := domain.OccurrenceEdit{
		Original:  mustDate(t, "2026-11-05"),
		Exception: domain.Exception{Date: mustDate(t, "2026-11-07"), Amount: 130000}, Description: "paid late", TagIDs: []domain.ID{late.ID},
	}
	dec := domain.OccurrenceEdit{Original: mustDate(t, "2026-12-05"), Exception: domain.Exception{Deleted: true}}
	for _, e := range []domain.OccurrenceEdit{dec, {Original: nov.Original, Exception: domain.Exception{Deleted: true}}, nov} {
		if err := repos.Movements.PutEdit(ctx, userID, rent.ID, e); err != nil {
			t.Fatalf("PutEdit(%s) error = %v", e.Original, err)
		}
	}
	got, err := repos.Movements.Edits(ctx, userID, rent.ID)
	if want := []domain.OccurrenceEdit{nov, dec}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Edits = %+v, %v; want the November override (upserted over its deletion) and the December deletion", got, err)
	}

	if err := repos.Movements.Delete(ctx, userID, rent.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := repos.Movements.Get(ctx, userID, rent.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Get after Delete error = %v, want %v", err, port.ErrNotFound)
	}
	if _, err := repos.Movements.Edits(ctx, userID, rent.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Edits after Delete error = %v, want %v", err, port.ErrNotFound)
	}
}

func TestMovements_AnotherUserCannotReachThem(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	w := newWorkspace(t, repos, owner, "Main")
	ownerTag := newTag(t, repos, owner, "mine")
	rent := create(t, repos, owner, monthlySeries(t, w, "2026-01-05", 120000))
	ctx := t.Context()

	intrudersWorkspace := newWorkspace(t, repos, intruder, "Theirs")
	withOwnersTag := oneOff(t, intrudersWorkspace, nov03, domain.TypeDaily, 1)
	withOwnersTag.TagIDs = []domain.ID{ownerTag.ID}
	stolen := rent
	stolen.Description = "mine now"
	edit := domain.OccurrenceEdit{Original: mustDate(t, "2026-11-05"), Exception: domain.Exception{Deleted: true}}
	attempts := map[string]error{
		"get":      func() error { _, err := repos.Movements.Get(ctx, intruder, rent.ID); return err }(),
		"update":   repos.Movements.Update(ctx, intruder, stolen),
		"delete":   repos.Movements.Delete(ctx, intruder, rent.ID),
		"put edit": repos.Movements.PutEdit(ctx, intruder, rent.ID, edit),
		"edits":    func() error { _, err := repos.Movements.Edits(ctx, intruder, rent.ID); return err }(),
		"create in their ws": func() error {
			_, err := repos.Movements.Create(ctx, intruder, oneOff(t, w, nov03, domain.TypeDaily, 1))
			return err
		}(),
		"tag with their tag": func() error { _, err := repos.Movements.Create(ctx, intruder, withOwnersTag); return err }(),
		"ledger of their ws": func() error {
			_, _, err := repos.Movements.ForLedger(ctx, intruder, w.ID, mustDate(t, "2026-12-31"))
			return err
		}(),
	}
	for name, err := range attempts {
		if !errors.Is(err, port.ErrNotFound) {
			t.Errorf("%s by another user: error = %v, want %v", name, err, port.ErrNotFound)
		}
	}
	if group, err := repos.Movements.Group(ctx, intruder, rent.GroupID); err != nil || len(group) != 0 {
		t.Errorf("Group by another user = %+v, %v; want nothing", group, err)
	}
	if got, err := repos.Movements.Get(ctx, owner, rent.ID); err != nil || !reflect.DeepEqual(got, rent) {
		t.Errorf("owner's movement after the attempts = %+v, %v; want it untouched", got, err)
	}
	if edits, err := repos.Movements.Edits(ctx, owner, rent.ID); err != nil || len(edits) != 0 {
		t.Errorf("owner's edits after the attempts = %+v, %v; want none", edits, err)
	}
}

func TestMovements_TheDatabaseIsTheSecondLock(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	w := newWorkspace(t, repos, owner, "Main")
	m := create(t, repos, owner, oneOff(t, w, nov03, domain.TypeDaily, 100))
	intrudersTag := newTag(t, repos, intruder, "theirs")
	pool := sharedPool(t)

	raw := map[string]struct {
		sql  string
		args []any
	}{
		"a tag of another user on a movement": {
			sql:  "insert into movement_tags (movement_id, tag_id, user_id) values ($1, $2, $3)",
			args: []any{m.ID.String(), intrudersTag.ID.String(), owner.String()},
		},
		"a series row without its anchor day": {
			sql: `insert into movements (user_id, workspace_id, type, amount, date, frequency, series_group_id)
			      values ($1, $2, 'expense', 100, '2026-11-01', 'monthly', uuidv7())`,
			args: []any{owner.String(), w.ID.String()},
		},
		"a one-off row with a part count": {
			sql:  "insert into movements (user_id, workspace_id, type, amount, date, parts) values ($1, $2, 'expense', 100, '2026-11-01', 3)",
			args: []any{owner.String(), w.ID.String()},
		},
	}
	for name, stmt := range raw {
		if _, err := pool.Exec(t.Context(), stmt.sql, stmt.args...); err == nil {
			t.Errorf("%s: insert succeeded, want the schema to refuse it", name)
		}
	}
}

func TestMovements_AnInvalidSeriesIsRefusedOnWrite(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	offAnchor := domain.Series{Type: domain.TypeExpense, Frequency: domain.FrequencyMonthly, Start: mustDate(t, "2026-11-10"), AnchorDay: 15, Amount: 100}

	if _, err := repos.Movements.Create(t.Context(), userID, domain.Movement{WorkspaceID: w.ID, Series: &offAnchor}); !errors.Is(err, domain.ErrInvalidSeries) {
		t.Errorf("Create(series off its anchor day) error = %v, want %v", err, domain.ErrInvalidSeries)
	}
	rent := create(t, repos, userID, monthlySeries(t, w, "2026-01-05", 120000))
	broken := rent
	broken.Series = &offAnchor
	if err := repos.Movements.Update(t.Context(), userID, broken); !errors.Is(err, domain.ErrInvalidSeries) {
		t.Errorf("Update(series off its anchor day) error = %v, want %v", err, domain.ErrInvalidSeries)
	}
	if movements, _, err := repos.Movements.ForLedger(t.Context(), userID, w.ID, mustDate(t, "2026-12-31")); err != nil || len(movements) != 1 {
		t.Errorf("ForLedger after the refused writes = %d movements, %v; want the rent only, readable", len(movements), err)
	}
}

func TestMovements_AnotherUserCannotOverwriteAStoredEdit(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	w := newWorkspace(t, repos, owner, "Main")
	rent := create(t, repos, owner, monthlySeries(t, w, "2026-01-05", 120000))
	moved := domain.OccurrenceEdit{Original: mustDate(t, "2026-11-05"), Exception: domain.Exception{Date: mustDate(t, "2026-11-07"), Amount: 130000}}
	if err := repos.Movements.PutEdit(t.Context(), owner, rent.ID, moved); err != nil {
		t.Fatalf("PutEdit error = %v", err)
	}

	deletion := domain.OccurrenceEdit{Original: moved.Original, Exception: domain.Exception{Deleted: true}}
	if err := repos.Movements.PutEdit(t.Context(), intruder, rent.ID, deletion); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("PutEdit over the owner's edit by another user: error = %v, want %v", err, port.ErrNotFound)
	}
	if edits, err := repos.Movements.Edits(t.Context(), owner, rent.ID); err != nil || !reflect.DeepEqual(edits, []domain.OccurrenceEdit{moved}) {
		t.Errorf("owner's edits = %+v, %v; want the moved occurrence untouched", edits, err)
	}
}

func TestMovements_ARepeatedTagIsStoredOnce(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	w := newWorkspace(t, repos, userID, "Main")
	tag := newTag(t, repos, userID, "market")

	m := monthlySeries(t, w, "2026-01-05", 120000)
	m.TagIDs = []domain.ID{tag.ID, tag.ID}
	created, err := repos.Movements.Create(t.Context(), userID, m)
	if err != nil {
		t.Fatalf("Create with a repeated tag error = %v, want nil", err)
	}
	if got, err := repos.Movements.Get(t.Context(), userID, created.ID); err != nil || !reflect.DeepEqual(got.TagIDs, []domain.ID{tag.ID}) {
		t.Errorf("tags = %v, %v; want the tag once", got.TagIDs, err)
	}

	edit := domain.OccurrenceEdit{Original: mustDate(t, "2026-11-05"), Exception: domain.Exception{Date: mustDate(t, "2026-11-05"), Amount: 1}, TagIDs: []domain.ID{tag.ID, tag.ID}}
	if err := repos.Movements.PutEdit(t.Context(), userID, created.ID, edit); err != nil {
		t.Fatalf("PutEdit with a repeated tag error = %v, want nil", err)
	}
	if edits, err := repos.Movements.Edits(t.Context(), userID, created.ID); err != nil || len(edits) != 1 || !reflect.DeepEqual(edits[0].TagIDs, []domain.ID{tag.ID}) {
		t.Errorf("edit tags = %+v, %v; want the tag once", edits, err)
	}
}
