package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/VishalDalwadi/anchor/internal/model"
)

func goal(id, parent, tier string) model.Goal {
	return model.Goal{ID: id, Text: "text " + id, Aspect: "career", Tier: tier, Status: model.GoalActive, Parent: parent}
}

// goalTreeLines prints the tree and returns each line's indent + id.
func goalTreeLines(goals []model.Goal) []string {
	var out bytes.Buffer
	printGoalTree(&out, goals)
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		lines = append(lines, strings.SplitN(line, "\t", 2)[0])
	}
	return lines
}

func TestGoalTreeShowsGoalsWhoseParentIsFilteredOut(t *testing.T) {
	// As with `goal list --tree --tier monthly`: the yearly parent y is
	// filtered out, so its monthly children must show at the top level.
	got := goalTreeLines([]model.Goal{
		goal("m1", "y", model.TierMonthly),
		goal("w1", "m1", model.TierWeekly),
		goal("m2", "y", model.TierMonthly),
	})
	want := []string{"m1", "  w1", "m2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("tree = %q, want %q", got, want)
	}
}

func TestGoalTreeSurvivesParentLoop(t *testing.T) {
	got := goalTreeLines([]model.Goal{goal("a", "b", model.TierYearly), goal("b", "a", model.TierYearly)})
	if len(got) != 2 {
		t.Errorf("tree for a 2-goal loop = %q, want both goals once", got)
	}
}

func TestValidateGoalParent(t *testing.T) {
	goals := []model.Goal{
		goal("life", "", model.TierLife),
		goal("y", "life", model.TierYearly),
		goal("m", "y", model.TierMonthly),
	}
	tests := []struct {
		child, parent, wantErr string
	}{
		{"m", "life", ""},
		{"y", "missing", "no goal"},
		{"y", "y", "itself"},
		{"life", "m", "own descendants"},
	}
	for _, tt := range tests {
		err := validateGoalParent(goals, tt.child, tt.parent)
		switch {
		case tt.wantErr == "" && err != nil:
			t.Errorf("validateGoalParent(%q, %q) = %v, want ok", tt.child, tt.parent, err)
		case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
			t.Errorf("validateGoalParent(%q, %q) = %v, want error mentioning %q", tt.child, tt.parent, err, tt.wantErr)
		}
	}
}
