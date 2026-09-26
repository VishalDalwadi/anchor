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
	cmd.AddCommand(newTaskBacklogCmd())
	cmd.AddCommand(newTaskActivateCmd())
	cmd.AddCommand(newTaskFocusCmd(true))
	cmd.AddCommand(newTaskFocusCmd(false))
	return cmd
}

func newTaskBacklogCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "backlog <id>",
		Short:             "Move a task to the backlog: still open, but hidden from `task list` until activated",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeActiveTaskIDs,
		RunE:              taskSetActive(false),
	}
}

func newTaskActivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "activate <id>",
		Short:             "Bring a backlogged task back into `task list`",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeBacklogTaskIDs,
		RunE:              taskSetActive(true),
	}
}

func taskSetActive(active bool) func(*cobra.Command, []string) error {
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
		tasks[idx].Active = active
		return s.SaveTasks(tasks)
	}
}

// isBacklogged reports whether t is open but set aside for now.
func isBacklogged(t model.Task) bool { return t.Status == model.TaskOpen && !t.Active }

func newTaskAddCmd() *cobra.Command {
	var aspectFlag, due, context, notes, parent string
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
			if parent != "" {
				if err := validateParent(tasks, "", parent); err != nil {
					return err
				}
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
				Parent:  parent,
				Active:  true,
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
	cmd.Flags().StringVar(&parent, "parent", "", "id of the parent task, to add this as a subtask")
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	registerTaskParentFlag(cmd)
	return cmd
}

func newTaskEditCmd() *cobra.Command {
	var text, due, context, aspectFlag, notes, moreNotes, parent string
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
			if cmd.Flags().Changed("parent") && parent != "" {
				if err := validateParent(tasks, args[0], parent); err != nil {
					return err
				}
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
			if cmd.Flags().Changed("parent") {
				tasks[idx].Parent = parent
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
	cmd.Flags().StringVar(&parent, "parent", "", `new parent task id ("--parent=" to make it a top-level task)`)
	cmd.MarkFlagsMutuallyExclusive("notes", "append-notes")
	registerAspectFlag(cmd)
	registerTaskParentFlag(cmd)
	return cmd
}

func newTaskDoneCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark a task done (its subtasks are not touched)",
		Long: `Mark a task done. Subtasks are not marked done along with it; each is
completed on its own. A task with open subtasks is refused unless --force.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTaskIDs,
		RunE: taskSetStatus(model.TaskDone, func(tasks []model.Task, id string) error {
			if open := openDescendants(tasks, id); len(open) > 0 && !force {
				return openSubtasksError(id, open)
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&force, "force", false, "mark done even if it has open subtasks")
	return cmd
}

func newTaskDropCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "drop <id>",
		Short:             "Drop a task",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTaskIDs,
		RunE:              taskSetStatus(model.TaskDropped, nil),
	}
}

// taskSetStatus sets a task's status, after check (if non-nil) approves.
func taskSetStatus(status string, check func(tasks []model.Task, id string) error) func(*cobra.Command, []string) error {
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
		if check != nil {
			if err := check(tasks, args[0]); err != nil {
				return err
			}
		}
		tasks[idx].Status = status
		return s.SaveTasks(tasks)
	}
}

func newTaskListCmd() *cobra.Command {
	var today bool
	var aspectFlag, context string
	var all, backlog bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List open tasks (not backlogged ones, unless --backlog or --all)",
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
			templates, err := s.LoadRecurring()
			if err != nil {
				return err
			}
			byID := templatesByID(templates)
			now := time.Now()
			focus, err := s.LoadFocus()
			if err != nil {
				return err
			}

			todayStr := validate.Today()
			var shown []model.Task
			for _, t := range tasks {
				switch {
				case all:
				case backlog && !isBacklogged(t):
					continue
				case !backlog && (t.Status != model.TaskOpen || isBacklogged(t)):
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
				shown = append(shown, t)
			}
			printTaskTree(cmd.OutOrStdout(), shown, func(t model.Task) string {
				return stillOwed(t, byID, now)
			}, taskInFocus(focus))
			return nil
		},
	}
	cmd.Flags().BoolVar(&today, "today", false, "only tasks due today")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "filter by life aspect")
	cmd.Flags().StringVar(&context, "context", "", "filter by context tag")
	cmd.Flags().BoolVar(&all, "all", false, "include done, dropped and backlogged tasks")
	cmd.Flags().BoolVar(&backlog, "backlog", false, "only backlogged tasks")
	cmd.MarkFlagsMutuallyExclusive("all", "backlog")
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

// printTask prints t as one list line: mark first (the focus marker, or
// blank padding), then indented two spaces per depth level (subtasks
// under their parent). A non-empty flag (e.g. "still owed") is shown right
// after the due date, where it can't be missed.
func printTask(w io.Writer, mark string, t model.Task, depth int, flag string) {
	status := t.Status
	if isBacklogged(t) {
		status += "/backlog"
	}
	line := fmt.Sprintf("%s%s%s\t[%s]\t%-8s\t%s", mark, strings.Repeat("  ", depth), t.ID, status, t.Aspect, t.Text)
	if t.Due != "" {
		line += "\t(due " + t.Due + ")"
	}
	if flag != "" {
		line += "\t" + flag
	}
	if t.Context != "" {
		line += "\t{" + t.Context + "}"
	}
	if t.Notes != "" {
		line += "\tnotes: " + notesPreview(t.Notes)
	}
	fmt.Fprintln(w, line)
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
			printTaskDetail(cmd.OutOrStdout(), tasks, tasks[idx])
			return nil
		},
	}
}

// printTaskDetail prints every set field of t, one per line, followed by
// its direct subtasks and its notes in full (indented, so multiline notes
// stay visibly grouped). tasks is the full list, to look up the parent's
// text and the subtasks.
func printTaskDetail(w io.Writer, tasks []model.Task, t model.Task) {
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
	if t.Parent != "" {
		parent := t.Parent
		if i := findTask(tasks, t.Parent); i >= 0 {
			parent += " (" + tasks[i].Text + ")"
		}
		field("parent", parent)
	}
	var subtasks []model.Task
	for _, c := range tasks {
		if c.Parent == t.ID {
			subtasks = append(subtasks, c)
		}
	}
	if len(subtasks) > 0 {
		fmt.Fprintln(w, "subtasks:")
		for _, c := range subtasks {
			fmt.Fprintf(w, "  %s\t[%s]\t%s\n", c.ID, c.Status, c.Text)
		}
	}
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
