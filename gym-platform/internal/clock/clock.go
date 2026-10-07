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
