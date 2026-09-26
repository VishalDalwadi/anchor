// Package model defines the record shapes persisted by anchor.
package model

import "encoding/json"

// Task statuses.
const (
	TaskOpen    = "open"
	TaskDone    = "done"
	TaskDropped = "dropped"
)

// Goal tiers, laddering upward via an optional parent id.
const (
	TierLife    = "life"
	TierYearly  = "yearly"
	TierMonthly = "monthly"
	TierWeekly  = "weekly"
)

// Goal statuses.
const (
	GoalActive  = "active"
	GoalDone    = "done"
	GoalDropped = "dropped"
)

// WatchItem statuses.
const (
	WatchWaiting  = "waiting"
	WatchResolved = "resolved"
	WatchStalled  = "stalled"
)

// ValidTiers lists the four valid goal tiers, in ladder order.
var ValidTiers = []string{TierLife, TierYearly, TierMonthly, TierWeekly}

type Task struct {
	ID              string `json:"id"`
	Text            string `json:"text"`
	Aspect          string `json:"aspect"`
	Created         string `json:"created"`                    // YYYY-MM-DD
	Due             string `json:"due,omitempty"`              // YYYY-MM-DD
	Status          string `json:"status"`                     // open | done | dropped
	Context         string `json:"context,omitempty"`          // e.g. phone, errand, desk, home
	RecurringSource string `json:"recurring_source,omitempty"` // id of recurring template, if spawned
	Notes           string `json:"notes,omitempty"`            // freeform, may be multiline
	Parent          string `json:"parent,omitempty"`           // id of the parent task, if this is a subtask
	Active          bool   `json:"active"`                     // false = backlogged: open, but not now
	Focus           bool   `json:"focus,omitempty"`            // marked as in focus: listed first
}

// FocusState is the aspect currently in focus, if any (focus.json). An
// aspect isn't a record with an id, so its focus lives here rather than
// on individual tasks and goals. Only one aspect can be in focus at a time.
type FocusState struct {
	Aspect string `json:"aspect,omitempty"` // one of the six aspects, or empty
	Since  string `json:"since,omitempty"`  // YYYY-MM-DD, when focus was set
}

// UnmarshalJSON decodes a Task, treating a missing "active" field as true:
// tasks saved before backlogging existed must stay visible, not all turn
// into backlog.
func (t *Task) UnmarshalJSON(data []byte) error {
	type plain Task // same fields, without this method (no recursion)
	p := plain{Active: true}
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*t = Task(p)
	return nil
}

// Collection is a lightweight named list of freeform text lines ("movies
// to watch"), for things that don't need a task's due dates, aspects or
// status. A collection can sit inside another as a sublist.
type Collection struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`             // unique across all collections, e.g. "movies-to-watch"
	Items  []string `json:"items"`            // freeform text lines, no per-item metadata
	Parent string   `json:"parent,omitempty"` // id of the collection this is a sublist of
}

type Goal struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Aspect string `json:"aspect"`
	Tier   string `json:"tier"`             // life | yearly | monthly | weekly
	Period string `json:"period,omitempty"` // e.g. "2026", "2026-08", "2026-W34"; empty for life tier
	Status string `json:"status"`           // active | done | dropped
	Parent string `json:"parent,omitempty"` // id of the goal one tier up
	Focus  bool   `json:"focus,omitempty"`  // marked as in focus: listed first
}

type WatchItem struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Aspect      string `json:"aspect"`
	Expected    string `json:"expected,omitempty"` // YYYY-MM-DD
	LastChecked string `json:"last_checked"`       // YYYY-MM-DD
	Status      string `json:"status"`             // waiting | resolved | stalled
	Notes       string `json:"notes,omitempty"`
}

type RecurringTemplate struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Aspect      string `json:"aspect"`
	Cron        string `json:"cron"` // standard 5-field cron expression
	Context     string `json:"context,omitempty"`
	LastSpawned string `json:"last_spawned,omitempty"` // YYYY-MM-DD, prevents double-spawn same period
	OnMiss      string `json:"on_miss,omitempty"`      // expire (default) | persist
	Until       string `json:"until,omitempty"`        // YYYY-MM-DD, last day an instance may spawn; empty = forever
}

// RecurringTemplate on_miss values: what happens to a spawned task that
// isn't done by the time its period has passed.
const (
	// OnMissExpire drops the unfinished instance when the next one spawns,
	// and doesn't catch up on days recur run missed: habits like a daily
	// workout, which aren't meant to be caught up on.
	OnMissExpire = "expire"
	// OnMissPersist keeps every unfinished instance open (and overdue)
	// until it's done, and catches up on days recur run missed: fixed
	// obligations like tax installments, where missing one doesn't cancel
	// it.
	OnMissPersist = "persist"
)

// ValidOnMiss lists the valid on_miss values.
var ValidOnMiss = []string{OnMissExpire, OnMissPersist}

// Persists reports whether r keeps unfinished instances around. An empty
// on_miss (templates from before it existed) means expire.
func (r RecurringTemplate) Persists() bool { return r.OnMiss == OnMissPersist }
