// Package cli wires up the anchor command surface. All command logic here
// operates purely against store.Store, which itself operates purely
// against the storage.Backend interface.
package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/storage"
	"github.com/VishalDalwadi/anchor/internal/store"
)

// anchorDir returns ~/.anchor/<parts...>.
func anchorDir(parts ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home, ".anchor"}, parts...)...), nil
}

// openStore builds a Store over the local data directory, ~/.anchor/data.
// Called lazily inside each command's RunE, not at startup, so commands
// that don't touch storage never create it.
func openStore() (*store.Store, error) {
	dir, err := anchorDir("data")
	if err != nil {
		return nil, err
	}
	backend, err := storage.NewLocal(dir)
	if err != nil {
		return nil, err
	}
	return store.New(backend), nil
}

// NewRootCmd builds the anchor root command.
func NewRootCmd() *cobra.Command {
	return newRootCmd(os.Args[1:])
}

// newRootCmd builds the root command for the given command-line args,
// which only matter for telling a completion request from a real run.
func newRootCmd(args []string) *cobra.Command {
	root := &cobra.Command{
		Use:   "anchor",
		Short: "anchor — externalize working memory: tasks, goals, watchlist, recurring tasks",
		// main prints the returned error once as "anchor: <err>"; without
		// these, cobra would also print it (plus the full usage text) first.
		SilenceErrors: true,
		SilenceUsage:  true,
		// Also enables `anchor --version`, printing the same line as `anchor version`.
		Version: versionString(Version, readBuildInfo()),
	}
	root.SetVersionTemplate("{{.Version}}\n")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newGoalCmd())
	root.AddCommand(newWatchCmd())
	root.AddCommand(newRecurCmd())
	root.AddCommand(newBriefCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newFocusCmd())
	root.AddCommand(newJournalCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newCleanCmd())
	addFlagCompletionAfterArgs(root, isCompletionRequest(args))
	addPowerShellCompletionExtras(root)

	return root
}
