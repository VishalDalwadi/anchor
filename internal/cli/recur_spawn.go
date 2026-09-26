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
// its cron fired. Safe to run repeatedly: a template's last_spawned day is
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
			tasks = append(tasks, model.Task{
				ID:              idgen.New("t"),
				Text:            r.Text,
				Aspect:          r.Aspect,
				Created:         todayStr,
				Due:             day.Format(validate.DateLayout),
				Status:          model.TaskOpen,
				Context:         r.Context,
				RecurringSource: r.ID,
			})
			res.spawned++
		}
		templates[i].LastSpawned = days[len(days)-1].Format(validate.DateLayout)
	}
	return tasks, res, nil
}

// stillOwed describes an open, overdue instance of a persist template —
// an obligation that's still pending — for list views. Empty for any
// other task.
func stillOwed(t model.Task, templates map[string]model.RecurringTemplate, today time.Time) string {
	if t.Status != model.TaskOpen || t.RecurringSource == "" || t.Due == "" {
		return ""
	}
	r, ok := templates[t.RecurringSource]
	if !ok || !r.Persists() {
		return ""
	}
	due, err := time.ParseInLocation(validate.DateLayout, t.Due, today.Location())
	if err != nil || !due.Before(startOfDay(today)) {
		return ""
	}

	days := int(startOfDay(today).Sub(due).Hours()/24 + 0.5)
	label := fmt.Sprintf("[!] still owed: %d day(s) overdue", days)
	sched, err := cronutil.Parse(r.Cron)
	through, uerr := spawnWindowEnd(r, today)
	if err == nil && uerr == nil {
		if n := len(cronutil.FireDays(sched, due, through)); n > 0 {
			label += fmt.Sprintf(", %d later period(s) already due", n)
		}
	}
	return label
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
