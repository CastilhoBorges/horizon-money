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

// toUUIDs maps ids to a uuid[] parameter.
func toUUIDs(ids []domain.ID) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		out = append(out, toUUID(id))
	}
	return out
}

// fromUUIDs maps a uuid[] column to ids, nil when empty.
func fromUUIDs(us []pgtype.UUID) []domain.ID {
	if len(us) == 0 {
		return nil
	}
	out := make([]domain.ID, 0, len(us))
	for _, u := range us {
		out = append(out, fromUUID(u))
	}
	return out
}

// toNullUUID maps the zero ID to NULL.
func toNullUUID(id domain.ID) pgtype.UUID {
	if id.IsZero() {
		return pgtype.UUID{}
	}
	return toUUID(id)
}

// toSmallint maps 0 to NULL for the optional smallint columns (anchor day, parts, first part), whose
// values the domain and the schema keep within 1–360.
func toSmallint(n int) *int16 {
	if n == 0 {
		return nil
	}
	v := int16(n) //nolint:gosec // bounded to 1–360 by the domain and the schema checks
	return &v
}

func fromSmallint(n *int16) int {
	if n == nil {
		return 0
	}
	return int(*n)
}

// toBigint maps 0 to NULL for optional amounts.
func toBigint(a domain.Amount) *int64 {
	if a == 0 {
		return nil
	}
	v := int64(a)
	return &v
}

func fromBigint(n *int64) domain.Amount {
	if n == nil {
		return 0
	}
	return domain.Amount(*n)
}
