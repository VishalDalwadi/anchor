package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	platform := " go1.27.0 " + runtime.GOOS + "/" + runtime.GOARCH
	tests := []struct {
		name string
		bi   buildInfo
		want string
	}{
		{"from a clean checkout", buildInfo{commit: "31f4c14", time: "2026-09-26", goVersion: "go1.27.0"},
			"anchor 0.0.1 (31f4c14, 2026-09-26)" + platform},
		{"with uncommitted changes", buildInfo{commit: "31f4c14", dirty: true, time: "2026-09-26", goVersion: "go1.27.0"},
			"anchor 0.0.1 (31f4c14-dirty, 2026-09-26)" + platform},
		{"no git info", buildInfo{goVersion: "go1.27.0"},
			"anchor 0.0.1" + platform},
	}
	for _, tt := range tests {
		if got := versionString("0.0.1", tt.bi); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestVersionCommandAndFlag(t *testing.T) {
	home := t.TempDir()
	for _, args := range [][]string{{"version"}, {"--version"}} {
		out, err := runAnchor(t, home, args...)
		if err != nil || !strings.HasPrefix(out, "anchor "+Version) || strings.Count(out, "\n") != 1 {
			t.Errorf("anchor %v = %q, %v; want one line starting \"anchor %s\"", args, out, err, Version)
		}
	}
}
