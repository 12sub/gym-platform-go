package clock

import "time"

// Lagos is the gym's local time. Lagos has no daylight saving, so a fixed
// offset is a safe fallback if the system has no timezone database.
var Lagos *time.Location

func init() {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		loc = time.FixedZone("WAT", 3600)
	}
	Lagos = loc
}

// Today is the current calendar date in Lagos, as midnight UTC (only the date matters).
func Today() time.Time {
	n := time.Now().In(Lagos)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// DayBounds is today in Lagos as a [start, end) range of instants, for querying timestamps.
func DayBounds() (time.Time, time.Time) {
	n := time.Now().In(Lagos)
	start := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, Lagos)
	return start, start.AddDate(0, 0, 1)
}

// MonthBounds is the current month in Lagos as a [start, end) range of instants.
func MonthBounds() (time.Time, time.Time) {
	n := time.Now().In(Lagos)
	start := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, Lagos)
	return start, start.AddDate(0, 1, 0)
}
