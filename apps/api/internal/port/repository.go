package port

import (
	"context"
	"errors"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

// ErrNotFound means the record does not exist for this user. A record of another user is reported the
// same way, so callers can never tell the two apart.
var ErrNotFound = errors.New("port: not found")

// NewUser is what creating a user takes; the settings start at their seeded defaults.
type NewUser struct {
	Email       string
	DisplayName string
}

// Users stores people and their settings.
type Users interface {
	Create(ctx context.Context, u NewUser) (domain.User, error)
	Get(ctx context.Context, userID domain.ID) (domain.User, error)
}

// NewWorkspace is what creating a workspace takes; its threshold and divisor start at their seeded
// defaults.
type NewWorkspace struct {
	Name           string
	Currency       domain.Currency
	InitialBalance domain.Amount
	Start          domain.Date
}

// Workspaces stores a user's workspaces and their forecast categories.
type Workspaces interface {
	Create(ctx context.Context, userID domain.ID, w NewWorkspace) (domain.Workspace, error)
	Get(ctx context.Context, userID, id domain.ID) (domain.Workspace, error)
	// List returns the workspaces in creation order, the archived ones only when asked.
	List(ctx context.Context, userID domain.ID, includeArchived bool) ([]domain.Workspace, error)
	// Update saves every field but the id and the archived flag.
	Update(ctx context.Context, userID domain.ID, w domain.Workspace) error
	SetArchived(ctx context.Context, userID, id domain.ID, archived bool) error
	// Delete removes the workspace with its movements, categories and checked days. A workspace with
	// transfers is refused with domain.ErrWorkspaceHasTransfers.
	Delete(ctx context.Context, userID, id domain.ID) error
	ForecastCategories(ctx context.Context, userID, workspaceID domain.ID) ([]domain.ForecastCategory, error)
	// ReplaceForecastCategories swaps the whole set at once.
	ReplaceForecastCategories(ctx context.Context, userID, workspaceID domain.ID, categories []domain.ForecastCategory) error
}

// NewTag is what creating a tag takes.
type NewTag struct {
	Name  string
	Color domain.TagColor
}

// Tags stores a user's tags. A name is unique per user, ignoring case (domain.ErrTagNameTaken).
type Tags interface {
	Create(ctx context.Context, userID domain.ID, t NewTag) (domain.Tag, error)
	// List returns the tags sorted by name, ignoring case.
	List(ctx context.Context, userID domain.ID) ([]domain.Tag, error)
}

// Transfers stores transfers, one row with both sides. Both workspaces must be the caller's.
type Transfers interface {
	// Create ignores t.ID. A workspace that is missing or another user's gives ErrNotFound.
	Create(ctx context.Context, userID domain.ID, t domain.Transfer) (domain.Transfer, error)
	Get(ctx context.Context, userID, id domain.ID) (domain.Transfer, error)
	Delete(ctx context.Context, userID, id domain.ID) error
	// ListForWorkspace returns the transfers into or out of the workspace dated from..to, inclusive, by
	// date. A zero from means no lower bound.
	ListForWorkspace(ctx context.Context, userID, workspaceID domain.ID, from, to domain.Date) ([]domain.Transfer, error)
}

// CheckedDays stores the display-only "I reviewed this day" marks.
type CheckedDays interface {
	// Set marks the day; marking it again changes nothing.
	Set(ctx context.Context, userID, workspaceID domain.ID, day domain.Date) error
	// Unset clears the mark; clearing a day never marked changes nothing.
	Unset(ctx context.Context, userID, workspaceID domain.ID, day domain.Date) error
	List(ctx context.Context, userID, workspaceID domain.ID, from, to domain.Date) ([]domain.Date, error)
}

// Movements stores one-off movements and series, with their tags and occurrence edits. A workspace,
// movement or tag that is missing or another user's gives ErrNotFound.
type Movements interface {
	// Create stores m with its tags and ignores m.ID. A series starts its own group unless m.GroupID
	// names one, as the tail of a following split does.
	Create(ctx context.Context, userID domain.ID, m domain.Movement) (domain.Movement, error)
	Get(ctx context.Context, userID, id domain.ID) (domain.Movement, error)
	// Update saves the template, the repetition, the group, the description and the tags.
	Update(ctx context.Context, userID domain.ID, m domain.Movement) error
	// Delete removes the movement with its tags and occurrence edits.
	Delete(ctx context.Context, userID, id domain.ID) error
	// Group lists the movements of a series group by start date, for all edits.
	Group(ctx context.Context, userID, groupID domain.ID) ([]domain.Movement, error)
	// PutEdit stores the change of one occurrence, replacing any earlier change of the same occurrence.
	PutEdit(ctx context.Context, userID, movementID domain.ID, e domain.OccurrenceEdit) error
	// Edits lists a series' occurrence edits by original date.
	Edits(ctx context.Context, userID, movementID domain.ID) ([]domain.OccurrenceEdit, error)
	// ForLedger loads what the workspace's ledger from start through to needs (persistence spec §6): the
	// one-offs in the range, every series that can produce an occurrence there, and their edits by
	// movement id. domain.NewLedger turns them into the ledger.
	ForLedger(ctx context.Context, userID, workspaceID domain.ID, start, to domain.Date) ([]domain.Movement, map[domain.ID][]domain.OccurrenceEdit, error)
}

// Repositories are every repository, bound to one connection or one transaction.
type Repositories struct {
	Users       Users
	Workspaces  Workspaces
	Tags        Tags
	Movements   Movements
	Transfers   Transfers
	CheckedDays CheckedDays
}
