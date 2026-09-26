package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// notesUsage is the help text for notes flags; "-" reads from stdin so
// multiline notes can be piped in without shell quoting.
func notesUsage(what string) string {
	return what + ` ("-" reads from stdin, for multiline notes)`
}

// utf8BOM is the byte-order mark some tools (notably PowerShell) prepend
// to text piped into a native program.
const utf8BOM = string(rune(0xFEFF))

// readNotes resolves a notes flag value: "-" means read all of stdin,
// anything else is taken literally. Stdin input has any leading BOM
// dropped, Windows line endings normalized, and trailing newlines trimmed,
// so piping a file in doesn't leave a dangling blank line.
func readNotes(cmd *cobra.Command, value string) (string, error) {
	if value != "-" {
		return value, nil
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("reading notes from stdin: %w", err)
	}
	s := strings.TrimPrefix(string(data), utf8BOM) // PowerShell may prepend a BOM when piping
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimRight(s, "\n"), nil
}

// appendNotes adds more on a new line after existing notes.
func appendNotes(existing, more string) string {
	if existing == "" {
		return more
	}
	if more == "" {
		return existing
	}
	return existing + "\n" + more
}

// notesPreviewLen caps the notes preview in list views, in runes.
const notesPreviewLen = 40

// notesPreview is the one-line form of notes for list views: the first
// line, cut to notesPreviewLen runes, with "..." whenever anything was
// left out (further lines, or the rest of a long first line). Full notes
// are only shown by `task show`.
func notesPreview(notes string) string {
	first, rest, multiline := strings.Cut(notes, "\n")
	truncated := multiline && strings.TrimSpace(rest) != ""
	if r := []rune(first); len(r) > notesPreviewLen {
		first = string(r[:notesPreviewLen])
		truncated = true
	}
	first = strings.TrimSpace(first)
	if truncated {
		first += "..."
	}
	return first
}
