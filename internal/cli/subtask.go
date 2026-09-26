package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/VishalDalwadi/anchor/internal/model"
)

// validateParent checks that parentID can be the parent of the task
// childID (empty for a task not created yet): the parent must exist, be
// open, and not be the task itself or one of its own subtasks, which
// would make a loop.
func validateParent(tasks []model.Task, childID, parentID string) error {
	idx := findTask(tasks, parentID)
	if idx < 0 {
		return fmt.Errorf("no task with id %q to use as parent", parentID)
	}
	if tasks[idx].Status != model.TaskOpen {
		return fmt.Errorf("task %s is %s; only an open task can have subtasks added", parentID, tasks[idx].Status)
	}
	if childID == "" {
		return nil
	}
	for id := parentID; id != ""; {
		if id == childID {
			return fmt.Errorf("task %s can't be moved under %s: that's itself or one of its own subtasks", childID, parentID)
		}
		i := findTask(tasks, id)
		if i < 0 {
			break
		}
		id = tasks[i].Parent
	}
	return nil
}

// openDescendants returns every open task beneath id, at any depth, in
// file order.
func openDescendants(tasks []model.Task, id string) []model.Task {
	under := map[string]bool{id: true}
	// Repeat until no new descendants turn up, so a subtask listed before
	// its own parent in the file is still found.
	for grew := true; grew; {
		grew = false
		for _, t := range tasks {
			if !under[t.ID] && under[t.Parent] {
				under[t.ID] = true
				grew = true
			}
		}
	}
	var open []model.Task
	for _, t := range tasks {
		if t.ID != id && under[t.ID] && t.Status == model.TaskOpen {
			open = append(open, t)
		}
	}
	return open
}

// openSubtasksError explains why a parent can't be marked done yet.
func openSubtasksError(id string, open []model.Task) error {
	var b strings.Builder
	fmt.Fprintf(&b, "task %s has %d open subtask(s):\n", id, len(open))
	for _, t := range open {
		fmt.Fprintf(&b, "  %s\t%s\n", t.ID, t.Text)
	}
	b.WriteString("finish or drop them first, or use --force to mark it done anyway")
	return fmt.Errorf("%s", b.String())
}

// printTaskTree prints tasks with subtasks indented under their parent,
// keeping file order among siblings. A task whose parent isn't among
// those shown (done, or filtered out) is printed at the top level rather
// than hidden. flag, if non-nil, supplies an extra marker for a task's
// line (see printTask). inFocus, if non-nil, says which tasks are in
// focus: they're marked, and any tree (or subtree) containing one is
// listed before its siblings.
func printTaskTree(w io.Writer, tasks []model.Task, flag func(model.Task) string, inFocus func(model.Task) bool) {
	shown := map[string]model.Task{}
	for _, t := range tasks {
		shown[t.ID] = t
	}
	children := map[string][]model.Task{}
	var roots []model.Task
	for _, t := range tasks {
		if _, ok := shown[t.Parent]; ok && t.Parent != t.ID {
			children[t.Parent] = append(children[t.Parent], t)
		} else {
			roots = append(roots, t)
		}
	}

	// hot: tasks in focus, and every shown ancestor of one.
	hot := map[string]bool{}
	anyFocus := false
	if inFocus != nil {
		for _, t := range tasks {
			if !inFocus(t) {
				continue
			}
			anyFocus = true
			for id := t.ID; id != "" && !hot[id]; id = shown[id].Parent {
				if _, ok := shown[id]; !ok {
					break
				}
				hot[id] = true
			}
		}
	}
	hotFirst := func(list []model.Task) {
		sort.SliceStable(list, func(i, j int) bool { return hot[list[i].ID] && !hot[list[j].ID] })
	}
	hotFirst(roots)
	for _, list := range children {
		hotFirst(list)
	}

	printed := map[string]bool{} // guards against a parent loop in hand-edited data
	var walk func(t model.Task, depth int)
	walk = func(t model.Task, depth int) {
		if printed[t.ID] {
			return
		}
		printed[t.ID] = true
		label := ""
		if flag != nil {
			label = flag(t)
		}
		mark := ""
		if anyFocus {
			mark = noFocusMark
			if inFocus(t) {
				mark = focusMark
			}
		}
		printTask(w, mark, t, depth, label)
		for _, c := range children[t.ID] {
			walk(c, depth+1)
		}
	}
	for _, t := range roots {
		walk(t, 0)
	}
	// Anything left is stuck in a loop with no root; still show it.
	for _, t := range tasks {
		walk(t, 0)
	}
}
