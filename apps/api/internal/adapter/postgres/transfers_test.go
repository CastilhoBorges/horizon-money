//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

func newTransfer(t *testing.T, repos port.Repositories, userID domain.ID, from, to domain.Workspace, day string) domain.Transfer {
	t.Helper()
	tr, err := repos.Transfers.Create(t.Context(), userID, domain.Transfer{
		FromWorkspaceID: from.ID, ToWorkspaceID: to.ID, AmountSent: 100000, AmountReceived: 18500,
		Date: mustDate(t, day), Description: "savings abroad",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	return tr
}

func TestTransfers_OneRowSeenFromBothWorkspaces(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	brlWorkspace, usdWorkspace := newWorkspace(t, repos, userID, "BRL"), newWorkspace(t, repos, userID, "USD")
	tr := newTransfer(t, repos, userID, brlWorkspace, usdWorkspace, "2026-11-10")
	newTransfer(t, repos, userID, brlWorkspace, usdWorkspace, "2026-12-10")

	if got, err := repos.Transfers.Get(t.Context(), userID, tr.ID); err != nil || got != tr || tr.ID.IsZero() {
		t.Errorf("Get = %+v, %v; want %+v", got, err, tr)
	}
	for _, w := range []domain.Workspace{brlWorkspace, usdWorkspace} {
		got, err := repos.Transfers.ListForWorkspace(t.Context(), userID, w.ID, mustDate(t, "2026-11-01"), mustDate(t, "2026-11-30"))
		if err != nil || len(got) != 1 || got[0] != tr {
			t.Errorf("November transfers of %s = %+v, %v; want only the Nov 10 transfer", w.Name, got, err)
		}
	}
}

func TestTransfers_AWorkspaceWithTransfersCannotBeDeleted(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	from, to := newWorkspace(t, repos, userID, "From"), newWorkspace(t, repos, userID, "To")
	tr := newTransfer(t, repos, userID, from, to, "2026-11-10")

	for _, w := range []domain.Workspace{from, to} {
		if err := repos.Workspaces.Delete(t.Context(), userID, w.ID); !errors.Is(err, domain.ErrWorkspaceHasTransfers) {
			t.Errorf("Delete(%s) error = %v, want %v", w.Name, err, domain.ErrWorkspaceHasTransfers)
		}
	}
	if err := repos.Transfers.Delete(t.Context(), userID, tr.ID); err != nil {
		t.Fatalf("Delete transfer error = %v", err)
	}
	if err := repos.Workspaces.Delete(t.Context(), userID, from.ID); err != nil {
		t.Errorf("Delete(From) once its transfer is gone error = %v, want nil", err)
	}
}

func TestTransfers_DeletingTheUserCascadesThroughTransfers(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	from, to := newWorkspace(t, repos, userID, "From"), newWorkspace(t, repos, userID, "To")
	newTransfer(t, repos, userID, from, to, "2026-11-10")

	if _, err := sharedPool(t).Exec(t.Context(), "delete from users where id = $1", userID.String()); err != nil {
		t.Fatalf("deleting the user error = %v, want the NO ACTION keys to let the cascade through", err)
	}
	if _, err := repos.Workspaces.Get(t.Context(), userID, from.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("workspace after the user's deletion: error = %v, want %v", err, port.ErrNotFound)
	}
}

func TestTransfers_BothWorkspacesMustBeTheCallersAndDifferent(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	mine, theirs := newWorkspace(t, repos, intruder, "Mine"), newWorkspace(t, repos, owner, "Theirs")

	_, err := repos.Transfers.Create(t.Context(), intruder, domain.Transfer{
		FromWorkspaceID: mine.ID, ToWorkspaceID: theirs.ID, AmountSent: 1, AmountReceived: 1, Date: mustDate(t, "2026-11-10"),
	})
	if !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Create into another user's workspace error = %v, want %v", err, port.ErrNotFound)
	}
	_, err = repos.Transfers.Create(t.Context(), intruder, domain.Transfer{
		FromWorkspaceID: mine.ID, ToWorkspaceID: mine.ID, AmountSent: 1, AmountReceived: 1, Date: mustDate(t, "2026-11-10"),
	})
	if err == nil {
		t.Error("Create within one workspace error = nil, want the check constraint to refuse it")
	}
}

func TestTransfers_AnotherUserCannotReachThem(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, intruder := newUser(t, repos), newUser(t, repos)
	from, to := newWorkspace(t, repos, owner, "From"), newWorkspace(t, repos, owner, "To")
	tr := newTransfer(t, repos, owner, from, to, "2026-11-10")

	if _, err := repos.Transfers.Get(t.Context(), intruder, tr.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Get by another user error = %v, want %v", err, port.ErrNotFound)
	}
	if err := repos.Transfers.Delete(t.Context(), intruder, tr.ID); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Delete by another user error = %v, want %v", err, port.ErrNotFound)
	}
	if got, err := repos.Transfers.ListForWorkspace(t.Context(), intruder, from.ID, domain.Date{}, mustDate(t, "2027-01-01")); err != nil || len(got) != 0 {
		t.Errorf("ListForWorkspace by another user = %+v, %v; want nothing", got, err)
	}
	if got, err := repos.Transfers.Get(t.Context(), owner, tr.ID); err != nil || got != tr {
		t.Errorf("owner's transfer after the attempts = %+v, %v; want it untouched", got, err)
	}
}
