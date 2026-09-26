package cli

import (
	"fmt"
	"strings"
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
	var cronExpr, aspectFlag, context, onMiss, until, dueIn string
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
			if err := validateOnMiss(onMiss); err != nil {
				return err
			}
			if _, err := validate.DueOffset(dueIn, time.Now()); err != nil {
				return err
			}
			untilDate, err := validate.FlexDate(until, time.Now())
			if err != nil {
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
				OnMiss:  onMiss,
				Until:   untilDate,
				DueIn:   strings.TrimSpace(dueIn),
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
	cmd.Flags().StringVar(&onMiss, "on-miss", model.OnMissExpire, onMissUsage)
	cmd.Flags().StringVar(&until, "until", "", dateUsage("last day a task may spawn (default: forever)"))
	cmd.Flags().StringVar(&dueIn, "due-in", "", dueInUsage("how long after each fire day a task is due (default: due the day it fires)"))
	registerOnMissFlag(cmd)
	cmd.MarkFlagRequired("cron")
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	return cmd
}

func newRecurEditCmd() *cobra.Command {
	var text, cronExpr, context, onMiss, until, dueIn string
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
			if cmd.Flags().Changed("on-miss") {
				if err := validateOnMiss(onMiss); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("until") {
				var err error
				if until, err = validate.FlexDate(until, time.Now()); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("due-in") {
				if _, err := validate.DueOffset(dueIn, time.Now()); err != nil {
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
			if cmd.Flags().Changed("on-miss") {
				templates[idx].OnMiss = onMiss
			}
			if cmd.Flags().Changed("until") {
				templates[idx].Until = until
			}
			if cmd.Flags().Changed("due-in") {
				templates[idx].DueIn = strings.TrimSpace(dueIn)
			}
			return s.SaveRecurring(templates)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "new text")
	cmd.Flags().StringVar(&cronExpr, "cron", "", "new cron expression")
	cmd.Flags().StringVar(&context, "context", "", "new context tag")
	cmd.Flags().StringVar(&onMiss, "on-miss", "", onMissUsage)
	cmd.Flags().StringVar(&until, "until", "", dateUsage(`new last day a task may spawn ("--until=" for forever)`))
	cmd.Flags().StringVar(&dueIn, "due-in", "", dueInUsage(`new offset from fire day to due date ("--due-in=" for due the day it fires)`))
	registerOnMissFlag(cmd)
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
			now := time.Now()
			for _, r := range templates {
				onMiss := r.OnMiss
				if onMiss == "" {
					onMiss = model.OnMissExpire
				}
				line := fmt.Sprintf("%s\t%-8s\t%q\t%-7s\t%s", r.ID, r.Aspect, r.Cron, onMiss, r.Text)
				if r.DueIn != "" {
					line += "\tdue in " + r.DueIn
				}
				if r.Until != "" {
					line += "\tuntil " + r.Until
					if recurEnded(r, now) {
						line += " (ended)"
					}
				}
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

// newRecurRunCmd spawns tasks for every template due by today (see
// spawnRecurring for how on_miss shapes that). Idempotent: safe to invoke
// multiple times per day. Intended to be driven by Windows Task Scheduler
// running daily, not invoked by other commands.
func newRecurRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Spawn tasks for recurring templates that are due",
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

			tasks, res, err := spawnRecurring(templates, tasks, time.Now())
			if err != nil {
				return err
			}
			if res.spawned == 0 {
				return nil
			}
			if err := s.SaveTasks(tasks); err != nil {
				return err
			}
			if err := s.SaveRecurring(templates); err != nil {
				return err
			}
			msg := fmt.Sprintf("spawned %d task(s)", res.spawned)
			if res.dropped > 0 {
				msg += fmt.Sprintf(", dropped %d unfinished expired instance(s)", res.dropped)
			}
			fmt.Fprintln(cmd.OutOrStdout(), msg)
			return nil
		},
	}
}

const onMissUsage = `what happens to an unfinished instance once its period passes: "expire" (dropped when the next spawns) or "persist" (stays open and overdue until done)`

// dueInUsage is the help text for --due-in, which takes relative forms only.
func dueInUsage(what string) string {
	return what + ": 2d/1w/1m or 72h (no fixed dates)"
}

func validateOnMiss(s string) error {
	for _, v := range model.ValidOnMiss {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("invalid --on-miss %q: must be one of %v", s, model.ValidOnMiss)
}

func registerOnMissFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("on-miss", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return model.ValidOnMiss, cobra.ShellCompDirectiveNoFileComp
	})
}

func findRecurring(templates []model.RecurringTemplate, id string) int {
	for i, r := range templates {
		if r.ID == id {
			return i
		}
	}
	return -1
}
