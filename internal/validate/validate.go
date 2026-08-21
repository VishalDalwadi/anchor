// Package validate holds small shared validators used before anything is
// written to storage.
package validate

import (
	"fmt"
	"time"
)

const DateLayout = "2006-01-02"

// Date checks s is a valid YYYY-MM-DD date. Empty strings are allowed
// (callers decide whether the field itself is required).
func Date(s string) error {
	if s == "" {
		return nil
	}
	if _, err := time.Parse(DateLayout, s); err != nil {
		return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", s)
	}
	return nil
}

func Today() string {
	return time.Now().Format(DateLayout)
}
