package reports

import (
	"net/url"
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse(Layout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseRangeDefaultsToThisMonth(t *testing.T) {
	r := ParseRange(date("2026-10-10"), url.Values{})
	if !r.From.Equal(date("2026-10-01")) || !r.To.Equal(date("2026-10-10")) || r.Group != "day" {
		t.Errorf("got %v..%v %s", r.From, r.To, r.Group)
	}
}

func TestParseRangeCorrectsInput(t *testing.T) {
	today := date("2026-10-10")

	swapped := ParseRange(today, url.Values{"from": {"2026-10-05"}, "to": {"2026-10-01"}})
	if !swapped.From.Equal(date("2026-10-01")) || !swapped.To.Equal(date("2026-10-05")) {
		t.Errorf("swapped dates not fixed: %v..%v", swapped.From, swapped.To)
	}

	future := ParseRange(today, url.Values{"from": {"2026-10-05"}, "to": {"2027-01-01"}})
	if !future.To.Equal(today) {
		t.Errorf("future end date not clamped: %v", future.To)
	}

	allFuture := ParseRange(today, url.Values{"from": {"2027-01-01"}, "to": {"2027-02-01"}})
	if allFuture.From.After(allFuture.To) {
		t.Errorf("from is after to: %v..%v", allFuture.From, allFuture.To)
	}

	long := ParseRange(today, url.Values{"from": {"2020-01-01"}, "to": {"2026-10-10"}})
	if long.To.Sub(long.From) > 365*24*time.Hour {
		t.Errorf("range longer than a year: %v..%v", long.From, long.To)
	}

	junk := ParseRange(today, url.Values{"from": {"not-a-date"}})
	if !junk.From.Equal(date("2026-10-01")) {
		t.Errorf("junk input should fall back to the default, got %v", junk.From)
	}
}

func TestParseRangeGrouping(t *testing.T) {
	today := date("2026-10-10")
	if r := ParseRange(today, url.Values{"from": {"2026-01-01"}}); r.Group != "month" {
		t.Errorf("a long range should default to months, got %s", r.Group)
	}
	if r := ParseRange(today, url.Values{"from": {"2026-01-01"}, "group": {"day"}}); r.Group != "day" {
		t.Errorf("an explicit group should win, got %s", r.Group)
	}
}

func TestDailySeriesFillsGaps(t *testing.T) {
	got := DailySeries(date("2026-10-01"), date("2026-10-04"), map[string]int64{"2026-10-02": 500, "2026-10-04": 900})
	want := []int64{0, 500, 0, 900}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Kobo != w {
			t.Errorf("point %d = %d, want %d", i, got[i].Kobo, w)
		}
	}
	if got[0].Label != "01 Oct" {
		t.Errorf("label = %q", got[0].Label)
	}
}

func TestMonthlySeriesAcrossYears(t *testing.T) {
	got := MonthlySeries(date("2026-11-15"), date("2027-02-03"), map[string]int64{"2026-12": 700})
	if len(got) != 4 {
		t.Fatalf("got %d months, want 4 (Nov, Dec, Jan, Feb)", len(got))
	}
	if got[0].Label != "Nov 2026" || got[3].Label != "Feb 2027" || got[1].Kobo != 700 {
		t.Errorf("unexpected series: %+v", got)
	}
}

func TestPresets(t *testing.T) {
	p := Presets(date("2026-10-10"))
	if len(p) != 5 {
		t.Fatalf("got %d presets", len(p))
	}
	if !p[3].From.Equal(date("2026-09-01")) || !p[3].To.Equal(date("2026-09-30")) {
		t.Errorf("last month = %v..%v", p[3].From, p[3].To)
	}
	if !p[1].From.Equal(date("2026-10-04")) {
		t.Errorf("last 7 days should start on 4 Oct, got %v", p[1].From)
	}
}