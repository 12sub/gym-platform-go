package services

import (
	"errors"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestEndDate(t *testing.T) {
	if got := EndDate(d("2026-10-01"), 30); !got.Equal(d("2026-10-30")) {
		t.Errorf("30 days from 1 Oct ends %v", got)
	}
	if got := EndDate(d("2026-01-01"), 365); !got.Equal(d("2026-12-31")) {
		t.Errorf("365 days from 1 Jan ends %v", got)
	}
	if got := EndDate(d("2026-10-01"), 1); !got.Equal(d("2026-10-01")) {
		t.Errorf("a 1-day plan should end the same day, got %v", got)
	}
}

func TestNextStart(t *testing.T) {
	today := d("2026-10-10")

	cases := []struct {
		name        string
		current     []subState
		wantStart   string
		wantRenewal bool
		wantErr     error
	}{
		{"nothing current starts today", nil, "2026-10-10", false, nil},
		{"active renews the day after it ends", []subState{{"active", d("2026-10-20")}}, "2026-10-21", true, nil},
		{"active ending today renews tomorrow", []subState{{"active", d("2026-10-10")}}, "2026-10-11", true, nil},
		{"overdue active is ignored", []subState{{"active", d("2026-10-09")}}, "2026-10-10", false, nil},
		{"frozen blocks", []subState{{"frozen", d("2026-10-20")}}, "", false, ErrFrozen},
		{"queued renewal blocks",
			[]subState{{"active", d("2026-10-20")}, {"pending", d("2026-11-19")}}, "", false, ErrQueued},
	}

	for _, c := range cases {
		start, renewal, err := nextStart(c.current, today)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}
		if !start.Equal(d(c.wantStart)) || renewal != c.wantRenewal {
			t.Errorf("%s: got %v renewal=%v, want %s renewal=%v", c.name, start, renewal, c.wantStart, c.wantRenewal)
		}
	}
}