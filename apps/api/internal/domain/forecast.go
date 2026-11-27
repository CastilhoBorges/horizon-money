package domain

// The forecast divisor a workspace accepts (§4.3). Its default (30) is a database column default.
const (
	MinForecastDivisor = 28
	MaxForecastDivisor = 31
)

// DailyValue is the expected daily spending applied to future days: the sum of the workspace's monthly
// forecast categories divided by its divisor, rounded half-up to the minor unit (§4.7).
func DailyValue(monthlyAmounts []Amount, divisor int) (Amount, error) {
	if divisor < MinForecastDivisor || divisor > MaxForecastDivisor {
		return 0, ErrForecastDivisorOutOfRange
	}
	var total Amount
	for _, a := range monthlyAmounts {
		if a <= 0 {
			return 0, ErrForecastAmountNotPositive
		}
		total += a
	}
	return Amount(divHalfUp(int64(total), int64(divisor))), nil
}
