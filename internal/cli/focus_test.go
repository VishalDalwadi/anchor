package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/VishalDalwadi/anchor/internal/model"
)

// lineHeads returns each output line up to its first tab: the focus mark,
// indent and id.
func lineHeads(out string) []string {
	var heads []string
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		heads = append(heads, strings.SplitN(line, "\t", 2)[0])
	}
	return heads
}

func TestTaskTreeFocusFirstAndMarked(t *testing.T) {
	tasks := []model.Task{
		{ID: "a", Aspect: "career", Status: model.TaskOpen},
		{ID: "b", Aspect: "physical", Status: model.TaskOpen},
		{ID: "b1", Aspect: "career", Status: model.TaskOpen, Parent: "b", Focus: true},
		{ID: "c", Aspect: "finances", Status: model.TaskOpen},
		{ID: "d", Aspect: "finances", Status: model.TaskOpen},
	}
	var out bytes.Buffer
	printTaskTree(&out, tasks, nil, taskInFocus(model.FocusState{Aspect: "finances"}))

	// b rises because its subtask b1 is focused; c and d because their
	// aspect is; a stays last. File order is kept within each group.
	want := []string{"  b", "*   b1", "* c", "* d", "  a"}
	if got := lineHeads(out.String()); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("tree = %q, want %q", got, want)
	}
}

func TestTaskTreeUnchangedWithoutFocus(t *testing.T) {
	tasks := []model.Task{
		{ID: "a", Aspect: "career", Status: model.TaskOpen},
		{ID: "b", Aspect: "physical", Status: model.TaskOpen},
	}
	var out bytes.Buffer
	printTaskTree(&out, tasks, nil, taskInFocus(model.FocusState{}))
	if want := []string{"a", "b"}; strings.Join(lineHeads(out.String()), "|") != strings.Join(want, "|") {
		t.Errorf("with nothing in focus, lines = %q, want %q (no marks, no reordering)", lineHeads(out.String()), want)
	}
}

func TestGoalTreeFocusFirstAndMarked(t *testing.T) {
	goals := []model.Goal{
		{ID: "y1", Text: "a yearly", Aspect: "career", Tier: model.TierYearly, Status: model.GoalActive},
		{ID: "y2", Text: "b yearly", Aspect: "career", Tier: model.TierYearly, Status: model.GoalActive},
		{ID: "m1", Text: "a monthly", Aspect: "career", Tier: model.TierMonthly, Status: model.GoalActive, Parent: "y2"},
		{ID: "m2", Text: "b monthly", Aspect: "career", Tier: model.TierMonthly, Status: model.GoalActive, Parent: "y2", Focus: true},
	}
	var out bytes.Buffer
	printGoalTree(&out, goals, goalInFocus(model.FocusState{}))

	// y2 rises above y1 (its child m2 is focused), and m2 above m1.
	want := []string{"  y2", "*   m2", "    m1", "  y1"}
	if got := lineHeads(out.String()); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("tree = %q, want %q", got, want)
	}
}

func TestInFocus(t *testing.T) {
	f := model.FocusState{Aspect: "career"}
	if !taskInFocus(f)(model.Task{Aspect: "career"}) || !taskInFocus(f)(model.Task{Aspect: "mental", Focus: true}) {
		t.Error("task in focused aspect, or flagged, should be in focus")
	}
	if taskInFocus(f)(model.Task{Aspect: "mental"}) || taskInFocus(model.FocusState{})(model.Task{}) {
		t.Error("unflagged task outside the focused aspect should not be in focus")
	}
	if !goalInFocus(f)(model.Goal{Aspect: "career"}) || goalInFocus(f)(model.Goal{Aspect: "mental"}) {
		t.Error("goal focus by aspect is wrong")
	}
}
