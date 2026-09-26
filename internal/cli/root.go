// Package cli wires up the anchor command surface. All command logic here
// operates purely against store.Store, which itself operates purely
// against the storage.Backend interface — no command knows which concrete
// backend (local, Dropbox) is active.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/config"
	"github.com/VishalDalwadi/anchor/internal/store"
)

// openStore loads config and builds a Store against the active backend.
// Called lazily inside each command's RunE, not at startup, so that a
// misconfigured backend only breaks commands that actually touch storage.
func openStore() (*store.Store, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	backend, err := config.NewBackend(cfg)
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
	}

	root.AddCommand(newTaskCmd())
	root.AddCommand(newGoalCmd())
	root.AddCommand(newWatchCmd())
	root.AddCommand(newRecurCmd())
	root.AddCommand(newBriefCmd())
	root.AddCommand(newInstallCmd())
	addFlagCompletionAfterArgs(root, isCompletionRequest(args))
	addPowerShellCompletionExtras(root)

	return root
}
