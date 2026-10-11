package reports

import "time"

type Point struct {
	Label string
	Kobo  int64
}

// DailySeries returns one point per day from..to, using zero for days with no
// payments, so the chart's time axis has no gaps. totals is keyed "2006-01-02".
func DailySeries(from, to time.Time, totals map[string]int64) []Point {
	var out []Point
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		out = append(out, Point{Label: d.Format("02 Jan"), Kobo: totals[d.Format(Layout)]})
	}
	return out
}

// MonthlySeries returns one point per calendar month touched by from..to.
// totals is keyed "2006-01".
func MonthlySeries(from, to time.Time, totals map[string]int64) []Point {
	var out []Point
	last := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	for m := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(last); m = m.AddDate(0, 1, 0) {
		out = append(out, Point{Label: m.Format("Jan 2006"), Kobo: totals[m.Format("2006-01")]})
	}
	return out
}