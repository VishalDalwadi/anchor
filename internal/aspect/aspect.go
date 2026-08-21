// Package aspect defines the six fixed life aspects used across tasks,
// goals, and the watchlist.
package aspect

import "fmt"

const (
	Career       = "career"
	Passion      = "passion"
	Physical     = "physical"
	Mental       = "mental"
	Finances     = "finances"
	Relationship = "relationship"
)

// Valid lists the six valid aspect values, in the canonical order.
var Valid = []string{Career, Passion, Physical, Mental, Finances, Relationship}

// Validate returns an error listing the valid options if a is not one of them.
func Validate(a string) error {
	for _, v := range Valid {
		if a == v {
			return nil
		}
	}
	return fmt.Errorf("invalid aspect %q: must be one of %v", a, Valid)
}
