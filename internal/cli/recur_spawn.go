package cli

import (
	"fmt"
	"time"

	"github.com/VishalDalwadi/anchor/internal/cronutil"
	"github.com/VishalDalwadi/anchor/internal/idgen"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

// spawnResult summarizes one recur run.
type spawnResult struct {
	spawned int // new task instances created
	dropped int // unfinished expire instances dropped
}

// spawnRecurring creates task instances for every template due by today,
// updating tasks and templates in place. Each instance is due on the day
// its cron fired, pushed out by the template's due_in if set. Safe to run repeatedly: a template's last_spawned day is
// never spawned again.
//
// on_miss decides what a missed period means:
//   - expire: only today's instance can spawn (days recur run didn't run
//     are skipped), and any still-open earlier instance is dropped.
//   - persist: every fire day since last_spawned spawns its own instance,
//     so days recur run didn't run are caught up, and earlier instances
//     stay open alongside the new ones until each is done.
//
// A template that has never spawned starts from today either way; it
// doesn't backfill fire days from before it existed. Nothing spawns for a
// fire day after the template's until date. An ended template is simply
// skipped: tasks it already spawned are left exactly as they are.
func spawnRecurring(templates []model.RecurringTemplate, tasks []model.Task, today time.Time) ([]model.Task, spawnResult, error) {
	var res spawnResult
	todayStr := today.Format(validate.DateLayout)

	for i, r := range templates {
		sched, err := cronutil.Parse(r.Cron)
		if err != nil {
			return tasks, res, fmt.Errorf("template %s: %w", r.ID, err)
		}
		through, err := spawnWindowEnd(r, today)
		if err != nil {
			return tasks, res, err
		}

		var days []time.Time
		if r.Persists() && r.LastSpawned != "" {
			last, err := time.ParseInLocation(validate.DateLayout, r.LastSpawned, today.Location())
			if err != nil {
				return tasks, res, fmt.Errorf("template %s: bad last_spawned %q: %w", r.ID, r.LastSpawned, err)
			}
			days = cronutil.FireDays(sched, last, through)
		} else if r.LastSpawned != todayStr && !through.Before(startOfDay(today)) && cronutil.MatchesDay(sched, today) {
			days = []time.Time{today}
		}
		if len(days) == 0 {
			continue
		}

		if !r.Persists() {
			for j := range tasks {
				if tasks[j].RecurringSource == r.ID && tasks[j].Status == model.TaskOpen {
					tasks[j].Status = model.TaskDropped
					res.dropped++
				}
			}
		}
		for _, day := range days {
			due, err := instanceDue(r, day)
			if err != nil {
				return tasks, res, err
			}
			tasks = append(tasks, model.Task{
				ID:              idgen.New("t"),
				Text:            r.Text,
				Aspect:          r.Aspect,
				Created:         todayStr,
				Due:             due,
				Status:          model.TaskOpen,
				Context:         r.Context,
				RecurringSource: r.ID,
				Active:          true,
			})
			res.spawned++
		}
		templates[i].LastSpawned = days[len(days)-1].Format(validate.DateLayout)
	}
	return tasks, res, nil
}

// instanceDue is the due date of r's instance for fire day: the fire day
// itself, or offset by r's due_in.
func instanceDue(r model.RecurringTemplate, day time.Time) (string, error) {
	if r.DueIn == "" {
		return day.Format(validate.DateLayout), nil
	}
	due, err := validate.DueOffset(r.DueIn, day)
	if err != nil {
		return "", fmt.Errorf("template %s: bad due_in: %w", r.ID, err)
	}
	return due, nil
}

// owed describes an open, overdue instance of a persist template: an
// obligation that's still pending.
type owed struct {
	daysOverdue  int
	laterPeriods int // fire days since it was due, up to today (or the template's end)
}

// owedStatus reports whether t is still owed, and by how much. False for
// anything that isn't an open, overdue instance of a persist template.
func owedStatus(t model.Task, templates map[string]model.RecurringTemplate, today time.Time) (owed, bool) {
	if t.Status != model.TaskOpen || t.RecurringSource == "" || t.Due == "" {
		return owed{}, false
	}
	r, ok := templates[t.RecurringSource]
	if !ok || !r.Persists() {
		return owed{}, false
	}
	due, err := time.ParseInLocation(validate.DateLayout, t.Due, today.Location())
	if err != nil || !due.Before(startOfDay(today)) {
		return owed{}, false
	}

	o := owed{daysOverdue: daysBetween(due, today)}
	sched, err := cronutil.Parse(r.Cron)
	through, uerr := spawnWindowEnd(r, today)
	if err == nil && uerr == nil {
		o.laterPeriods = len(cronutil.FireDays(sched, due, through))
	}
	return o, true
}

// stillOwed is the list-view flag for a still-owed task; empty otherwise.
func stillOwed(t model.Task, templates map[string]model.RecurringTemplate, today time.Time) string {
	o, ok := owedStatus(t, templates, today)
	if !ok {
		return ""
	}
	label := fmt.Sprintf("[!] still owed: %d day(s) overdue", o.daysOverdue)
	if o.laterPeriods > 0 {
		label += fmt.Sprintf(", %d later period(s) already due", o.laterPeriods)
	}
	return label
}

// daysBetween counts calendar days from from's day to to's day (negative
// if to is earlier), independent of time of day and DST.
func daysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// spawnWindowEnd is the last moment r may spawn for as of today: today
// itself, or the end of r's until day if that comes first.
func spawnWindowEnd(r model.RecurringTemplate, today time.Time) (time.Time, error) {
	if r.Until == "" {
		return today, nil
	}
	until, err := time.ParseInLocation(validate.DateLayout, r.Until, today.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("template %s: bad until %q: %w", r.ID, r.Until, err)
	}
	if endOfUntil := until.AddDate(0, 0, 1).Add(-time.Second); endOfUntil.Before(today) {
		return endOfUntil, nil
	}
	return today, nil
}

// recurEnded reports whether r's until date has passed as of today.
func recurEnded(r model.RecurringTemplate, today time.Time) bool {
	return r.Until != "" && r.Until < today.Format(validate.DateLayout)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// templatesByID indexes templates for stillOwed lookups.
func templatesByID(templates []model.RecurringTemplate) map[string]model.RecurringTemplate {
	m := make(map[string]model.RecurringTemplate, len(templates))
	for _, r := range templates {
		m[r.ID] = r
	}
	return m
}
