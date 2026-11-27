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

// Movements and series.
var (
	ErrAmountNotPositive       = &Error{Code: "movement.amount_not_positive"}
	ErrUnknownMovementType     = &Error{Code: "movement.type_unknown"}
	ErrUnknownFrequency        = &Error{Code: "series.frequency_unknown"}
	ErrEndBeforeStart          = &Error{Code: "series.end_before_start"}
	ErrInstallmentsOutOfRange  = &Error{Code: "series.installments_out_of_range"}
	ErrInstallmentPartTooSmall = &Error{Code: "series.installment_part_too_small"}
	ErrNotAnOccurrence         = &Error{Code: "series.not_an_occurrence"}
	ErrSplitAtFirstOccurrence  = &Error{Code: "series.split_at_first_occurrence"}
)

// Forecast and views.
var (
	ErrForecastDivisorOutOfRange = &Error{Code: "workspace.forecast_divisor_out_of_range"}
	ErrForecastAmountNotPositive = &Error{Code: "forecast.amount_not_positive"}
	ErrHorizonOutOfRange         = &Error{Code: "horizon.months_out_of_range"}
)
