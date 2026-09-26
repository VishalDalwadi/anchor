package cli

import (
	"fmt"

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
	return cmd
}

func newTaskAddCmd() *cobra.Command {
	var aspectFlag, due, context string
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aspect.Validate(aspectFlag); err != nil {
				return err
			}
			if err := validate.Date(due); err != nil {
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
				Due:     due,
				Status:  model.TaskOpen,
				Context: context,
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
	cmd.Flags().StringVar(&due, "due", "", "due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&context, "context", "", "context tag, e.g. phone, errand, desk, home")
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	return cmd
}

func newTaskEditCmd() *cobra.Command {
	var text, due, context, aspectFlag string
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
			if cmd.Flags().Changed("due") {
				if err := validate.Date(due); err != nil {
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
			return s.SaveTasks(tasks)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "new text")
	cmd.Flags().StringVar(&due, "due", "", "new due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&context, "context", "", "new context tag")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "new life aspect")
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
	fmt.Println(line)
}
