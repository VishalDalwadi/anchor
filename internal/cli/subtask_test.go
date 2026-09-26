package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/VishalDalwadi/anchor/internal/model"
)

func task(id, parent, status string) model.Task {
	return model.Task{ID: id, Text: "text " + id, Aspect: "career", Status: status, Parent: parent}
}

// A tree used across tests:
//
//	p (open)
//	  c1 (open)
//	    g1 (open)
//	  c2 (done)
//	other (open)
//	closed (done)
var tree = []model.Task{
	task("g1", "c1", model.TaskOpen), // listed before its parent on purpose
	task("p", "", model.TaskOpen),
	task("c1", "p", model.TaskOpen),
	task("c2", "p", model.TaskDone),
	task("other", "", model.TaskOpen),
	task("closed", "", model.TaskDone),
}

func ids(tasks []model.Task) string {
	var out []string
	for _, t := range tasks {
		out = append(out, t.ID)
	}
	return strings.Join(out, ",")
}

func TestOpenDescendants(t *testing.T) {
	tests := []struct{ id, want string }{
		{"p", "g1,c1"}, // any depth, file order; done c2 excluded
		{"c1", "g1"},
		{"g1", ""},
		{"other", ""},
	}
	for _, tt := range tests {
		if got := ids(openDescendants(tree, tt.id)); got != tt.want {
			t.Errorf("openDescendants(%s) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestValidateParent(t *testing.T) {
	tests := []struct {
		child, parent string
		wantErr       string
	}{
		{"", "p", ""},                // new subtask under an open task
		{"other", "g1", ""},          // move under a deep task
		{"", "missing", "no task"},   // parent must exist
		{"", "closed", "is done"},    // parent must be open
		{"p", "p", "itself"},         // not itself
		{"p", "g1", "own subtasks"},  // not its own descendant (loop)
		{"c1", "g1", "own subtasks"}, // direct child too
	}
	for _, tt := range tests {
		err := validateParent(tree, tt.child, tt.parent)
		switch {
		case tt.wantErr == "" && err != nil:
			t.Errorf("validateParent(%q, %q) = %v, want ok", tt.child, tt.parent, err)
		case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
			t.Errorf("validateParent(%q, %q) = %v, want error mentioning %q", tt.child, tt.parent, err, tt.wantErr)
		}
	}
}

func TestPrintTaskTree(t *testing.T) {
	// Only open tasks, as `task list` shows by default: done c2 is hidden.
	var shown []model.Task
	for _, tk := range tree {
		if tk.Status == model.TaskOpen {
			shown = append(shown, tk)
		}
	}
	var out bytes.Buffer
	printTaskTree(&out, shown, nil)

	var got []string
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		got = append(got, strings.SplitN(line, "\t", 2)[0]) // indent + id
	}
	want := []string{"p", "  c1", "    g1", "other"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("tree = %q, want %q", got, want)
	}
}

func TestPrintTaskTreeOrphanShownAtTop(t *testing.T) {
	// c1's parent p is filtered out (e.g. by --aspect): c1 and its own
	// subtask must still show, at the top level.
	var out bytes.Buffer
	printTaskTree(&out, []model.Task{task("c1", "p", model.TaskOpen), task("g1", "c1", model.TaskOpen)}, nil)
	if got := out.String(); !strings.HasPrefix(got, "c1\t") || !strings.Contains(got, "\n  g1\t") {
		t.Errorf("orphaned subtree printed as:\n%s", got)
	}
}

func TestPrintTaskTreeSurvivesParentLoop(t *testing.T) {
	// Hand-edited data with a loop must not hang or drop tasks.
	var out bytes.Buffer
	printTaskTree(&out, []model.Task{task("a", "b", model.TaskOpen), task("b", "a", model.TaskOpen)}, nil)
	if n := strings.Count(out.String(), "\n"); n != 2 {
		t.Errorf("printed %d lines for a 2-task loop, want 2:\n%s", n, out.String())
	}
}
