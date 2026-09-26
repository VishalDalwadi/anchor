package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/idgen"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newTaskAddCmd())
	cmd.AddCommand(newTaskEditCmd())
	cmd.AddCommand(newTaskDoneCmd())
	cmd.AddCommand(newTaskDropCmd())
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskShowCmd())
	return cmd
}

func newTaskAddCmd() *cobra.Command {
	var aspectFlag, due, context, notes string
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aspect.Validate(aspectFlag); err != nil {
				return err
			}
			dueDate, err := validate.FlexDate(due, time.Now())
			if err != nil {
				return err
			}
			if notes, err = readNotes(cmd, notes); err != nil {
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

			t := model.Task{
				ID:      idgen.New("t"),
				Text:    args[0],
				Aspect:  aspectFlag,
				Created: validate.Today(),
				Due:     dueDate,
				Status:  model.TaskOpen,
				Context: context,
				Notes:   notes,
			}
			tasks = append(tasks, t)
			if err := s.SaveTasks(tasks); err != nil {
				return err
			}
			fmt.Println(t.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "life aspect (required)")
	cmd.Flags().StringVar(&due, "due", "", dateUsage("due date"))
	cmd.Flags().StringVar(&context, "context", "", "context tag, e.g. phone, errand, desk, home")
	cmd.Flags().StringVar(&notes, "notes", "", notesUsage("notes"))
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	return cmd
}

func newTaskEditCmd() *cobra.Command {
	var text, due, context, aspectFlag, notes, moreNotes string
	cmd := &cobra.Command{
		Use:               "edit <id>",
		Short:             "Edit a task",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTaskIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("aspect") {
				if err := aspect.Validate(aspectFlag); err != nil {
					return err
				}
			}
			var err error
			if cmd.Flags().Changed("due") {
				if due, err = validate.FlexDate(due, time.Now()); err != nil {
					return err
				}
			}
			if notes, err = readNotes(cmd, notes); err != nil {
				return err
			}
			if moreNotes, err = readNotes(cmd, moreNotes); err != nil {
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

			idx := findTask(tasks, args[0])
			if idx < 0 {
				return fmt.Errorf("no task with id %q", args[0])
			}
			if cmd.Flags().Changed("text") {
				tasks[idx].Text = text
			}
			if cmd.Flags().Changed("due") {
				tasks[idx].Due = due
			}
			if cmd.Flags().Changed("context") {
				tasks[idx].Context = context
			}
			if cmd.Flags().Changed("aspect") {
				tasks[idx].Aspect = aspectFlag
			}
			if cmd.Flags().Changed("notes") {
				tasks[idx].Notes = notes
			}
			if cmd.Flags().Changed("append-notes") {
				tasks[idx].Notes = appendNotes(tasks[idx].Notes, moreNotes)
			}
			return s.SaveTasks(tasks)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "new text")
	cmd.Flags().StringVar(&due, "due", "", dateUsage("new due date"))
	cmd.Flags().StringVar(&context, "context", "", "new context tag")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "new life aspect")
	cmd.Flags().StringVar(&notes, "notes", "", notesUsage("replace notes"))
	cmd.Flags().StringVar(&moreNotes, "append-notes", "", notesUsage("append a line to notes"))
	cmd.MarkFlagsMutuallyExclusive("notes", "append-notes")
	registerAspectFlag(cmd)
	return cmd
}

func newTaskDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "done <id>",
		Short:             "Mark a task done",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTaskIDs,
		RunE:              taskSetStatus(model.TaskDone),
	}
}

func newTaskDropCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "drop <id>",
		Short:             "Drop a task",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTaskIDs,
		RunE:              taskSetStatus(model.TaskDropped),
	}
}

func taskSetStatus(status string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
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
		tasks[idx].Status = status
		return s.SaveTasks(tasks)
	}
}

func newTaskListCmd() *cobra.Command {
	var today bool
	var aspectFlag, context string
	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if aspectFlag != "" {
				if err := aspect.Validate(aspectFlag); err != nil {
					return err
				}
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			tasks, err := s.LoadTasks()
			if err != nil {
				return err
			}

			todayStr := validate.Today()
			for _, t := range tasks {
				if !all && t.Status != model.TaskOpen {
					continue
				}
				if today && t.Due != todayStr {
					continue
				}
				if aspectFlag != "" && t.Aspect != aspectFlag {
					continue
				}
				if context != "" && t.Context != context {
					continue
				}
				printTask(t)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&today, "today", false, "only tasks due today")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "filter by life aspect")
	cmd.Flags().StringVar(&context, "context", "", "filter by context tag")
	cmd.Flags().BoolVar(&all, "all", false, "include done/dropped tasks")
	registerAspectFlag(cmd)
	return cmd
}

func findTask(tasks []model.Task, id string) int {
	for i, t := range tasks {
		if t.ID == id {
			return i
		}
	}
	return -1
}

func printTask(t model.Task) {
	line := fmt.Sprintf("%s\t[%s]\t%-8s\t%s", t.ID, t.Status, t.Aspect, t.Text)
	if t.Due != "" {
		line += "\t(due " + t.Due + ")"
	}
	if t.Context != "" {
		line += "\t{" + t.Context + "}"
	}
	if t.Notes != "" {
		line += "\tnotes: " + notesPreview(t.Notes)
	}
	fmt.Println(line)
}

func newTaskShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "show <id>",
		Short:             "Show a task in full, including notes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTaskIDs,
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
			printTaskDetail(cmd.OutOrStdout(), tasks[idx])
			return nil
		},
	}
}

// printTaskDetail prints every set field of t, one per line, followed by
// its notes in full (indented, so multiline notes stay visibly grouped).
func printTaskDetail(w io.Writer, t model.Task) {
	field := func(label, value string) {
		if value != "" {
			fmt.Fprintf(w, "%-10s %s\n", label+":", value)
		}
	}
	field("id", t.ID)
	field("text", t.Text)
	field("status", t.Status)
	field("aspect", t.Aspect)
	field("created", t.Created)
	field("due", t.Due)
	field("context", t.Context)
	field("recurring", t.RecurringSource)
	if t.Notes != "" {
		fmt.Fprintln(w, "notes:")
		for _, line := range strings.Split(t.Notes, "\n") {
			fmt.Fprintln(w, "  "+line)
		}
	}
}

// dateUsage is the help text for flexible date flags (--due, --expected).
func dateUsage(what string) string {
	return what + ": 2026-08-31, 2d/1w/1m, 72h, a month (aug), or a year (2026)"
}
