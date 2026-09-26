package cli

import (
	"bytes"
	"strings"
	"testing"
)

// complete runs a shell completion request for args (the last of which is
// the word being completed) and returns the candidate names.
func complete(t *testing.T, args ...string) []string {
	t.Helper()
	full := append([]string{"__complete"}, args...)
	root := newRootCmd(full)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(full)
	if err := root.Execute(); err != nil {
		t.Fatalf("__complete %v: %v", args, err)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if strings.HasPrefix(line, ":") {
			break // directive line ends the candidates
		}
		names = append(names, strings.SplitN(line, "\t", 2)[0])
	}
	return names
}

func TestEmptyWordCompletion(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		// Positional filled in: flags.
		{[]string{"task", "edit", "t_x", ""}, []string{"--append-notes", "--aspect", "--context", "--due", "--help", "--notes", "--text"}},
		{[]string{"task", "add", "foo", ""}, []string{"--aspect", "--context", "--due", "--help", "--notes"}},
		// Flags already given aren't offered again.
		{[]string{"task", "add", "foo", "--aspect", "work", ""}, []string{"--context", "--due", "--help", "--notes"}},
		// Nor are flags mutually exclusive with one already given.
		{[]string{"task", "edit", "t_x", "--notes", "n", ""}, []string{"--aspect", "--context", "--due", "--help", "--text"}},
		// No positional args at all: flags straight away.
		{[]string{"task", "list", ""}, []string{"--all", "--aspect", "--context", "--help", "--today"}},
		// Free text expected: nothing, not even the required --aspect.
		{[]string{"task", "add", ""}, nil},
	}
	for _, tt := range tests {
		got := complete(t, tt.args...)
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("complete %q = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestRequiredFlagsStillEnforced(t *testing.T) {
	args := []string{"task", "add", "foo"}
	root := newRootCmd(args)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "aspect" not set`) {
		t.Errorf("task add without --aspect: err = %v, want required-flag error", err)
	}
}

func TestPowerShellCompletionIncludesTabHandler(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"completion", "powershell"})
	if err := root.Execute(); err != nil {
		t.Fatalf("completion powershell: %v", err)
	}

	script := out.String()
	if !strings.Contains(script, "Register-ArgumentCompleter") {
		t.Errorf("cobra's own completion script is missing from the output")
	}
	if !strings.Contains(script, "-BriefDescription 'AnchorTab'") {
		t.Errorf("output does not install the anchor Tab handler")
	}
	if !strings.Contains(script, "function global:TabExpansion2") {
		t.Errorf("output does not wrap TabExpansion2 for bare-dash flag completion")
	}
	for _, extra := range []string{"AnchorTab", "function global:TabExpansion2"} {
		if strings.Index(script, extra) < strings.Index(script, "Register-ArgumentCompleter") {
			t.Errorf("%q should come after cobra's script, not replace or precede it", extra)
		}
	}
}
