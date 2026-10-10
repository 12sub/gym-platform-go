package jobs

import (
	"context"
	"log"
	"time"

	"github.com/12sub/gym-platform/internal/clock"
)

// RunDaily runs fn once right away (to catch up after a restart), then every
// day at hour:minute Lagos time, until ctx is cancelled.
func RunDaily(ctx context.Context, name string, hour, minute int, fn func(context.Context) error) {
	run := func() {
		if err := fn(ctx); err != nil {
			log.Printf("job %q failed: %v", name, err)
		}
	}
	run()

	for {
		timer := time.NewTimer(time.Until(nextRun(time.Now(), hour, minute)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			run()
		}
	}
}

func nextRun(now time.Time, hour, minute int) time.Time {
	n := now.In(clock.Lagos)
	t := time.Date(n.Year(), n.Month(), n.Day(), hour, minute, 0, 0, clock.Lagos)
	if !t.After(n) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}
