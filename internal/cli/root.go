// Package cli wires up the anchor command surface. All command logic here
// operates purely against store.Store, which itself operates purely
// against the storage.Backend interface — no command knows which concrete
// backend (local, Dropbox) is active.
package cli

import (
	"fmt"

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
	root := &cobra.Command{
		Use:   "anchor",
		Short: "anchor — externalize working memory: tasks, goals, watchlist, recurring tasks",
	}

	root.AddCommand(newTaskCmd())
	root.AddCommand(newGoalCmd())
	root.AddCommand(newWatchCmd())
	root.AddCommand(newRecurCmd())
	root.AddCommand(newBriefCmd())
	root.AddCommand(newInstallCmd())

	return root
}
