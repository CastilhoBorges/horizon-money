package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/CastilhoBorges/horizon-money/apps/api/internal/domain"
)

func toUUID(id domain.ID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }

func fromUUID(u pgtype.UUID) domain.ID {
	if !u.Valid {
		return domain.ID{}
	}
	return domain.ID(u.Bytes)
}

// toDate maps the zero Date ("no date") to NULL.
func toDate(d domain.Date) pgtype.Date {
	if d.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

// fromDate maps NULL to the zero Date.
func fromDate(d pgtype.Date) (domain.Date, error) {
	if !d.Valid {
		return domain.Date{}, nil
	}
	date, err := domain.NewDate(d.Time.Year(), d.Time.Month(), d.Time.Day())
	if err != nil {
		return domain.Date{}, fmt.Errorf("date %v: %w", d.Time, err)
	}
	return date, nil
}

// toText maps "" to NULL, for optional text such as descriptions.
func toText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func fromText(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
