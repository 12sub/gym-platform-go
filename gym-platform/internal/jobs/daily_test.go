package jobs

import (
	"testing"
	"time"

	"github.com/12sub/gym-platform/internal/clock"
)

func TestNextRun(t *testing.T) {
	at := func(day, hour, min int) time.Time {
		return time.Date(2026, 10, day, hour, min, 0, 0, clock.Lagos)
	}
	if got := nextRun(at(10, 10, 0), 0, 5); !got.Equal(at(11, 0, 5)) {
		t.Errorf("after the time today, want tomorrow 00:05, got %v", got)
	}
	if got := nextRun(at(10, 0, 1), 0, 5); !got.Equal(at(10, 0, 5)) {
		t.Errorf("before the time today, want today 00:05, got %v", got)
	}
}