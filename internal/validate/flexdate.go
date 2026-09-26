package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// flexDateForms is shown when --due input matches none of the accepted forms.
const flexDateForms = `accepted forms:
  72h, 1h30m     Go duration from now (units h, m, s, ms, us, ns)
  2d, 1w, 1m     days, weeks, or months from today (a bare <N>m is months, not minutes)
  august, aug    last day of that month (this year, or next year if it has passed)
  2026           December 31 of that year
  2026-08-31     exact date (YYYY-MM-DD)`

var (
	shorthandRe = regexp.MustCompile(`^(\d+)([dwm])$`)
	yearRe      = regexp.MustCompile(`^\d{4}$`)
)

var months = map[string]time.Month{}

func init() {
	for m := time.January; m <= time.December; m++ {
		name := strings.ToLower(m.String())
		months[name] = m
		months[name[:3]] = m
	}
}

// FlexDate parses flexible date-flag input (--due, --expected) into a YYYY-MM-DD date, relative to now.
// Forms are tried in order: Go duration, <N>d/<N>w/<N>m shorthand, bare
// month name, bare year, then strict YYYY-MM-DD. A bare <N>m is always
// months, never Go's minutes — minutes are meaningless for a due date,
// and "1m" reading as "next month" is what's meant in practice. Durations
// that merely end in m (e.g. "1h30m") are still Go durations. Empty input
// returns "" (callers decide whether the field is required).
func FlexDate(s string, now time.Time) (string, error) {
	in := strings.ToLower(strings.TrimSpace(s))
	if in == "" {
		return "", nil
	}

	bareMonths := shorthandRe.MatchString(in) && strings.HasSuffix(in, "m")
	if !bareMonths {
		if d, err := time.ParseDuration(in); err == nil {
			return now.Add(d).Format(DateLayout), nil
		}
	}

	if m := shorthandRe.FindStringSubmatch(in); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return "", fmt.Errorf("invalid date %q: %w", s, err)
		}
		switch m[2] {
		case "d":
			return now.AddDate(0, 0, n).Format(DateLayout), nil
		case "w":
			return now.AddDate(0, 0, 7*n).Format(DateLayout), nil
		case "m":
			return addMonthsClamped(now, n).Format(DateLayout), nil
		}
	}

	if month, ok := months[in]; ok {
		year := now.Year()
		if month < now.Month() {
			year++
		}
		return lastDayOfMonth(year, month).Format(DateLayout), nil
	}

	if yearRe.MatchString(in) {
		year, _ := strconv.Atoi(in)
		return time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC).Format(DateLayout), nil
	}

	if t, err := time.Parse(DateLayout, in); err == nil {
		return t.Format(DateLayout), nil
	}

	return "", fmt.Errorf("invalid date %q\n%s", s, flexDateForms)
}

// addMonthsClamped adds n calendar months, clamping to the last day of the
// target month instead of overflowing into the next one (Jan 31 + 1m is
// Feb 28/29, not Mar 3 as time.AddDate would give).
func addMonthsClamped(t time.Time, n int) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, n, 0)
	last := lastDayOfMonth(first.Year(), first.Month())
	if t.Day() > last.Day() {
		return last
	}
	return time.Date(first.Year(), first.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func lastDayOfMonth(year int, month time.Month) time.Time {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)
}
