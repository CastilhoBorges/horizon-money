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

// TagColor is one of the eight tag colours (§4.4).
type TagColor string

// The tag colours.
const (
	TagGray   TagColor = "gray"
	TagBlue   TagColor = "blue"
	TagYellow TagColor = "yellow"
	TagGreen  TagColor = "green"
	TagRed    TagColor = "red"
	TagPurple TagColor = "purple"
	TagPink   TagColor = "pink"
	TagBrown  TagColor = "brown"
)

// Tag is a user's label for movements. It never changes a calculation (§4.4).
type Tag struct {
	ID    ID
	Name  string
	Color TagColor
}

// Transfer moves money between two workspaces of the same user, in one row with both sides (§4.6).
type Transfer struct {
	ID              ID
	FromWorkspaceID ID
	ToWorkspaceID   ID
	AmountSent      Amount // in the source workspace's currency
	AmountReceived  Amount // in the destination workspace's currency
	Date            Date
	Description     string
}
