package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

// Focus is a nudge, not a filter: whatever's in focus — tasks and goals
// marked with `focus`, and everything in the focused aspect — is marked
// and listed first by `task list` and `goal list`, with nothing hidden.

// focusMark prefixes a list line that's in focus; lines that aren't get
// the same width of blank so columns stay aligned.
const (
	focusMark   = "* "
	noFocusMark = "  "
)

func taskInFocus(f model.FocusState) func(model.Task) bool {
	return func(t model.Task) bool { return t.Focus || (f.Aspect != "" && t.Aspect == f.Aspect) }
}

func goalInFocus(f model.FocusState) func(model.Goal) bool {
	return func(g model.Goal) bool { return g.Focus || (f.Aspect != "" && g.Aspect == f.Aspect) }
}

func newFocusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "focus",
		Short: "Put an aspect in focus: its tasks and goals are marked and listed first",
	}
	cmd.AddCommand(newFocusSetCmd())
	cmd.AddCommand(newFocusClearCmd())
	cmd.AddCommand(newFocusShowCmd())
	return cmd
}

func newFocusSetCmd() *cobra.Command {
	var aspectFlag string
	cmd := &cobra.Command{
		Use:   "set --aspect <aspect>",
		Short: "Focus on an aspect (replacing any aspect already in focus)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aspect.Validate(aspectFlag); err != nil {
				return err
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			return s.SaveFocus(model.FocusState{Aspect: aspectFlag, Since: validate.Today()})
		},
	}
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "life aspect to focus on (required)")
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	return cmd
}

func newFocusClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Stop focusing on an aspect (focused tasks and goals stay focused)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			return s.SaveFocus(model.FocusState{})
		},
	}
}

func newFocusShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the aspect in focus, and every focused goal and task",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			f, err := s.LoadFocus()
			if err != nil {
				return err
			}
			goals, err := s.LoadGoals()
			if err != nil {
				return err
			}
			tasks, err := s.LoadTasks()
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if f.Aspect == "" {
				fmt.Fprintln(w, "aspect: none (anchor focus set --aspect <aspect>)")
			} else {
				fmt.Fprintf(w, "aspect: %s (since %s)\n", f.Aspect, f.Since)
			}

			var fg []model.Goal
			for _, g := range goals {
				if g.Focus && g.Status == model.GoalActive {
					fg = append(fg, g)
				}
			}
			var ft []model.Task
			for _, t := range tasks {
				if t.Focus && t.Status == model.TaskOpen {
					ft = append(ft, t)
				}
			}
			if len(fg) > 0 {
				fmt.Fprintln(w, "\nfocused goals:")
				for _, g := range fg {
					printGoal(w, "  ", g, 0)
				}
			}
			if len(ft) > 0 {
				fmt.Fprintln(w, "\nfocused tasks:")
				for _, t := range ft {
					printTask(w, "  ", t, 0, "")
				}
			}
			if len(fg) == 0 && len(ft) == 0 {
				fmt.Fprintln(w, "no focused goals or tasks (goal focus <id>, task focus <id>)")
			}
			return nil
		},
	}
}

func newTaskFocusCmd(focus bool) *cobra.Command {
	use, short, complete := "focus <id>", "Mark a task as in focus: listed first, with a *", completeUnfocusedTaskIDs
	if !focus {
		use, short, complete = "unfocus <id>", "Take a task out of focus", completeFocusedTaskIDs
	}
	return &cobra.Command{
		Use:               use,
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			tasks, err := s.LoadTasks()
			if err != nil {
				return err
			}
			idx := findTask(tasks, args[0])
			if idx < 0 {
				return fmt.Errorf("no task with id %q", args[0])
			}
			tasks[idx].Focus = focus
			return s.SaveTasks(tasks)
		},
	}
}

func newGoalFocusCmd(focus bool) *cobra.Command {
	use, short, complete := "focus <id>", "Mark a goal as in focus: listed first, with a *", completeUnfocusedGoalIDs
	if !focus {
		use, short, complete = "unfocus <id>", "Take a goal out of focus", completeFocusedGoalIDs
	}
	return &cobra.Command{
		Use:               use,
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			goals, err := s.LoadGoals()
			if err != nil {
				return err
			}
			idx := findGoal(goals, args[0])
			if idx < 0 {
				return fmt.Errorf("no goal with id %q", args[0])
			}
			goals[idx].Focus = focus
			return s.SaveGoals(goals)
		},
	}
}
