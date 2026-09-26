package validate

import (
	"strings"
	"testing"
	"time"
)

func TestFlexDate(t *testing.T) {
	// A Wednesday afternoon in September, so month-name tests have months
	// on both sides of "now".
	now := time.Date(2026, time.September, 16, 15, 0, 0, 0, time.Local)

	tests := []struct {
		in, want string
	}{
		{"", ""},

		// Go durations.
		{"72h", "2026-09-19"},
		{"8h", "2026-09-16"},
		{"9h", "2026-09-17"}, // crosses midnight
		{"1h30m", "2026-09-16"},
		{"48H", "2026-09-18"}, // case-insensitive

		// Shorthand: a bare <N>m is months, never minutes.
		{"0d", "2026-09-16"},
		{"2d", "2026-09-18"},
		{"20d", "2026-10-06"},
		{"1w", "2026-09-23"},
		{"1m", "2026-10-16"},
		{"30m", "2029-03-16"},
		{"4m", "2027-01-16"}, // crosses a year

		// Month names: last day, this year unless already past.
		{"september", "2026-09-30"}, // current month hasn't passed
		{"December", "2026-12-31"},
		{"feb", "2027-02-28"}, // passed → next year
		{"AUG", "2027-08-31"},

		// Bare year.
		{"2026", "2026-12-31"},
		{"2030", "2030-12-31"},

		// Strict date.
		{"2026-08-31", "2026-08-31"},
		{" 2026-08-31 ", "2026-08-31"},
	}
	for _, tt := range tests {
		got, err := FlexDate(tt.in, now)
		if err != nil {
			t.Errorf("FlexDate(%q): unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("FlexDate(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFlexDateMonthsClampToMonthEnd(t *testing.T) {
	tests := []struct {
		now  time.Time
		in   string
		want string
	}{
		{time.Date(2026, time.January, 31, 9, 0, 0, 0, time.Local), "1m", "2026-02-28"},
		{time.Date(2028, time.January, 31, 9, 0, 0, 0, time.Local), "1m", "2028-02-29"}, // leap year
		{time.Date(2026, time.August, 31, 9, 0, 0, 0, time.Local), "1m", "2026-09-30"},
		{time.Date(2026, time.December, 31, 9, 0, 0, 0, time.Local), "2m", "2027-02-28"},
	}
	for _, tt := range tests {
		got, err := FlexDate(tt.in, tt.now)
		if err != nil {
			t.Errorf("FlexDate(%q) from %s: %v", tt.in, tt.now.Format(DateLayout), err)
			continue
		}
		if got != tt.want {
			t.Errorf("FlexDate(%q) from %s = %q, want %q", tt.in, tt.now.Format(DateLayout), got, tt.want)
		}
	}
}

func TestFlexDateRejectsGarbage(t *testing.T) {
	now := time.Date(2026, time.September, 16, 15, 0, 0, 0, time.Local)
	for _, in := range []string{"tomorrow", "2d3", "d2", "1y", "26", "2026-13-01", "2026/08/31", "septembre"} {
		_, err := FlexDate(in, now)
		if err == nil {
			t.Errorf("FlexDate(%q): expected an error", in)
			continue
		}
		if !strings.Contains(err.Error(), "accepted forms") {
			t.Errorf("FlexDate(%q) error does not list accepted forms: %v", in, err)
		}
	}
}
