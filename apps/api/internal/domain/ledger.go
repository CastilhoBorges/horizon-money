package domain

// The horizon's length in months (§4.9): the current month and the next two by default.
const (
	DefaultHorizonMonths = 3
	MaxHorizonMonths     = 12
)

// Entry is one movement that counts on a date: a one-off movement or an expanded occurrence of a series.
type Entry struct {
	Date   Date
	Type   MovementType
	Amount Amount // > 0; the type gives the sign
}

// TransferLeg is one workspace's side of a transfer, in that workspace's currency.
type TransferLeg struct {
	Date     Date
	Amount   Amount // > 0
	Incoming bool   // true when the money arrives in this workspace
}

// Totals are the amounts of each movement type.
type Totals struct {
	Income, Expense, Daily, Saving, Card Amount
}

func (t *Totals) add(kind MovementType, a Amount) {
	switch kind {
	case TypeIncome:
		t.Income += a
	case TypeExpense:
		t.Expense += a
	case TypeDaily:
		t.Daily += a
	case TypeSaving:
		t.Saving += a
	case TypeCard:
		t.Card += a
	}
}

// net is what the movements do to the balance: income in, everything else out.
func (t Totals) net() Amount { return t.Income - t.Expense - t.Daily - t.Saving - t.Card }

// Status flags a day's end balance.
type Status string

// The day statuses (§4.9). A day is low when its balance is at least 0 and below the workspace's
// low-balance threshold, so the default threshold of 0 never flags one.
const (
	StatusOK       Status = "ok"
	StatusLow      Status = "low"
	StatusNegative Status = "negative"
)

// Day is one row of the balance table.
type Day struct {
	Date         Date
	Totals       Totals
	TransfersIn  Amount
	TransfersOut Amount
	Forecast     Amount
	End          Amount
	IsWeekend    bool
	Status       Status
}

// Ledger is one workspace as the domain reads it: its settings plus every entry and transfer leg that
// counts. Use cases build it from one-off movements, expanded series and transfers.
type Ledger struct {
	// Start is D0, the first day the workspace tracks.
	Start Date
	// InitialBalance is the balance at the start of D0.
	InitialBalance Amount
	// DailyForecast is the daily value (DailyValue) applied to days after today.
	DailyForecast Amount
	// LowThreshold is the workspace's low-balance threshold.
	LowThreshold Amount
	Entries      []Entry
	Transfers    []TransferLeg
}

// flows are what moves the balance on one date.
type flows struct {
	totals  Totals
	in, out Amount
}

// Days is the balance table between the from and to dates, inclusive (§4.9). It never starts before D0, and entries and
// legs dated before D0 are ignored. The forecast applies to days after today, from D0 on.
func (l Ledger) Days(from, to, today Date) []Day {
	if from.Before(l.Start) {
		from = l.Start
	}
	if to.Before(from) {
		return nil
	}
	byDate := l.flowsByDate(to)

	out := make([]Day, 0, to.DaysSince(from)+1)
	balance := l.InitialBalance
	for d := l.Start; !d.After(to); d = d.AddDays(1) {
		f := byDate[d]
		forecast := l.forecastOn(d, today)
		balance += f.totals.net() + f.in - f.out - forecast
		if d.Before(from) {
			continue
		}
		out = append(out, Day{
			Date: d, Totals: f.totals, TransfersIn: f.in, TransfersOut: f.out, Forecast: forecast,
			End: balance, IsWeekend: d.IsWeekend(), Status: l.status(balance),
		})
	}
	return out
}

// Horizon is the balance table over the given number of months, starting with first (§4.9).
func (l Ledger) Horizon(first Month, months int, today Date) ([]Day, error) {
	if months < 1 || months > MaxHorizonMonths {
		return nil, ErrHorizonOutOfRange
	}
	return l.Days(first.First(), first.Add(months-1).Last(), today), nil
}

// flowsByDate groups the entries and legs dated from D0 through to.
func (l Ledger) flowsByDate(to Date) map[Date]flows {
	byDate := make(map[Date]flows)
	for _, e := range l.Entries {
		if inRange(e.Date, l.Start, to) {
			f := byDate[e.Date]
			f.totals.add(e.Type, e.Amount)
			byDate[e.Date] = f
		}
	}
	for _, leg := range l.Transfers {
		if inRange(leg.Date, l.Start, to) {
			f := byDate[leg.Date]
			if leg.Incoming {
				f.in += leg.Amount
			} else {
				f.out += leg.Amount
			}
			byDate[leg.Date] = f
		}
	}
	return byDate
}

// forecastOn is the forecast that counts on d: the daily value after today and from D0 on, else zero.
// At the user's midnight the day becomes today and its forecast disappears.
func (l Ledger) forecastOn(d, today Date) Amount {
	if d.After(today) && !d.Before(l.Start) {
		return l.DailyForecast
	}
	return 0
}

func (l Ledger) status(balance Amount) Status {
	switch {
	case balance < 0:
		return StatusNegative
	case balance < l.LowThreshold:
		return StatusLow
	default:
		return StatusOK
	}
}
