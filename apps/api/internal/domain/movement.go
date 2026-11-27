package domain

// MovementType is what a movement is. The type gives the sign: income adds to the balance and every
// other type subtracts from it.
type MovementType string

// The five movement types (§4.5).
const (
	TypeIncome  MovementType = "income"
	TypeExpense MovementType = "expense"
	TypeDaily   MovementType = "daily"
	TypeSaving  MovementType = "saving"
	TypeCard    MovementType = "card"
)

func (t MovementType) isValid() bool {
	switch t {
	case TypeIncome, TypeExpense, TypeDaily, TypeSaving, TypeCard:
		return true
	}
	return false
}
