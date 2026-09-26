package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/idgen"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/store"
)

func newGoalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goal",
		Short: "Manage tiered goals",
	}
	cmd.AddCommand(newGoalAddCmd())
	cmd.AddCommand(newGoalEditCmd())
	cmd.AddCommand(newGoalDoneCmd())
	cmd.AddCommand(newGoalDropCmd())
	cmd.AddCommand(newGoalListCmd())
	return cmd
}

func validateTier(tier string) error {
	for _, v := range model.ValidTiers {
		if tier == v {
			return nil
		}
	}
	return fmt.Errorf("invalid tier %q: must be one of %v", tier, model.ValidTiers)
}

func newGoalAddCmd() *cobra.Command {
	var tier, aspectFlag, period, parent string
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a goal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTier(tier); err != nil {
				return err
			}
			if err := aspect.Validate(aspectFlag); err != nil {
				return err
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			goals, err := s.LoadGoals()
			if err != nil {
				return err
			}
			if parent != "" && findGoal(goals, parent) < 0 {
				return fmt.Errorf("no goal with id %q to use as parent", parent)
			}
			return addGoal(s, goals, args[0], aspectFlag, tier, period, parent)
		},
	}
	cmd.Flags().StringVar(&tier, "tier", "", "goal tier: life|yearly|monthly|weekly (required)")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "life aspect (required)")
	cmd.Flags().StringVar(&period, "period", "", `period, e.g. "2026", "2026-08", "2026-W34"`)
	cmd.Flags().StringVar(&parent, "parent", "", "id of the goal one tier up")
	cmd.MarkFlagRequired("tier")
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	registerTierFlag(cmd)
	registerParentFlag(cmd)
	return cmd
}

func addGoal(s *store.Store, goals []model.Goal, text, aspectFlag, tier, period, parent string) error {
	g := model.Goal{
		ID:     idgen.New("g"),
		Text:   text,
		Aspect: aspectFlag,
		Tier:   tier,
		Period: period,
		Status: model.GoalActive,
		Parent: parent,
	}
	goals = append(goals, g)
	if err := s.SaveGoals(goals); err != nil {
		return err
	}
	fmt.Println(g.ID)
	return nil
}

func newGoalEditCmd() *cobra.Command {
	var text, period, parent string
	cmd := &cobra.Command{
		Use:               "edit <id>",
		Short:             "Edit a goal",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeGoalIDs,
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
			if cmd.Flags().Changed("parent") && findGoal(goals, parent) < 0 {
				return fmt.Errorf("no goal with id %q to use as parent", parent)
			}
			if cmd.Flags().Changed("text") {
				goals[idx].Text = text
			}
			if cmd.Flags().Changed("period") {
				goals[idx].Period = period
			}
			if cmd.Flags().Changed("parent") {
				goals[idx].Parent = parent
			}
			return s.SaveGoals(goals)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "new text")
	cmd.Flags().StringVar(&period, "period", "", "new period")
	cmd.Flags().StringVar(&parent, "parent", "", "new parent goal id")
	registerParentFlag(cmd)
	return cmd
}

func newGoalDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "done <id>",
		Short:             "Mark a goal done",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeGoalIDs,
		RunE:              goalSetStatus(model.GoalDone),
	}
}

func newGoalDropCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "drop <id>",
		Short:             "Drop a goal",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeGoalIDs,
		RunE:              goalSetStatus(model.GoalDropped),
	}
}

func goalSetStatus(status string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
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
		goals[idx].Status = status
		return s.SaveGoals(goals)
	}
}

func newGoalListCmd() *cobra.Command {
	var tier, aspectFlag string
	var tree bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List goals",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if tier != "" {
				if err := validateTier(tier); err != nil {
					return err
				}
			}
			if aspectFlag != "" {
				if err := aspect.Validate(aspectFlag); err != nil {
					return err
				}
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			goals, err := s.LoadGoals()
			if err != nil {
				return err
			}

			var filtered []model.Goal
			for _, g := range goals {
				if tier != "" && g.Tier != tier {
					continue
				}
				if aspectFlag != "" && g.Aspect != aspectFlag {
					continue
				}
				filtered = append(filtered, g)
			}

			if tree {
				printGoalTree(filtered)
			} else {
				for _, g := range filtered {
					printGoal(g, 0)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tier, "tier", "", "filter by tier")
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "filter by life aspect")
	cmd.Flags().BoolVar(&tree, "tree", false, "print as a parent/child tree")
	registerAspectFlag(cmd)
	registerTierFlag(cmd)
	return cmd
}

func printGoalTree(goals []model.Goal) {
	byParent := map[string][]model.Goal{}
	for _, g := range goals {
		byParent[g.Parent] = append(byParent[g.Parent], g)
	}
	for _, list := range byParent {
		sort.Slice(list, func(i, j int) bool { return list[i].Text < list[j].Text })
	}

	var walk func(parent string, depth int)
	walk = func(parent string, depth int) {
		for _, g := range byParent[parent] {
			printGoal(g, depth)
			walk(g.ID, depth+1)
		}
	}
	walk("", 0)
}

func printGoal(g model.Goal, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	line := fmt.Sprintf("%s%s\t[%s]\t%-7s\t%-8s\t%s", indent, g.ID, g.Status, g.Tier, g.Aspect, g.Text)
	if g.Period != "" {
		line += "\t(" + g.Period + ")"
	}
	fmt.Println(line)
}

func findGoal(goals []model.Goal, id string) int {
	for i, g := range goals {
		if g.ID == id {
			return i
		}
	}
	return -1
}
