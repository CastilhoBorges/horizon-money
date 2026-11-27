package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

type movements struct{ repo }

// template is the columns a movement row takes from a domain.Movement, on create and on update.
type template struct {
	kind        string
	amount      int64
	description *string
	date        pgtype.Date
	frequency   *string
	endDate     pgtype.Date
	anchorDay   *int16
	lastAmount  *int64
	parts       *int16
	firstPart   *int16
	groupID     pgtype.UUID
}

func templateOf(m domain.Movement) template {
	t := template{description: toText(m.Description)}
	if m.Series == nil {
		t.kind, t.amount, t.date = string(m.Entry.Type), int64(m.Entry.Amount), toDate(m.Entry.Date)
		return t
	}
	s, frequency := m.Series, string(m.Series.Frequency)
	t.kind, t.amount, t.date = string(s.Type), int64(s.Amount), toDate(s.Start)
	t.frequency, t.endDate, t.anchorDay = &frequency, toDate(s.End), toSmallint(s.AnchorDay)
	t.lastAmount, t.parts, t.firstPart = toBigint(s.LastAmount), toSmallint(s.Parts), toSmallint(s.FirstPart)
	t.groupID = toNullUUID(m.GroupID)
	return t
}

func (r movements) Create(ctx context.Context, userID domain.ID, m domain.Movement) (domain.Movement, error) {
	t := templateOf(m)
	err := r.inTx(ctx, func(q *sqlcgen.Queries) error {
		row, err := q.CreateMovement(ctx, sqlcgen.CreateMovementParams{
			UserID: toUUID(userID), WorkspaceID: toUUID(m.WorkspaceID), Type: t.kind, Amount: t.amount,
			Description: t.description, Date: t.date, Frequency: t.frequency, EndDate: t.endDate,
			AnchorDay: t.anchorDay, LastAmount: t.lastAmount, Parts: t.parts, FirstPart: t.firstPart,
			SeriesGroupID: t.groupID,
		})
		if err != nil {
			return err
		}
		m.ID, m.GroupID = fromUUID(row.ID), fromUUID(row.SeriesGroupID)
		return addTags(ctx, q, userID, m.ID, m.TagIDs)
	})
	if err != nil {
		return domain.Movement{}, r.done(ctx, "create", err, "workspace_id", m.WorkspaceID.String())
	}
	return m, nil
}

func (r movements) Get(ctx context.Context, userID, id domain.ID) (domain.Movement, error) {
	row, err := r.q.GetMovement(ctx, sqlcgen.GetMovementParams{ID: toUUID(id), UserID: toUUID(userID)})
	if isNoRows(err) {
		return domain.Movement{}, port.ErrNotFound
	}
	if err != nil {
		return domain.Movement{}, r.done(ctx, "get", err, "movement_id", id.String())
	}
	return r.toMovement(ctx, row.Movement, row.TagIds)
}

func (r movements) Update(ctx context.Context, userID domain.ID, m domain.Movement) error {
	if m.Series != nil && m.GroupID.IsZero() {
		m.GroupID = m.ID // a one-off turned into a series starts its own group
	}
	t := templateOf(m)
	err := r.inTx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.UpdateMovement(ctx, sqlcgen.UpdateMovementParams{
			ID: toUUID(m.ID), UserID: toUUID(userID), Type: t.kind, Amount: t.amount, Description: t.description,
			Date: t.date, Frequency: t.frequency, EndDate: t.endDate, AnchorDay: t.anchorDay,
			LastAmount: t.lastAmount, Parts: t.parts, FirstPart: t.firstPart, SeriesGroupID: t.groupID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return port.ErrNotFound
		}
		if err := q.DeleteMovementTags(ctx, sqlcgen.DeleteMovementTagsParams{MovementID: toUUID(m.ID), UserID: toUUID(userID)}); err != nil {
			return err
		}
		return addTags(ctx, q, userID, m.ID, m.TagIDs)
	})
	return r.done(ctx, "update", err, "movement_id", m.ID.String())
}

func (r movements) Delete(ctx context.Context, userID, id domain.ID) error {
	n, err := r.q.DeleteMovement(ctx, sqlcgen.DeleteMovementParams{ID: toUUID(id), UserID: toUUID(userID)})
	return r.affected(ctx, "delete", n, err, id)
}

func (r movements) Group(ctx context.Context, userID, groupID domain.ID) ([]domain.Movement, error) {
	rows, err := r.q.ListGroupMovements(ctx, sqlcgen.ListGroupMovementsParams{SeriesGroupID: toUUID(groupID), UserID: toUUID(userID)})
	if err != nil {
		return nil, r.done(ctx, "group", err, "series_group_id", groupID.String())
	}
	out := make([]domain.Movement, 0, len(rows))
	for _, row := range rows {
		m, err := r.toMovement(ctx, row.Movement, row.TagIds)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (r movements) PutEdit(ctx context.Context, userID, movementID domain.ID, e domain.OccurrenceEdit) error {
	err := r.inTx(ctx, func(q *sqlcgen.Queries) error {
		if err := movementExists(ctx, q, userID, movementID); err != nil {
			return err
		}
		params := sqlcgen.UpsertMovementExceptionParams{
			MovementID: toUUID(movementID), UserID: toUUID(userID), OriginalDate: toDate(e.Original),
			Deleted: e.Exception.Deleted,
		}
		if !e.Exception.Deleted {
			params.Date, params.Amount, params.Description = toDate(e.Exception.Date), toBigint(e.Exception.Amount), toText(e.Description)
		}
		if err := q.UpsertMovementException(ctx, params); err != nil {
			return err
		}
		key := sqlcgen.DeleteExceptionTagsParams{MovementID: toUUID(movementID), OriginalDate: toDate(e.Original), UserID: toUUID(userID)}
		if err := q.DeleteExceptionTags(ctx, key); err != nil {
			return err
		}
		if e.Exception.Deleted || len(e.TagIDs) == 0 {
			return nil
		}
		return q.AddExceptionTags(ctx, sqlcgen.AddExceptionTagsParams{
			MovementID: toUUID(movementID), OriginalDate: toDate(e.Original), UserID: toUUID(userID), TagIds: toUUIDs(e.TagIDs),
		})
	})
	return r.done(ctx, "put_edit", err, "movement_id", movementID.String())
}

func (r movements) Edits(ctx context.Context, userID, movementID domain.ID) ([]domain.OccurrenceEdit, error) {
	if err := movementExists(ctx, r.q, userID, movementID); err != nil {
		return nil, r.done(ctx, "edits", err, "movement_id", movementID.String())
	}
	edits, err := r.editsOf(ctx, userID, []domain.ID{movementID})
	if err != nil {
		return nil, err
	}
	return edits[movementID], nil
}

func (r movements) ForLedger(ctx context.Context, userID, workspaceID domain.ID, start, to domain.Date) ([]domain.Movement, map[domain.ID][]domain.OccurrenceEdit, error) {
	if err := workspaceExists(ctx, r.q, userID, workspaceID); err != nil {
		return nil, nil, r.done(ctx, "for_ledger", err, "workspace_id", workspaceID.String())
	}
	rows, err := r.q.ListLedgerMovements(ctx, sqlcgen.ListLedgerMovementsParams{
		UserID: toUUID(userID), WorkspaceID: toUUID(workspaceID), StartDate: toDate(start), ToDate: toDate(to),
	})
	if err != nil {
		return nil, nil, r.done(ctx, "for_ledger", err, "workspace_id", workspaceID.String())
	}
	out := make([]domain.Movement, 0, len(rows))
	var series []domain.ID
	for _, row := range rows {
		m, err := r.toMovement(ctx, row.Movement, row.TagIds)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, m)
		if m.Series != nil {
			series = append(series, m.ID)
		}
	}
	edits, err := r.editsOf(ctx, userID, series)
	if err != nil {
		return nil, nil, err
	}
	return out, edits, nil
}

// editsOf loads the occurrence edits of the given movements, by movement id.
func (r movements) editsOf(ctx context.Context, userID domain.ID, movementIDs []domain.ID) (map[domain.ID][]domain.OccurrenceEdit, error) {
	if len(movementIDs) == 0 {
		return map[domain.ID][]domain.OccurrenceEdit{}, nil
	}
	rows, err := r.q.ListMovementExceptions(ctx, sqlcgen.ListMovementExceptionsParams{UserID: toUUID(userID), MovementIds: toUUIDs(movementIDs)})
	if err != nil {
		return nil, r.done(ctx, "edits", err, "user_id", userID.String())
	}
	out := make(map[domain.ID][]domain.OccurrenceEdit)
	for _, row := range rows {
		e := row.MovementException
		original, errOriginal := fromDate(e.OriginalDate)
		day, errDay := fromDate(e.Date)
		if errOriginal != nil || errDay != nil {
			return nil, r.fail(ctx, "read", fmt.Errorf("corrupt exception: %w %w", errOriginal, errDay), "movement_id", fromUUID(e.MovementID).String())
		}
		id := fromUUID(e.MovementID)
		out[id] = append(out[id], domain.OccurrenceEdit{
			Original:    original,
			Exception:   domain.Exception{Deleted: e.Deleted, Date: day, Amount: fromBigint(e.Amount)},
			Description: fromText(e.Description),
			TagIDs:      fromUUIDs(row.TagIds),
		})
	}
	return out, nil
}

func (r movements) toMovement(ctx context.Context, row sqlcgen.Movement, tagIDs []pgtype.UUID) (domain.Movement, error) {
	m := domain.Movement{
		ID: fromUUID(row.ID), WorkspaceID: fromUUID(row.WorkspaceID), Description: fromText(row.Description),
		TagIDs: fromUUIDs(tagIDs), GroupID: fromUUID(row.SeriesGroupID),
	}
	day, err := fromDate(row.Date)
	if err != nil {
		return domain.Movement{}, r.fail(ctx, "read", fmt.Errorf("corrupt row: %w", err), "movement_id", m.ID.String())
	}
	if row.Frequency == nil {
		m.Entry = domain.Entry{Date: day, Type: domain.MovementType(row.Type), Amount: domain.Amount(row.Amount)}
		return m, nil
	}
	end, err := fromDate(row.EndDate)
	s := domain.Series{
		Type: domain.MovementType(row.Type), Frequency: domain.Frequency(*row.Frequency), Start: day, End: end,
		AnchorDay: fromSmallint(row.AnchorDay), Amount: domain.Amount(row.Amount), LastAmount: fromBigint(row.LastAmount),
		Parts: fromSmallint(row.Parts), FirstPart: fromSmallint(row.FirstPart),
	}
	if err == nil {
		err = s.Validate()
	}
	if err != nil {
		return domain.Movement{}, r.fail(ctx, "read", fmt.Errorf("corrupt series: %w", err), "movement_id", m.ID.String())
	}
	m.Series = &s
	return m, nil
}

func addTags(ctx context.Context, q *sqlcgen.Queries, userID, movementID domain.ID, tagIDs []domain.ID) error {
	if len(tagIDs) == 0 {
		return nil
	}
	return q.AddMovementTags(ctx, sqlcgen.AddMovementTagsParams{MovementID: toUUID(movementID), TagIds: toUUIDs(tagIDs), UserID: toUUID(userID)})
}
