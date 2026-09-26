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
