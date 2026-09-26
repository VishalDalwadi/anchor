package cli

import (
	"strings"
	"testing"

	"github.com/VishalDalwadi/anchor/internal/model"
)

func briefTask(id, text, due string) model.Task {
	return model.Task{ID: id, Text: text, Aspect: "career", Due: due, Status: model.TaskOpen, Active: true}
}

func TestBriefReminders(t *testing.T) {
	templates := templatesByID([]model.RecurringTemplate{
		{ID: "r_tax", Cron: "0 9 15 6,9,12,3 *", OnMiss: model.OnMissPersist},
	})
	june := briefTask("t_june", "advance tax", "2026-06-15")
	june.RecurringSource = "r_tax"
	sept := briefTask("t_sept", "advance tax", "2026-09-15")
	sept.RecurringSource = "r_tax"
	backlogged := briefTask("t_visa", "renew visa", "2026-09-27")
	backlogged.Active = false

	d := briefData{
		tasks: []model.Task{
			briefTask("t_far", "plan trip", "2026-12-01"),
			briefTask("t_plus3", "book CA call", "2026-09-29"),
			sept,
			briefTask("t_today", "pay rent", "2026-09-26"),
			june, // listed out of order on purpose: owed are sorted by due
			briefTask("t_late", "reply to landlord", "2026-09-24"),
			backlogged,
			briefTask("t_nodue", "read paper", ""),
			{ID: "t_done", Text: "done already", Due: "2026-09-26", Status: model.TaskDone, Active: true},
		},
		templates: templates,
	}

	got := buildBrief(d, at("2026-09-26"), 3)
	want := `# Briefing — 2026-09-26

## Reminders

### Still owed (recurring, overdue)

- [t_june] advance tax (career) — due 2026-06-15, 103 days overdue, 1 later period already due
- [t_sept] advance tax (career) — due 2026-09-15, 11 days overdue

### Overdue

- [t_late] reply to landlord (career) — due 2026-09-24, 2 days overdue

### Due today

- [t_today] pay rent (career)

### Due tomorrow

- [t_visa] renew visa (career)

### Due in 3 days (2026-09-29)

- [t_plus3] book CA call (career)

## Open tasks (no due date)

- [t_nodue] read paper (career)

## Later due dates

- [t_far] plan trip (career) — due 2026-12-01

## Active goals

_No active goals._

## Stale watchlist items

_Nothing stale._
`
	if got != want {
		t.Errorf("brief:\n%s\nwant:\n%s", got, want)
	}
}

func TestBriefLookahead(t *testing.T) {
	d := briefData{tasks: []model.Task{briefTask("t_plus2", "x", "2026-09-28")}}

	// Within a 3-day window: a reminder, not a later date.
	if got := buildBrief(d, at("2026-09-26"), 3); !strings.Contains(got, "### Due in 2 days (2026-09-28)") {
		t.Errorf("lookahead 3: no reminder for a task due in 2 days:\n%s", got)
	}
	// With --lookahead 1 it's a later due date instead.
	got := buildBrief(d, at("2026-09-26"), 1)
	if strings.Contains(got, "### Due in") || !strings.Contains(got, "## Later due dates\n\n- [t_plus2]") {
		t.Errorf("lookahead 1: task due in 2 days should be under Later due dates:\n%s", got)
	}
	if !strings.Contains(got, "_Nothing overdue, and nothing due in the next 1 day._") {
		t.Errorf("lookahead 1: missing the empty-reminders note:\n%s", got)
	}
}

func TestDaysBetween(t *testing.T) {
	if n := daysBetween(at("2026-09-26"), at("2026-10-01")); n != 5 {
		t.Errorf("Sep 26 → Oct 1 = %d days, want 5", n)
	}
	if n := daysBetween(at("2026-09-26"), at("2026-09-24")); n != -2 {
		t.Errorf("Sep 26 → Sep 24 = %d days, want -2", n)
	}
}
