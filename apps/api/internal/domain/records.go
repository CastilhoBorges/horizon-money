package domain

// The records the use cases load and save. They carry ids and labels around the pure rules; the
// repositories in internal/port read and write them.

// User is a person with a login and their settings (§4.2).
type User struct {
	ID            ID
	Email         string
	DisplayName   string
	MainCurrency  Currency
	Timezone      string // IANA name; defines "today"
	SavingsTarget SavingsTarget
}

// Workspace is one balance in one currency (§4.3).
type Workspace struct {
	ID              ID
	Name            string
	Currency        Currency
	InitialBalance  Amount
	Start           Date // D0, the initial balance date
	LowThreshold    Amount
	ForecastDivisor int
	Archived        bool
}

// ForecastCategory is one line of a workspace's daily forecast (§4.7).
type ForecastCategory struct {
	ID            ID
	Name          string
	MonthlyAmount Amount
}
