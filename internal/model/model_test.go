package model

import (
	"encoding/json"
	"testing"
)

func TestTaskActiveDefaultsToTrue(t *testing.T) {
	var tasks []Task
	data := `[
		{"id": "t_old", "status": "open"},
		{"id": "t_active", "status": "open", "active": true},
		{"id": "t_backlog", "status": "open", "active": false}
	]`
	if err := json.Unmarshal([]byte(data), &tasks); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"t_old": true, "t_active": true, "t_backlog": false}
	for _, tk := range tasks {
		if tk.Active != want[tk.ID] {
			t.Errorf("%s: Active = %v, want %v", tk.ID, tk.Active, want[tk.ID])
		}
	}

	// And it round-trips: a backlogged task stays backlogged.
	out, err := json.Marshal(tasks[2])
	if err != nil {
		t.Fatal(err)
	}
	var back Task
	if err := json.Unmarshal(out, &back); err != nil || back.Active {
		t.Errorf("round-tripped backlog task: Active = %v, err %v; want false", back.Active, err)
	}
}
