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

// Repositories are every repository, bound to one connection or one transaction.
type Repositories struct {
	Users      Users
	Workspaces Workspaces
}
