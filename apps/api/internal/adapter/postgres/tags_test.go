//go:build integration

package postgres_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

func newTag(t *testing.T, repos port.Repositories, userID domain.ID, name string) domain.Tag {
	t.Helper()
	tag, err := repos.Tags.Create(t.Context(), userID, port.NewTag{Name: name, Color: domain.TagBlue})
	if err != nil {
		t.Fatalf("create tag %q: %v", name, err)
	}
	return tag
}

func TestTags_CreateAndListByName(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	market := newTag(t, repos, userID, "market")
	newTag(t, repos, userID, "Bills")

	got, err := repos.Tags.List(t.Context(), userID)
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	var names []string
	for _, tag := range got {
		names = append(names, tag.Name)
	}
	if !slices.Equal(names, []string{"Bills", "market"}) || got[1] != market || market.Color != domain.TagBlue {
		t.Errorf("List = %+v, want Bills then market, sorted ignoring case", got)
	}
}

func TestTags_NameIsUniquePerUserIgnoringCase(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	owner, other := newUser(t, repos), newUser(t, repos)
	newTag(t, repos, owner, "Market")

	if _, err := repos.Tags.Create(t.Context(), owner, port.NewTag{Name: "MARKET", Color: domain.TagRed}); !errors.Is(err, domain.ErrTagNameTaken) {
		t.Errorf("Create(same name, other case) error = %v, want %v", err, domain.ErrTagNameTaken)
	}
	newTag(t, repos, other, "Market") // another user may use the same name
	if got, err := repos.Tags.List(t.Context(), other); err != nil || len(got) != 1 {
		t.Errorf("the other user's tags = %+v, %v; want only their own", got, err)
	}
}

func TestTags_TheDatabaseRefusesAnUnknownColor(t *testing.T) {
	t.Parallel()

	repos := newStore(t).Repositories()
	userID := newUser(t, repos)
	if _, err := repos.Tags.Create(t.Context(), userID, port.NewTag{Name: "x", Color: "orange"}); err == nil {
		t.Error("Create(color orange) error = nil, want the check constraint to refuse it")
	}
}
