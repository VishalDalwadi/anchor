// Package idgen generates short, sortable record IDs.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// New returns an id of the form "<prefix>_YYYYMMDD-xxxx", e.g.
// "t_20260820-a1b2".
func New(prefix string) string {
	buf := make([]byte, 2)
	_, _ = rand.Read(buf) // crypto/rand.Read never errors on the platforms Go supports
	return prefix + "_" + time.Now().Format("20060102") + "-" + hex.EncodeToString(buf)
}
