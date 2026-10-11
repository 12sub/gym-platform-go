package reports

import (
	"net/url"
	"time"

	"github.com/12sub/gym-platform/internal/clock"
)

const Layout = "2006-01-02"

// Range is an inclusive span of Lagos calendar dates, plus how to group it.
type Range struct {
	From  time.Time // midnight UTC, date part only
	To    time.Time
	Group string // "day" or "month"
}

// ParseRange reads from/to/group from a query string, defaulting to "this month
// so far", and quietly corrects anything unreasonable: swapped dates, dates in
// the future, or spans longer than a year.
func ParseRange(today time.Time, v url.Values) Range {
	from := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := today

	if t, err := time.Parse(Layout, v.Get("from")); err == nil {
		from = t
	}
	if t, err := time.Parse(Layout, v.Get("to")); err == nil {
		to = t
	}
	if to.Before(from) {
		from, to = to, from
	}
	if to.After(today) {
		to = today
	}
	if from.After(to) {
		from = to
	}
	if to.Sub(from) > 365*24*time.Hour {
		from = to.AddDate(0, 0, -365)
	}

	days := int(to.Sub(from).Hours()/24) + 1
	group := v.Get("group")
	if group != "day" && group != "month" {
		group = "day"
		if days > 62 {
			group = "month"
		}
	}
	return Range{From: from, To: to, Group: group}
}

// Bounds converts the dates to the exact instants to query: Lagos midnight at
// the start of From up to (not including) Lagos midnight after To.
func (r Range) Bounds() (start, end time.Time) {
	start = time.Date(r.From.Year(), r.From.Month(), r.From.Day(), 0, 0, 0, 0, clock.Lagos)
	next := r.To.AddDate(0, 0, 1)
	end = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, clock.Lagos)
	return start, end
}

type Preset struct {
	Label string
	From  time.Time
	To    time.Time
}

func Presets(today time.Time) []Preset {
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthEnd := monthStart.AddDate(0, 0, -1)
	lastMonthStart := time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(today.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	return []Preset{
		{"Today", today, today},
		{"Last 7 days", today.AddDate(0, 0, -6), today},
		{"This month", monthStart, today},
		{"Last month", lastMonthStart, lastMonthEnd},
		{"This year", yearStart, today},
	}
}