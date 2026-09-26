package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/config"
)

// runAnchor runs anchor with args against a throwaway home directory.
func runAnchor(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	t.Setenv("HOME", home)        // elsewhere
	root := newRootCmd(args)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestConfigSetEditorKeepsEditorFlags(t *testing.T) {
	home := t.TempDir()
	if _, err := runAnchor(t, home, "config", "set", "editor", `C:\Program Files\JetBrains\GoLand\bin\goland64.exe`, "-e", "--wait"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`C:\Program Files\JetBrains\GoLand\bin\goland64.exe`, "-e", "--wait"}
	if strings.Join(c.Editor, "|") != strings.Join(want, "|") {
		t.Errorf("editor = %q, want %q (flags passed through, path kept whole)", c.Editor, want)
	}

	out, err := runAnchor(t, home, "config", "show")
	if err != nil || !strings.Contains(out, `"C:\\Program Files\\JetBrains\\GoLand\\bin\\goland64.exe" -e --wait`) {
		t.Errorf("config show = %q, %v; want the path quoted", out, err)
	}

	if _, err := runAnchor(t, home, "config", "unset", "editor"); err != nil {
		t.Fatal(err)
	}
	if c, _ := config.Load(); len(c.Editor) != 0 {
		t.Errorf("editor after unset = %q, want empty", c.Editor)
	}
}

func TestLookaheadPrecedence(t *testing.T) {
	home := t.TempDir()
	reminderFor := func(args ...string) bool {
		out, err := runAnchor(t, home, append([]string{"brief"}, args...)...)
		if err != nil {
			t.Fatalf("brief %v: %v", args, err)
		}
		return strings.Contains(out, "### Due in 5 days")
	}
	if _, err := runAnchor(t, home, "task", "add", "x", "--aspect", "career", "--due", "5d"); err != nil {
		t.Fatal(err)
	}

	if reminderFor() {
		t.Error("default lookahead (3) reminded about a task due in 5 days")
	}
	if _, err := runAnchor(t, home, "config", "set", "lookahead", "7"); err != nil {
		t.Fatal(err)
	}
	if !reminderFor() {
		t.Error("configured lookahead 7 didn't remind about a task due in 5 days")
	}
	if reminderFor("--lookahead", "2") {
		t.Error("--lookahead 2 didn't override the configured 7")
	}
	if _, err := runAnchor(t, home, "config", "unset", "lookahead"); err != nil {
		t.Fatal(err)
	}
	if reminderFor() {
		t.Error("after unset, lookahead should be back to the default 3")
	}

	for _, bad := range []string{"-1", "soon", "2.5"} {
		if _, err := runAnchor(t, home, "config", "set", "lookahead", bad); err == nil {
			t.Errorf("config set lookahead %s: want an error", bad)
		}
	}
}

func TestConfigRejectsUnknownKeys(t *testing.T) {
	home := t.TempDir()
	for _, args := range [][]string{{"config", "set", "colour", "blue"}, {"config", "unset", "colour"}} {
		if _, err := runAnchor(t, home, args...); err == nil || !strings.Contains(err.Error(), "unknown setting") {
			t.Errorf("%v: err = %v, want unknown setting", args, err)
		}
	}
}

func TestJournalEntryFromPipedStdin(t *testing.T) {
	// Input that isn't a console counts as piped, no "-" needed.
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("piped entry\r\nline two\r\n"))
	got, err := journalEntryText(cmd, nil)
	if err != nil || got != "piped entry\nline two" {
		t.Errorf("journalEntryText(piped) = %q, %v", got, err)
	}
}
