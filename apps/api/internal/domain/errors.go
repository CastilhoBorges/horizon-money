package domain

// Error is a broken business rule. Its Code is stable and is what HTTP and MCP return; clients
// translate it into the user's language.
type Error struct {
	Code string
}

func (e *Error) Error() string { return e.Code }

// Currencies and amounts.
var (
	ErrUnknownCurrency = &Error{Code: "currency.unknown"}
	ErrInvalidAmount   = &Error{Code: "amount.invalid"}
	ErrTooManyDecimals = &Error{Code: "amount.too_many_decimals"}
	ErrAmountTooLarge  = &Error{Code: "amount.too_large"}
)

// Dates and months.
var (
	ErrInvalidDate  = &Error{Code: "date.invalid"}
	ErrInvalidMonth = &Error{Code: "month.invalid"}
)
