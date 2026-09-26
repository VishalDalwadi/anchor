package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/config"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

// defaultLookahead is how many days ahead brief's reminders look.
const defaultLookahead = 3

// newBriefCmd prints today's briefing — reminders about what's due or
// overdue, open tasks, active goals, and stale watchlist items — and also
// writes it to log/YYYY-MM-DD.md (used when this is invoked by a Claude
// Cowork scheduled task running server-side, where there's no terminal to
// read stdout from). log/ is output-only and never read back in.
func newBriefCmd() *cobra.Command {
	var lookahead int
	cmd := &cobra.Command{
		Use:   "brief",
		Short: "Generate today's briefing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			lookahead, err := briefLookahead(cmd, lookahead)
			if err != nil {
				return err
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			tasks, err := s.LoadTasks()
			if err != nil {
				return err
			}
			goals, err := s.LoadGoals()
			if err != nil {
				return err
			}
			watchItems, err := s.LoadWatchItems()
			if err != nil {
				return err
			}
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}

			now := time.Now()
			brief := buildBrief(briefData{tasks, goals, watchItems, templatesByID(templates)}, now, lookahead)
			if err := s.WriteLog(now.Format(validate.DateLayout), brief); err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), brief)
			return nil
		},
	}
	cmd.Flags().IntVar(&lookahead, "lookahead", defaultLookahead,
		"remind about tasks due within this many days (default: `anchor config set lookahead`, else 3)")
	return cmd
}

// briefLookahead picks the reminder window: --lookahead if given, else
// the configured lookahead, else the default.
func briefLookahead(cmd *cobra.Command, flag int) (int, error) {
	if cmd.Flags().Changed("lookahead") {
		if flag < 0 {
			return 0, fmt.Errorf("--lookahead must be 0 or more, not %d", flag)
		}
		return flag, nil
	}
	c, err := config.Load()
	if err != nil {
		return 0, err
	}
	if c.Lookahead != nil {
		return *c.Lookahead, nil
	}
	return defaultLookahead, nil
}

type briefData struct {
	tasks      []model.Task
	goals      []model.Goal
	watchItems []model.WatchItem
	templates  map[string]model.RecurringTemplate
}

func buildBrief(d briefData, now time.Time, lookahead int) string {
	todayStr := now.Format(validate.DateLayout)
	var b strings.Builder
	fmt.Fprintf(&b, "# Briefing — %s\n\n", todayStr)

	writeReminders(&b, d, now, lookahead)

	section(&b, "Open tasks (no due date)", "_Nothing without a due date._", func(add func(string)) {
		for _, t := range d.tasks {
			if t.Status == model.TaskOpen && t.Active && t.Due == "" {
				add(fmt.Sprintf("- [%s] %s (%s)", t.ID, t.Text, t.Aspect))
			}
		}
	})

	horizon := now.AddDate(0, 0, lookahead).Format(validate.DateLayout)
	section(&b, "Later due dates", "_Nothing further out._", func(add func(string)) {
		for _, t := range sortedByDue(d.tasks) {
			if t.Status == model.TaskOpen && t.Due > horizon {
				add(fmt.Sprintf("- [%s] %s (%s) — due %s", t.ID, t.Text, t.Aspect, t.Due))
			}
		}
	})

	section(&b, "Active goals", "_No active goals._", func(add func(string)) {
		for _, g := range d.goals {
			if g.Status == model.GoalActive {
				add(fmt.Sprintf("- [%s] (%s/%s) %s", g.ID, g.Aspect, g.Tier, g.Text))
			}
		}
	})

	section(&b, "Stale watchlist items", "_Nothing stale._", func(add func(string)) {
		for _, w := range d.watchItems {
			if w.Status != model.WatchResolved && isStale(w) {
				add(fmt.Sprintf("- [%s] %s (%s) — last checked %s", w.ID, w.Text, w.Aspect, w.LastChecked))
			}
		}
	})
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// writeReminders writes the Reminders section: every still-owed recurring
// task (always, however overdue), other overdue tasks, then tasks due
// today, tomorrow, and each day up to lookahead days out. Backlogged tasks
// are included: a deadline applies whether or not a task is in focus.
func writeReminders(b *strings.Builder, d briefData, now time.Time, lookahead int) {
	type group struct {
		title string
		lines []string
	}
	var owedLines, overdueLines []string
	dueOn := make([][]string, lookahead+1) // dueOn[n]: due n days from today

	for _, t := range sortedByDue(d.tasks) {
		if t.Status != model.TaskOpen || t.Due == "" {
			continue
		}
		line := fmt.Sprintf("- [%s] %s (%s)", t.ID, t.Text, t.Aspect)
		if o, ok := owedStatus(t, d.templates, now); ok {
			line += fmt.Sprintf(" — due %s, %s overdue", t.Due, plural(o.daysOverdue, "day"))
			if o.laterPeriods > 0 {
				line += fmt.Sprintf(", %s already due", plural(o.laterPeriods, "later period"))
			}
			owedLines = append(owedLines, line)
			continue
		}
		due, err := time.ParseInLocation(validate.DateLayout, t.Due, now.Location())
		if err != nil {
			continue
		}
		switch n := daysBetween(now, due); {
		case n < 0:
			overdueLines = append(overdueLines, line+fmt.Sprintf(" — due %s, %s overdue", t.Due, plural(-n, "day")))
		case n <= lookahead:
			dueOn[n] = append(dueOn[n], line)
		}
	}

	groups := []group{
		{"Still owed (recurring, overdue)", owedLines},
		{"Overdue", overdueLines},
	}
	for n, lines := range dueOn {
		title := fmt.Sprintf("Due in %d days (%s)", n, now.AddDate(0, 0, n).Format(validate.DateLayout))
		switch n {
		case 0:
			title = "Due today"
		case 1:
			title = "Due tomorrow"
		}
		groups = append(groups, group{title, lines})
	}

	b.WriteString("## Reminders\n\n")
	any := false
	for _, g := range groups {
		if len(g.lines) == 0 {
			continue
		}
		any = true
		fmt.Fprintf(b, "### %s\n\n%s\n\n", g.title, strings.Join(g.lines, "\n"))
	}
	if !any {
		fmt.Fprintf(b, "_Nothing overdue, and nothing due in the next %s._\n\n", plural(lookahead, "day"))
	}
}

// section writes a "## title" section with the lines fill adds, or the
// empty placeholder if it adds none.
func section(b *strings.Builder, title, empty string, fill func(add func(string))) {
	var lines []string
	fill(func(l string) { lines = append(lines, l) })
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(lines) == 0 {
		b.WriteString(empty + "\n\n")
		return
	}
	b.WriteString(strings.Join(lines, "\n") + "\n\n")
}

// sortedByDue returns tasks ordered by due date (undated last), keeping
// file order among equal dates.
func sortedByDue(tasks []model.Task) []model.Task {
	out := append([]model.Task(nil), tasks...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Due, out[j].Due
		if a == "" || b == "" {
			return a != "" && b == ""
		}
		return a < b
	})
	return out
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
