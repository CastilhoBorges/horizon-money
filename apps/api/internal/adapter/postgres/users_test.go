//go:build integration

package postgres_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

func TestUsers_CreateAppliesTheSeededDefaults(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	email := uniqueEmail(t)
	created, err := repos.Users.Create(t.Context(), port.NewUser{Email: email, DisplayName: "Ana"})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	got, err := repos.Users.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if got.ID.IsZero() || got.ID != created.ID || got.Email != email || got.DisplayName != "Ana" {
		t.Errorf("Get = %+v, want the created user %+v", got, created)
	}
	if got.MainCurrency.Code() != "BRL" || got.Timezone != "America/Sao_Paulo" ||
		got.SavingsTarget != (domain.SavingsTarget{Min: 2000, Max: 3000}) {
		t.Errorf("defaults = %s, %s, %+v; want BRL, America/Sao_Paulo, 20%%–30%%",
			got.MainCurrency.Code(), got.Timezone, got.SavingsTarget)
	}
}

func TestUsers_EmailIsUniqueIgnoringCase(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	email := uniqueEmail(t)
	if _, err := repos.Users.Create(t.Context(), port.NewUser{Email: email, DisplayName: "Ana"}); err != nil {
		t.Fatalf("Create error = %v", err)
	}
	_, err := repos.Users.Create(t.Context(), port.NewUser{Email: strings.ToUpper(email), DisplayName: "Bia"})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("Create(same email, other case) error = %v, want %v", err, domain.ErrEmailTaken)
	}
}

func TestUsers_GetUnknownIsNotFound(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	unknown, err := domain.ParseID("01a21ec4-e9fa-7aa5-8edc-d14ef635e779")
	if err != nil {
		t.Fatalf("ParseID error = %v", err)
	}
	if _, err := repos.Users.Get(t.Context(), unknown); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("Get(unknown) error = %v, want %v", err, port.ErrNotFound)
	}
}
