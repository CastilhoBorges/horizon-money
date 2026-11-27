package postgres

import (
	"context"
	"fmt"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

type workspaces struct{ repo }

func (r workspaces) Create(ctx context.Context, userID domain.ID, w port.NewWorkspace) (domain.Workspace, error) {
	row, err := r.q.CreateWorkspace(ctx, sqlcgen.CreateWorkspaceParams{
		UserID: toUUID(userID), Name: w.Name, Currency: w.Currency.Code(),
		InitialBalance: int64(w.InitialBalance), InitialBalanceDate: toDate(w.Start),
	})
	if err != nil {
		return domain.Workspace{}, r.fail(ctx, "create", err, "user_id", userID.String())
	}
	return r.toWorkspace(ctx, row)
}

func (r workspaces) Get(ctx context.Context, userID, id domain.ID) (domain.Workspace, error) {
	row, err := r.q.GetWorkspace(ctx, sqlcgen.GetWorkspaceParams{ID: toUUID(id), UserID: toUUID(userID)})
	if isNoRows(err) {
		return domain.Workspace{}, port.ErrNotFound
	}
	if err != nil {
		return domain.Workspace{}, r.fail(ctx, "get", err, "workspace_id", id.String())
	}
	return r.toWorkspace(ctx, row)
}

func (r workspaces) List(ctx context.Context, userID domain.ID, includeArchived bool) ([]domain.Workspace, error) {
	rows, err := r.q.ListWorkspaces(ctx, sqlcgen.ListWorkspacesParams{UserID: toUUID(userID), IncludeArchived: includeArchived})
	if err != nil {
		return nil, r.fail(ctx, "list", err, "user_id", userID.String())
	}
	out := make([]domain.Workspace, 0, len(rows))
	for _, row := range rows {
		w, err := r.toWorkspace(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func (r workspaces) Update(ctx context.Context, userID domain.ID, w domain.Workspace) error {
	n, err := r.q.UpdateWorkspace(ctx, sqlcgen.UpdateWorkspaceParams{
		ID: toUUID(w.ID), UserID: toUUID(userID), Name: w.Name, Currency: w.Currency.Code(),
		InitialBalance: int64(w.InitialBalance), InitialBalanceDate: toDate(w.Start),
		LowBalanceThreshold: int64(w.LowThreshold), ForecastDivisor: int16(w.ForecastDivisor), //nolint:gosec // 28–31, checked by the domain and the schema
	})
	return r.affected(ctx, "update", n, err, w.ID)
}

func (r workspaces) SetArchived(ctx context.Context, userID, id domain.ID, archived bool) error {
	n, err := r.q.SetWorkspaceArchived(ctx, sqlcgen.SetWorkspaceArchivedParams{ID: toUUID(id), UserID: toUUID(userID), Archived: archived})
	return r.affected(ctx, "archive", n, err, id)
}

func (r workspaces) Delete(ctx context.Context, userID, id domain.ID) error {
	n, err := r.q.DeleteWorkspace(ctx, sqlcgen.DeleteWorkspaceParams{ID: toUUID(id), UserID: toUUID(userID)})
	if violates(err, foreignKeyViolation, "transfers_from_workspace_fkey") || violates(err, foreignKeyViolation, "transfers_to_workspace_fkey") {
		return domain.ErrWorkspaceHasTransfers
	}
	return r.affected(ctx, "delete", n, err, id)
}

func (r workspaces) ForecastCategories(ctx context.Context, userID, workspaceID domain.ID) ([]domain.ForecastCategory, error) {
	if err := workspaceExists(ctx, r.q, userID, workspaceID); err != nil {
		return nil, r.done(ctx, "list_categories", err, "workspace_id", workspaceID.String())
	}
	rows, err := r.q.ListForecastCategories(ctx, sqlcgen.ListForecastCategoriesParams{WorkspaceID: toUUID(workspaceID), UserID: toUUID(userID)})
	if err != nil {
		return nil, r.done(ctx, "list_categories", err, "workspace_id", workspaceID.String())
	}
	out := make([]domain.ForecastCategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ForecastCategory{ID: fromUUID(row.ID), Name: row.Name, MonthlyAmount: domain.Amount(row.MonthlyAmount)})
	}
	return out, nil
}

func (r workspaces) ReplaceForecastCategories(ctx context.Context, userID, workspaceID domain.ID, categories []domain.ForecastCategory) error {
	err := r.inTx(ctx, func(q *sqlcgen.Queries) error {
		if err := workspaceExists(ctx, q, userID, workspaceID); err != nil {
			return err
		}
		ids := sqlcgen.DeleteForecastCategoriesParams{WorkspaceID: toUUID(workspaceID), UserID: toUUID(userID)}
		if err := q.DeleteForecastCategories(ctx, ids); err != nil {
			return err
		}
		for _, c := range categories {
			err := q.CreateForecastCategory(ctx, sqlcgen.CreateForecastCategoryParams{
				UserID: toUUID(userID), WorkspaceID: toUUID(workspaceID), Name: c.Name, MonthlyAmount: int64(c.MonthlyAmount),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return r.done(ctx, "replace_categories", err, "workspace_id", workspaceID.String())
}

func (r workspaces) toWorkspace(ctx context.Context, row sqlcgen.Workspace) (domain.Workspace, error) {
	currency, err := domain.ParseCurrency(row.Currency)
	if err == nil {
		var start domain.Date
		if start, err = fromDate(row.InitialBalanceDate); err == nil {
			return domain.Workspace{
				ID: fromUUID(row.ID), Name: row.Name, Currency: currency,
				InitialBalance: domain.Amount(row.InitialBalance), Start: start,
				LowThreshold: domain.Amount(row.LowBalanceThreshold), ForecastDivisor: int(row.ForecastDivisor),
				Archived: row.ArchivedAt.Valid,
			}, nil
		}
	}
	return domain.Workspace{}, r.fail(ctx, "read", fmt.Errorf("corrupt row: %w", err), "workspace_id", fromUUID(row.ID).String())
}
