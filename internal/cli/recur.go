package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/cronutil"
	"github.com/VishalDalwadi/anchor/internal/idgen"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

func newRecurCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recur",
		Short: "Manage cron-driven recurring task templates",
	}
	cmd.AddCommand(newRecurAddCmd())
	cmd.AddCommand(newRecurEditCmd())
	cmd.AddCommand(newRecurListCmd())
	cmd.AddCommand(newRecurRemoveCmd())
	cmd.AddCommand(newRecurRunCmd())
	return cmd
}

func newRecurAddCmd() *cobra.Command {
	var cronExpr, aspectFlag, context string
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a recurring task template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aspect.Validate(aspectFlag); err != nil {
				return err
			}
			if _, err := cronutil.Parse(cronExpr); err != nil {
				return err
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}

			r := model.RecurringTemplate{
				ID:      idgen.New("r"),
				Text:    args[0],
				Aspect:  aspectFlag,
				Cron:    cronExpr,
				Context: context,
			}
			templates = append(templates, r)
			if err := s.SaveRecurring(templates); err != nil {
				return err
			}
			fmt.Println(r.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&cronExpr, "cron", "", "5-field cron expression (required)")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "life aspect (required)")
	cmd.Flags().StringVar(&context, "context", "", "context tag applied to spawned tasks")
	cmd.MarkFlagRequired("cron")
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	return cmd
}

func newRecurEditCmd() *cobra.Command {
	var text, cronExpr, context string
	cmd := &cobra.Command{
		Use:               "edit <id>",
		Short:             "Edit a recurring task template",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeRecurringIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("cron") {
				if _, err := cronutil.Parse(cronExpr); err != nil {
					return err
				}
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}
			idx := findRecurring(templates, args[0])
			if idx < 0 {
				return fmt.Errorf("no recurring template with id %q", args[0])
			}
			if cmd.Flags().Changed("text") {
				templates[idx].Text = text
			}
			if cmd.Flags().Changed("cron") {
				templates[idx].Cron = cronExpr
			}
			if cmd.Flags().Changed("context") {
				templates[idx].Context = context
			}
			return s.SaveRecurring(templates)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "new text")
	cmd.Flags().StringVar(&cronExpr, "cron", "", "new cron expression")
	cmd.Flags().StringVar(&context, "context", "", "new context tag")
	return cmd
}

func newRecurListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List recurring task templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}
			for _, r := range templates {
				line := fmt.Sprintf("%s\t%-8s\t%q\t%s", r.ID, r.Aspect, r.Cron, r.Text)
				if r.LastSpawned != "" {
					line += "\t(last spawned " + r.LastSpawned + ")"
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}

func newRecurRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "remove <id>",
		Short:             "Remove a recurring task template",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeRecurringIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}
			idx := findRecurring(templates, args[0])
			if idx < 0 {
				return fmt.Errorf("no recurring template with id %q", args[0])
			}
			templates = append(templates[:idx], templates[idx+1:]...)
			return s.SaveRecurring(templates)
		},
	}
}

// newRecurRunCmd evaluates every template's cron against today and spawns a
// Task for any that are due and haven't already been spawned today.
// Idempotent: safe to invoke multiple times per day. Intended to be driven
// by Windows Task Scheduler running daily, not invoked by other commands.
func newRecurRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Spawn tasks for recurring templates due today",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}
			tasks, err := s.LoadTasks()
			if err != nil {
				return err
			}

			today := time.Now()
			todayStr := validate.Today()
			spawned := 0

			for i, r := range templates {
				if r.LastSpawned == todayStr {
					continue
				}
				sched, err := cronutil.Parse(r.Cron)
				if err != nil {
					return fmt.Errorf("template %s: %w", r.ID, err)
				}
				if !cronutil.MatchesDay(sched, today) {
					continue
				}

				tasks = append(tasks, model.Task{
					ID:              idgen.New("t"),
					Text:            r.Text,
					Aspect:          r.Aspect,
					Created:         todayStr,
					Status:          model.TaskOpen,
					Context:         r.Context,
					RecurringSource: r.ID,
				})
				templates[i].LastSpawned = todayStr
				spawned++
			}

			if spawned == 0 {
				return nil
			}
			if err := s.SaveTasks(tasks); err != nil {
				return err
			}
			return s.SaveRecurring(templates)
		},
	}
}

func findRecurring(templates []model.RecurringTemplate, id string) int {
	for i, r := range templates {
		if r.ID == id {
			return i
		}
	}
	return -1
}
