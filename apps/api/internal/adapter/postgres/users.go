package postgres

import (
	"context"
	"fmt"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/adapter/postgres/sqlcgen"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
	"github.com/CastilhoBorges/horizon-money/apps/api/internal/port"
)

type users struct{ repo }

func (r users) Create(ctx context.Context, u port.NewUser) (domain.User, error) {
	row, err := r.q.CreateUser(ctx, sqlcgen.CreateUserParams{Email: u.Email, DisplayName: u.DisplayName})
	if violates(err, uniqueViolation, "users_email_key") {
		return domain.User{}, domain.ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, r.fail(ctx, "create", err)
	}
	return r.toUser(ctx, row)
}

func (r users) Get(ctx context.Context, userID domain.ID) (domain.User, error) {
	row, err := r.q.GetUser(ctx, toUUID(userID))
	if isNoRows(err) {
		return domain.User{}, port.ErrNotFound
	}
	if err != nil {
		return domain.User{}, r.fail(ctx, "get", err, "user_id", userID.String())
	}
	return r.toUser(ctx, row)
}

func (r users) toUser(ctx context.Context, row sqlcgen.User) (domain.User, error) {
	currency, err := domain.ParseCurrency(row.MainCurrency)
	if err != nil {
		return domain.User{}, r.fail(ctx, "read", fmt.Errorf("main_currency: %w", err), "user_id", fromUUID(row.ID).String())
	}
	return domain.User{
		ID:            fromUUID(row.ID),
		Email:         row.Email,
		DisplayName:   row.DisplayName,
		MainCurrency:  currency,
		Timezone:      row.Timezone,
		SavingsTarget: domain.SavingsTarget{Min: domain.BasisPoints(row.SavingsRateTargetMin), Max: domain.BasisPoints(row.SavingsRateTargetMax)},
	}, nil
}
