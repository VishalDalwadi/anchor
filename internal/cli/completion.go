package cli

import (
	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/model"
)

// completeAspect completes --aspect with the six fixed life aspects.
func completeAspect(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return aspect.Valid, cobra.ShellCompDirectiveNoFileComp
}

// completeTier completes --tier with the four goal tiers.
func completeTier(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return model.ValidTiers, cobra.ShellCompDirectiveNoFileComp
}

// registerAspectFlag wires --aspect completion onto cmd. Call after the
// flag has been defined.
func registerAspectFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("aspect", completeAspect)
}

// registerTierFlag wires --tier completion onto cmd.
func registerTierFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("tier", completeTier)
}

// registerParentFlag wires --parent completion onto cmd with goal IDs.
func registerParentFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("parent", completeGoalIDs)
}

// completeTaskIDs completes a task <id> positional argument.
func completeTaskIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	tasks, err := s.LoadTasks()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, t.ID+"\t"+t.Text)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeGoalIDs completes a goal <id> positional argument (and the
// --parent flag, which also expects a goal ID).
func completeGoalIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	goals, err := s.LoadGoals()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(goals))
	for _, g := range goals {
		out = append(out, g.ID+"\t"+g.Text)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeWatchIDs completes a watchlist <id> positional argument.
func completeWatchIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	items, err := s.LoadWatchItems()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(items))
	for _, w := range items {
		out = append(out, w.ID+"\t"+w.Text)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeRecurringIDs completes a recurring template <id> positional
// argument.
func completeRecurringIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	templates, err := s.LoadRecurring()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(templates))
	for _, r := range templates {
		out = append(out, r.ID+"\t"+r.Text)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
