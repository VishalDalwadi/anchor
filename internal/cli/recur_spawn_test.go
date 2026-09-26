package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/VishalDalwadi/anchor/internal/model"
)

func at(date string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		panic(err)
	}
	return t.Add(9 * time.Hour) // recur run fires in the morning
}

// instances returns "<due>:<status>" for each task spawned from template id.
func instances(tasks []model.Task, id string) string {
	var out []string
	for _, t := range tasks {
		if t.RecurringSource == id {
			out = append(out, t.Due+":"+t.Status)
		}
	}
	return strings.Join(out, " ")
}

func TestSpawnPersistKeepsEveryInstanceOpen(t *testing.T) {
	// Quarterly advance tax, June's installment never paid.
	templates := []model.RecurringTemplate{{ID: "r_tax", Text: "advance tax", Aspect: "finances", Cron: "0 9 15 6,9,12,3 *", OnMiss: model.OnMissPersist}}

	tasks, res, err := spawnRecurring(templates, nil, at("2026-06-15"))
	if err != nil || res.spawned != 1 {
		t.Fatalf("June run: spawned %d, err %v", res.spawned, err)
	}
	tasks, res, err = spawnRecurring(templates, tasks, at("2026-09-15"))
	if err != nil || res.spawned != 1 || res.dropped != 0 {
		t.Fatalf("September run: %+v, err %v", res, err)
	}
	if got, want := instances(tasks, "r_tax"), "2026-06-15:open 2026-09-15:open"; got != want {
		t.Errorf("instances = %q, want %q (June kept open alongside September)", got, want)
	}
}

func TestSpawnPersistCatchesUpMissedRuns(t *testing.T) {
	templates := []model.RecurringTemplate{{ID: "r_gst", Text: "GST return", Aspect: "finances", Cron: "0 9 20 * *", OnMiss: model.OnMissPersist, LastSpawned: "2026-06-20"}}
	// recur run didn't run for three months (machine off, etc.).
	tasks, res, err := spawnRecurring(templates, nil, at("2026-09-26"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := instances(tasks, "r_gst"), "2026-07-20:open 2026-08-20:open 2026-09-20:open"; got != want {
		t.Errorf("instances = %q, want %q (one per missed month, each due that month)", got, want)
	}
	if res.spawned != 3 || templates[0].LastSpawned != "2026-09-20" {
		t.Errorf("spawned %d, last_spawned %q; want 3 and the latest fire day", res.spawned, templates[0].LastSpawned)
	}
}

func TestSpawnExpireDropsUnfinishedAndSkipsMissedRuns(t *testing.T) {
	// Daily gym; OnMiss empty (pre-on_miss template) means expire.
	templates := []model.RecurringTemplate{{ID: "r_gym", Text: "gym", Aspect: "physical", Cron: "0 6 * * *"}}

	tasks, _, _ := spawnRecurring(templates, nil, at("2026-09-20"))
	// Skipped the gym, and recur run didn't run 21st–25th.
	tasks, res, err := spawnRecurring(templates, tasks, at("2026-09-26"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := instances(tasks, "r_gym"), "2026-09-20:dropped 2026-09-26:open"; got != want {
		t.Errorf("instances = %q, want %q (old one dropped, no catch-up)", got, want)
	}
	if res.spawned != 1 || res.dropped != 1 {
		t.Errorf("result = %+v, want 1 spawned, 1 dropped", res)
	}
}

func TestSpawnIsIdempotent(t *testing.T) {
	for _, onMiss := range model.ValidOnMiss {
		templates := []model.RecurringTemplate{{ID: "r", Text: "x", Aspect: "career", Cron: "0 9 * * *", OnMiss: onMiss}}
		tasks, _, _ := spawnRecurring(templates, nil, at("2026-09-26"))
		tasks, res, err := spawnRecurring(templates, tasks, at("2026-09-26").Add(5*time.Hour))
		if err != nil || res.spawned != 0 || res.dropped != 0 {
			t.Errorf("%s: second run the same day = %+v, %v; want nothing", onMiss, res, err)
		}
		if n := len(tasks); n != 1 {
			t.Errorf("%s: %d tasks after two runs, want 1", onMiss, n)
		}
	}
}

func TestSpawnNewPersistTemplateDoesNotBackfill(t *testing.T) {
	templates := []model.RecurringTemplate{{ID: "r", Text: "x", Aspect: "career", Cron: "0 9 15 * *", OnMiss: model.OnMissPersist}}
	tasks, res, _ := spawnRecurring(templates, nil, at("2026-09-26"))
	if res.spawned != 0 || len(tasks) != 0 {
		t.Errorf("a brand-new template spawned %d tasks for past fire days, want 0", res.spawned)
	}
}

func TestSpawnStopsAfterUntil(t *testing.T) {
	// Persist, catching up across its until date: Sep 20 is after the end.
	persist := []model.RecurringTemplate{{ID: "r_emi", Text: "EMI", Aspect: "finances", Cron: "0 9 20 * *",
		OnMiss: model.OnMissPersist, LastSpawned: "2026-06-20", Until: "2026-08-31"}}
	tasks, _, err := spawnRecurring(persist, nil, at("2026-09-26"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := instances(tasks, "r_emi"), "2026-07-20:open 2026-08-20:open"; got != want {
		t.Errorf("persist instances = %q, want %q (nothing past until)", got, want)
	}

	// The until day itself still spawns.
	daily := []model.RecurringTemplate{{ID: "r_d", Text: "x", Aspect: "career", Cron: "0 6 * * *", Until: "2026-09-26"}}
	if _, res, _ := spawnRecurring(daily, nil, at("2026-09-26")); res.spawned != 1 {
		t.Errorf("spawned %d on the until day itself, want 1", res.spawned)
	}
}

func TestEndedTemplateLeavesExistingTasksAlone(t *testing.T) {
	for _, onMiss := range model.ValidOnMiss {
		// Ended on the day of its last spawn: no fire days left to catch up.
		templates := []model.RecurringTemplate{{ID: "r", Text: "x", Aspect: "career", Cron: "0 6 * * *",
			OnMiss: onMiss, LastSpawned: "2026-09-20", Until: "2026-09-20"}}
		existing := []model.Task{{ID: "t_old", RecurringSource: "r", Due: "2026-09-20", Status: model.TaskOpen}}

		tasks, res, err := spawnRecurring(templates, existing, at("2026-09-26"))
		if err != nil {
			t.Fatal(err)
		}
		if res.spawned != 0 || res.dropped != 0 {
			t.Errorf("%s: ended template did %+v, want nothing", onMiss, res)
		}
		if len(tasks) != 1 || tasks[0].Status != model.TaskOpen {
			t.Errorf("%s: existing instance became %+v, want untouched and open", onMiss, tasks)
		}
	}
}

func TestStillOwed(t *testing.T) {
	templates := templatesByID([]model.RecurringTemplate{
		{ID: "r_tax", Cron: "0 9 15 6,9,12,3 *", OnMiss: model.OnMissPersist},
		{ID: "r_gym", Cron: "0 6 * * *"},
		{ID: "r_ended", Cron: "0 9 15 6,9,12,3 *", OnMiss: model.OnMissPersist, Until: "2026-08-31"},
	})
	today := at("2026-09-26")
	tests := []struct {
		name string
		task model.Task
		want string
	}{
		{"persisted, a later period already due", model.Task{RecurringSource: "r_tax", Due: "2026-06-15", Status: model.TaskOpen},
			"[!] still owed: 103 day(s) overdue, 1 later period(s) already due"},
		{"persisted, overdue within its period", model.Task{RecurringSource: "r_tax", Due: "2026-09-15", Status: model.TaskOpen},
			"[!] still owed: 11 day(s) overdue"},
		{"persisted, not yet due", model.Task{RecurringSource: "r_tax", Due: "2026-12-15", Status: model.TaskOpen}, ""},
		{"persisted, due today", model.Task{RecurringSource: "r_tax", Due: "2026-09-26", Status: model.TaskOpen}, ""},
		{"persisted but done", model.Task{RecurringSource: "r_tax", Due: "2026-06-15", Status: model.TaskDone}, ""},
		{"expire template", model.Task{RecurringSource: "r_gym", Due: "2026-09-20", Status: model.TaskOpen}, ""},
		{"ended template: still owed, but no periods counted past its end", model.Task{RecurringSource: "r_ended", Due: "2026-06-15", Status: model.TaskOpen},
			"[!] still owed: 103 day(s) overdue"},
		{"template removed", model.Task{RecurringSource: "r_gone", Due: "2026-06-15", Status: model.TaskOpen}, ""},
		{"ordinary overdue task", model.Task{Due: "2026-06-15", Status: model.TaskOpen}, ""},
	}
	for _, tt := range tests {
		if got := stillOwed(tt.task, templates, today); got != tt.want {
			t.Errorf("%s: stillOwed = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSpawnDueInOffsetsDueFromFireDay(t *testing.T) {
	// Weekly report spawned Monday, due Friday; one missed run caught up.
	templates := []model.RecurringTemplate{{ID: "r_rep", Text: "weekly report", Aspect: "career", Cron: "0 9 * * 1", OnMiss: model.OnMissPersist, DueIn: "4d", LastSpawned: "2026-09-07"}}
	tasks, _, err := spawnRecurring(templates, nil, at("2026-09-21"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := instances(tasks, "r_rep"), "2026-09-18:open 2026-09-25:open"; got != want {
		t.Errorf("instances = %q, want %q (each due 4 days after its Monday)", got, want)
	}
	if templates[0].LastSpawned != "2026-09-21" {
		t.Errorf("last_spawned = %q, want the fire day, not the due day", templates[0].LastSpawned)
	}
}

func TestSpawnRejectsBadDueIn(t *testing.T) {
	templates := []model.RecurringTemplate{{ID: "r_bad", Text: "x", Aspect: "physical", Cron: "0 9 * * *", DueIn: "2026-10-01"}}
	if _, _, err := spawnRecurring(templates, nil, at("2026-09-21")); err == nil {
		t.Error("expected an error for an absolute due_in")
	}
}
