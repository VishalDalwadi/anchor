package cli

import (
	"fmt"
	"io"
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
			if cmd.Flags().Changed("parent") && parent != "" {
				if err := validateGoalParent(goals, args[0], parent); err != nil {
					return err
				}
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
	cmd.Flags().StringVar(&parent, "parent", "", `new parent goal id ("--parent=" to detach it)`)
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
				printGoalTree(cmd.OutOrStdout(), filtered)
			} else {
				for _, g := range filtered {
					printGoal(cmd.OutOrStdout(), g, 0)
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

// validateGoalParent checks that parentID can be the parent of goal
// childID: it must exist and not be the goal itself or one of its own
// descendants, which would make a loop.
func validateGoalParent(goals []model.Goal, childID, parentID string) error {
	if findGoal(goals, parentID) < 0 {
		return fmt.Errorf("no goal with id %q to use as parent", parentID)
	}
	for id := parentID; id != ""; {
		if id == childID {
			return fmt.Errorf("goal %s can't be moved under %s: that's itself or one of its own descendants", childID, parentID)
		}
		i := findGoal(goals, id)
		if i < 0 {
			break
		}
		id = goals[i].Parent
	}
	return nil
}

// printGoalTree prints goals with children indented under their parent.
// A goal whose parent isn't among those shown (filtered out by --tier or
// --aspect, or missing) is printed at the top level rather than hidden.
func printGoalTree(w io.Writer, goals []model.Goal) {
	shown := map[string]bool{}
	for _, g := range goals {
		shown[g.ID] = true
	}
	byParent := map[string][]model.Goal{}
	for _, g := range goals {
		parent := g.Parent
		if !shown[parent] || parent == g.ID {
			parent = ""
		}
		byParent[parent] = append(byParent[parent], g)
	}
	for _, list := range byParent {
		sort.Slice(list, func(i, j int) bool { return list[i].Text < list[j].Text })
	}

	printed := map[string]bool{} // guards against a parent loop in hand-edited data
	var walk func(parent string, depth int)
	walk = func(parent string, depth int) {
		for _, g := range byParent[parent] {
			if printed[g.ID] {
				continue
			}
			printed[g.ID] = true
			printGoal(w, g, depth)
			walk(g.ID, depth+1)
		}
	}
	walk("", 0)
	// Anything left is stuck in a loop with no root; still show it.
	for _, g := range goals {
		if !printed[g.ID] {
			printed[g.ID] = true
			printGoal(w, g, 0)
			walk(g.ID, 1)
		}
	}
}

func printGoal(w io.Writer, g model.Goal, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	line := fmt.Sprintf("%s%s\t[%s]\t%-7s\t%-8s\t%s", indent, g.ID, g.Status, g.Tier, g.Aspect, g.Text)
	if g.Period != "" {
		line += "\t(" + g.Period + ")"
	}
	fmt.Fprintln(w, line)
}

func findGoal(goals []model.Goal, id string) int {
	for i, g := range goals {
		if g.ID == id {
			return i
		}
	}
	return -1
}
