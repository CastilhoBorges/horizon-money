package postgres

import (
	"context"
	"fmt"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

type transfers struct{ repo }

func (r transfers) Create(ctx context.Context, userID domain.ID, t domain.Transfer) (domain.Transfer, error) {
	row, err := r.q.CreateTransfer(ctx, sqlcgen.CreateTransferParams{
		UserID: toUUID(userID), FromWorkspaceID: toUUID(t.FromWorkspaceID), ToWorkspaceID: toUUID(t.ToWorkspaceID),
		AmountSent: int64(t.AmountSent), AmountReceived: int64(t.AmountReceived), Date: toDate(t.Date),
		Description: toText(t.Description),
	})
	if violates(err, foreignKeyViolation, "transfers_from_workspace_fkey") || violates(err, foreignKeyViolation, "transfers_to_workspace_fkey") {
		return domain.Transfer{}, port.ErrNotFound
	}
	if err != nil {
		return domain.Transfer{}, r.fail(ctx, "create", err, "user_id", userID.String())
	}
	return r.toTransfer(ctx, row)
}

func (r transfers) Get(ctx context.Context, userID, id domain.ID) (domain.Transfer, error) {
	row, err := r.q.GetTransfer(ctx, sqlcgen.GetTransferParams{ID: toUUID(id), UserID: toUUID(userID)})
	if isNoRows(err) {
		return domain.Transfer{}, port.ErrNotFound
	}
	if err != nil {
		return domain.Transfer{}, r.fail(ctx, "get", err, "transfer_id", id.String())
	}
	return r.toTransfer(ctx, row)
}

func (r transfers) Delete(ctx context.Context, userID, id domain.ID) error {
	n, err := r.q.DeleteTransfer(ctx, sqlcgen.DeleteTransferParams{ID: toUUID(id), UserID: toUUID(userID)})
	return r.affected(ctx, "delete", n, err, id)
}

func (r transfers) ListForWorkspace(ctx context.Context, userID, workspaceID domain.ID, from, to domain.Date) ([]domain.Transfer, error) {
	rows, err := r.q.ListWorkspaceTransfers(ctx, sqlcgen.ListWorkspaceTransfersParams{
		UserID: toUUID(userID), WorkspaceID: toUUID(workspaceID), FromDate: toDate(from), ToDate: toDate(to),
	})
	if err != nil {
		return nil, r.fail(ctx, "list", err, "workspace_id", workspaceID.String())
	}
	out := make([]domain.Transfer, 0, len(rows))
	for _, row := range rows {
		t, err := r.toTransfer(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r transfers) toTransfer(ctx context.Context, row sqlcgen.Transfer) (domain.Transfer, error) {
	day, err := fromDate(row.Date)
	if err != nil {
		return domain.Transfer{}, r.fail(ctx, "read", fmt.Errorf("corrupt row: %w", err), "transfer_id", fromUUID(row.ID).String())
	}
	return domain.Transfer{
		ID: fromUUID(row.ID), FromWorkspaceID: fromUUID(row.FromWorkspaceID), ToWorkspaceID: fromUUID(row.ToWorkspaceID),
		AmountSent: domain.Amount(row.AmountSent), AmountReceived: domain.Amount(row.AmountReceived),
		Date: day, Description: fromText(row.Description),
	}, nil
}
