// Package cronutil evaluates standard 5-field cron expressions against a
// single day, without needing a running scheduler/daemon.
package cronutil

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// Parse validates a standard 5-field cron expression, returning a clear
// error if it's malformed.
func Parse(expr string) (cron.Schedule, error) {
	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return sched, nil
}

// MatchesDay reports whether sched has a scheduled fire time falling on the
// given day (any time from 00:00:00 to 23:59:59).
func MatchesDay(sched cron.Schedule, day time.Time) bool {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	next := sched.Next(start.Add(-time.Second))
	return !next.After(start.Add(24*time.Hour - time.Second))
}
