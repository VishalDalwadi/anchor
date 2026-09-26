package cronutil

import (
	"fmt"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func dates(ts []time.Time) string {
	var out []string
	for _, t := range ts {
		out = append(out, t.Format("2006-01-02"))
	}
	return fmt.Sprint(out)
}

func TestFireDays(t *testing.T) {
	tests := []struct {
		name, cron, after, through, want string
	}{
		{"excludes after's day, includes through's", "0 9 15 3,6,9,12 *", "2026-06-15", "2026-12-15", "[2026-09-15 2026-12-15]"},
		{"crosses a year", "0 9 15 3,6,9,12 *", "2026-12-15", "2027-06-20", "[2027-03-15 2027-06-15]"},
		{"nothing due yet", "0 9 15 3,6,9,12 *", "2026-09-15", "2026-10-01", "[]"},
		{"daily", "30 6 * * *", "2026-09-24", "2026-09-27", "[2026-09-25 2026-09-26 2026-09-27]"},
		{"one entry per day", "0 9,18 * * *", "2026-09-25", "2026-09-26", "[2026-09-26]"},
	}
	for _, tt := range tests {
		sched, err := Parse(tt.cron)
		if err != nil {
			t.Fatal(err)
		}
		// through is mid-afternoon, like a real `recur run`.
		got := FireDays(sched, day(tt.after), day(tt.through).Add(15*time.Hour))
		if dates(got) != tt.want {
			t.Errorf("%s: FireDays = %s, want %s", tt.name, dates(got), tt.want)
		}
	}
}
