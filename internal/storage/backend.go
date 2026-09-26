// Package storage defines the persistence interface used by anchor, and
// the local filesystem backend that satisfies it. No command logic should
// depend on a concrete backend type.
package storage

// Backend is the storage interface every backend implements. All command
// logic operates purely against this interface.
type Backend interface {
	// ReadFile returns the raw bytes at the given logical path
	// (e.g. "tasks.json", "log/2026-08-20.md"). Returns (nil, nil)
	// if the file does not exist yet — callers treat that as empty state,
	// not an error.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes/overwrites the raw bytes at the given logical path.
	// Must create intermediate directories as needed.
	WriteFile(path string, data []byte) error

	// Remove deletes the file, or the whole directory tree, at the given
	// logical path (e.g. "tasks.json", "journal"). A path that doesn't
	// exist is not an error.
	Remove(path string) error
}
