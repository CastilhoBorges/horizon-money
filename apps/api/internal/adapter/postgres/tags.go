package postgres

import (
	"context"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

type tags struct{ repo }

func (r tags) Create(ctx context.Context, userID domain.ID, t port.NewTag) (domain.Tag, error) {
	row, err := r.q.CreateTag(ctx, sqlcgen.CreateTagParams{UserID: toUUID(userID), Name: t.Name, Color: string(t.Color)})
	if violates(err, uniqueViolation, "tags_user_id_name_key") {
		return domain.Tag{}, domain.ErrTagNameTaken
	}
	if err != nil {
		return domain.Tag{}, r.fail(ctx, "create", err, "user_id", userID.String())
	}
	return toTag(row), nil
}

func (r tags) List(ctx context.Context, userID domain.ID) ([]domain.Tag, error) {
	rows, err := r.q.ListTags(ctx, toUUID(userID))
	if err != nil {
		return nil, r.fail(ctx, "list", err, "user_id", userID.String())
	}
	out := make([]domain.Tag, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTag(row))
	}
	return out, nil
}

func toTag(row sqlcgen.Tag) domain.Tag {
	return domain.Tag{ID: fromUUID(row.ID), Name: row.Name, Color: domain.TagColor(row.Color)}
}
