package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

// newBriefCmd prints today's briefing — a human-readable summary of
// today's open tasks, upcoming due dates, active goals, and stale
// watchlist items — and also writes it to log/YYYY-MM-DD.md (used when
// this is invoked by a Claude Cowork scheduled task running server-side,
// where there's no terminal to read stdout from). log/ is output-only
// and never read back in.
func newBriefCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "brief",
		Short: "Generate today's briefing",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			todayStr := validate.Today()
			var b strings.Builder

			fmt.Fprintf(&b, "# Briefing — %s\n\n", todayStr)

			b.WriteString("## Today's open tasks\n\n")
			any := false
			for _, t := range tasks {
				if t.Status != model.TaskOpen {
					continue
				}
				if t.Due != "" && t.Due != todayStr {
					continue
				}
				fmt.Fprintf(&b, "- [%s] %s (%s)\n", t.ID, t.Text, t.Aspect)
				any = true
			}
			if !any {
				b.WriteString("_Nothing due today._\n")
			}
			b.WriteString("\n")

			b.WriteString("## Upcoming due dates\n\n")
			any = false
			for _, t := range tasks {
				if t.Status != model.TaskOpen || t.Due == "" || t.Due <= todayStr {
					continue
				}
				fmt.Fprintf(&b, "- [%s] %s (%s) — due %s\n", t.ID, t.Text, t.Aspect, t.Due)
				any = true
			}
			if !any {
				b.WriteString("_Nothing upcoming._\n")
			}
			b.WriteString("\n")

			b.WriteString("## Active goals\n\n")
			any = false
			for _, g := range goals {
				if g.Status != model.GoalActive {
					continue
				}
				fmt.Fprintf(&b, "- [%s] (%s/%s) %s\n", g.ID, g.Aspect, g.Tier, g.Text)
				any = true
			}
			if !any {
				b.WriteString("_No active goals._\n")
			}
			b.WriteString("\n")

			b.WriteString("## Stale watchlist items\n\n")
			any = false
			for _, w := range watchItems {
				if w.Status == model.WatchResolved || !isStale(w) {
					continue
				}
				fmt.Fprintf(&b, "- [%s] %s (%s) — last checked %s\n", w.ID, w.Text, w.Aspect, w.LastChecked)
				any = true
			}
			if !any {
				b.WriteString("_Nothing stale._\n")
			}

			if err := s.WriteLog(todayStr, b.String()); err != nil {
				return err
			}
			fmt.Print(b.String())
			return nil
		},
	}
}
