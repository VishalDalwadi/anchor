// Package validate holds small shared validators used before anything is
// written to storage.
package validate

import "time"

const DateLayout = "2006-01-02"

func Today() string {
	return time.Now().Format(DateLayout)
}
