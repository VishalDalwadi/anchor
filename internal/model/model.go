// Package model defines the record shapes persisted by anchor.
package model

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
}

type Goal struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Aspect string `json:"aspect"`
	Tier   string `json:"tier"`             // life | yearly | monthly | weekly
	Period string `json:"period,omitempty"` // e.g. "2026", "2026-08", "2026-W34"; empty for life tier
	Status string `json:"status"`           // active | done | dropped
	Parent string `json:"parent,omitempty"` // id of the goal one tier up
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
}
