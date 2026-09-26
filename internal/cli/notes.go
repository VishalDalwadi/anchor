package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// notesUsage is the help text for notes flags; "-" is for multiline notes.
func notesUsage(what string) string {
	return what + ` ("-": piped input, else your configured editor, else type it in)`
}

// readNotes resolves a notes flag value: "-" means multiline text from
// readTextInput, with the editor (if configured) starting from current —
// the notes being replaced, or "" for new or appended notes. Anything
// else is taken literally.
func readNotes(cmd *cobra.Command, value, current string) (string, error) {
	if value != "-" {
		return value, nil
	}
	return readTextInput(cmd, current)
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
