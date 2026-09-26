package cli

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Version is anchor's release version. Update it for each release.
const Version = "0.0.1"

// buildInfo is what Go records about how the running binary was built.
type buildInfo struct {
	commit    string // short hash, or "" if not built from a git checkout
	dirty     bool   // built with uncommitted changes
	time      string // commit time, YYYY-MM-DD
	goVersion string
}

func readBuildInfo() buildInfo {
	bi := buildInfo{goVersion: runtime.Version()}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return bi
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			bi.commit = s.Value
			if len(bi.commit) > 7 {
				bi.commit = bi.commit[:7]
			}
		case "vcs.modified":
			bi.dirty = s.Value == "true"
		case "vcs.time":
			bi.time, _, _ = strings.Cut(s.Value, "T")
		}
	}
	return bi
}

// versionString is the one-line version: release, then where it was
// built from, e.g. "anchor 0.0.1 (31f4c14, 2026-09-26) go1.27.0 windows/amd64".
func versionString(v string, bi buildInfo) string {
	var details []string
	if bi.commit != "" {
		commit := bi.commit
		if bi.dirty {
			commit += "-dirty"
		}
		details = append(details, commit)
	}
	if bi.time != "" {
		details = append(details, bi.time)
	}
	s := "anchor " + v
	if len(details) > 0 {
		s += " (" + strings.Join(details, ", ") + ")"
	}
	return fmt.Sprintf("%s %s %s/%s", s, bi.goVersion, runtime.GOOS, runtime.GOARCH)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print anchor's version and build details",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), versionString(Version, readBuildInfo()))
			return nil
		},
	}
}
