package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestHelperEditor isn't a real test: editText tests run the test binary
// itself as a stand-in editor, and this is what it does then — write
// ANCHOR_FAKE_EDITOR_TEXT into the file it's given (the last argument).
func TestHelperEditor(t *testing.T) {
	text, ok := os.LookupEnv("ANCHOR_FAKE_EDITOR_TEXT")
	if !ok {
		return
	}
	if text != "" {
		os.WriteFile(os.Args[len(os.Args)-1], []byte(text), 0o644)
	}
	os.Exit(0)
}

func fakeEditor(t *testing.T, writes string) []string {
	t.Setenv("ANCHOR_FAKE_EDITOR_TEXT", writes)
	return []string{os.Args[0], "-test.run=^TestHelperEditor$"}
}

func TestEditTextReadsWhatTheEditorSaved(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	got, err := editText(cmd, fakeEditor(t, utf8BOM+"from the editor\r\n"), "")
	if err != nil || got != "from the editor\r\n" {
		t.Errorf("editText = %q, %v; want the saved text, BOM dropped", got, err)
	}
}

func TestEditTextStartsFromCurrentText(t *testing.T) {
	// Replacing notes: the editor opens with the current ones in it. Here
	// it's closed without saving, so they come back unchanged.
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("\n"))
	got, err := editText(cmd, fakeEditor(t, ""), "current notes\nline two")
	if err != nil || normalizeText(got) != "current notes\nline two" {
		t.Errorf("editText = %q, %v; want the current notes back", got, err)
	}
}

func TestEditTextWaitsWhenEditorReturnsEarly(t *testing.T) {
	// The editor exits having saved nothing (like GoLand without --wait):
	// anchor asks, and pressing Enter with still nothing saved cancels.
	cmd := &cobra.Command{}
	var prompt bytes.Buffer
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&prompt)
	cmd.SetIn(strings.NewReader("\n"))
	got, err := editText(cmd, fakeEditor(t, ""), "")
	if err != nil || got != "" {
		t.Errorf("editText = %q, %v; want empty (cancelled)", got, err)
	}
	if !strings.Contains(prompt.String(), "Nothing saved yet") {
		t.Errorf("no prompt to finish in the editor; stderr = %q", prompt.String())
	}
}

func clock(hhmm string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", "2026-09-26 "+hhmm)
	if err != nil {
		panic(err)
	}
	return t
}

func TestJournalTimeline(t *testing.T) {
	day := ""
	day = appendJournalEntry(day, "2026-09-26", clock("09:05"), "woke up early")
	day = appendJournalEntry(day, "2026-09-26", clock("14:32"), "https://example.com\ngood read on focus")
	day = appendReviewMarker(day, "2026-09-26", clock("22:10"))

	want := `# Journal — 2026-09-26

## 09:05

woke up early

## 14:32

https://example.com
good read on focus

--- reviewed at 22:10 ---
`
	if day != want {
		t.Errorf("journal:\n%s\nwant:\n%s", day, want)
	}
}

func TestJournalAppendsToHandEditedFile(t *testing.T) {
	// A file edited by hand, without a trailing newline, still gets a
	// clean blank-line separation.
	got := appendJournalEntry("# my own title\nsome notes", "2026-09-26", clock("10:00"), "more")
	want := "# my own title\nsome notes\n\n## 10:00\n\nmore\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestJournalCompleteOnEmptyDay(t *testing.T) {
	got := appendReviewMarker("", "2026-09-26", clock("23:00"))
	want := "# Journal — 2026-09-26\n\n--- reviewed at 23:00 ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestJournalDate(t *testing.T) {
	if d, err := journalDate("2026-09-20"); err != nil || d != "2026-09-20" {
		t.Errorf("journalDate(2026-09-20) = %q, %v", d, err)
	}
	if _, err := journalDate("yesterday"); err == nil {
		t.Error("journalDate(yesterday): want an error")
	}
	if d, err := journalDate(""); err != nil || len(d) != len("2026-09-26") {
		t.Errorf("journalDate(\"\") = %q, %v; want today", d, err)
	}
}
