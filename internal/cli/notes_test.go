package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/model"
)

func TestReadNotes(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(utf8BOM + "https://example.com/doc\r\nread section 3 first\r\n\r\n"))

	got, err := readNotes(cmd, "-")
	if err != nil {
		t.Fatalf("readNotes(-): %v", err)
	}
	if want := "https://example.com/doc\nread section 3 first"; got != want {
		t.Errorf("readNotes(-) = %q, want %q", got, want)
	}

	got, err = readNotes(cmd, "literal")
	if err != nil || got != "literal" {
		t.Errorf("readNotes(literal) = %q, %v; want the value unchanged", got, err)
	}
}

func TestAppendNotes(t *testing.T) {
	tests := []struct{ existing, more, want string }{
		{"", "first", "first"},
		{"first", "second", "first\nsecond"},
		{"a\nb", "c", "a\nb\nc"},
		{"keep", "", "keep"},
	}
	for _, tt := range tests {
		if got := appendNotes(tt.existing, tt.more); got != tt.want {
			t.Errorf("appendNotes(%q, %q) = %q, want %q", tt.existing, tt.more, got, tt.want)
		}
	}
}

func TestNotesPreview(t *testing.T) {
	long := strings.Repeat("x", notesPreviewLen+5)
	tests := []struct{ notes, want string }{
		{"short", "short"},
		{"line one\nline two", "line one..."},
		{"only line\n\n", "only line"}, // trailing blank lines aren't "more"
		{long, strings.Repeat("x", notesPreviewLen) + "..."},
		{strings.Repeat("é", notesPreviewLen), strings.Repeat("é", notesPreviewLen)}, // counts runes, not bytes
	}
	for _, tt := range tests {
		if got := notesPreview(tt.notes); got != tt.want {
			t.Errorf("notesPreview(%q) = %q, want %q", tt.notes, got, tt.want)
		}
	}
}

func TestPrintTaskDetail(t *testing.T) {
	var out bytes.Buffer
	printTaskDetail(&out, model.Task{
		ID:      "t_1",
		Text:    "read the paper",
		Status:  model.TaskOpen,
		Aspect:  "career",
		Created: "2026-09-26",
		Notes:   "https://example.com\nsection 3",
	})
	want := `id:        t_1
text:      read the paper
status:    open
aspect:    career
created:   2026-09-26
notes:
  https://example.com
  section 3
`
	if out.String() != want {
		t.Errorf("printTaskDetail output:\n%s\nwant:\n%s", out.String(), want)
	}
}
