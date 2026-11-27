package postgres

import (
	"context"
	"fmt"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

type checkedDays struct{ repo }

func (r checkedDays) Set(ctx context.Context, userID, workspaceID domain.ID, day domain.Date) error {
	if err := r.workspaceExists(ctx, r.q, userID, workspaceID); err != nil {
		return err
	}
	err := r.q.SetCheckedDay(ctx, sqlcgen.SetCheckedDayParams{WorkspaceID: toUUID(workspaceID), UserID: toUUID(userID), Date: toDate(day)})
	if err != nil {
		return r.fail(ctx, "set", err, "workspace_id", workspaceID.String())
	}
	return nil
}

func (r checkedDays) Unset(ctx context.Context, userID, workspaceID domain.ID, day domain.Date) error {
	if err := r.workspaceExists(ctx, r.q, userID, workspaceID); err != nil {
		return err
	}
	err := r.q.UnsetCheckedDay(ctx, sqlcgen.UnsetCheckedDayParams{WorkspaceID: toUUID(workspaceID), UserID: toUUID(userID), Date: toDate(day)})
	if err != nil {
		return r.fail(ctx, "unset", err, "workspace_id", workspaceID.String())
	}
	return nil
}

func (r checkedDays) List(ctx context.Context, userID, workspaceID domain.ID, from, to domain.Date) ([]domain.Date, error) {
	if err := r.workspaceExists(ctx, r.q, userID, workspaceID); err != nil {
		return nil, err
	}
	rows, err := r.q.ListCheckedDays(ctx, sqlcgen.ListCheckedDaysParams{
		WorkspaceID: toUUID(workspaceID), UserID: toUUID(userID), FromDate: toDate(from), ToDate: toDate(to),
	})
	if err != nil {
		return nil, r.fail(ctx, "list", err, "workspace_id", workspaceID.String())
	}
	out := make([]domain.Date, 0, len(rows))
	for _, row := range rows {
		day, err := fromDate(row)
		if err != nil {
			return nil, r.fail(ctx, "read", fmt.Errorf("corrupt row: %w", err), "workspace_id", workspaceID.String())
		}
		out = append(out, day)
	}
	return out, nil
}
