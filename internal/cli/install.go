package cli

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/selfinstall"
)

// newInstallCmd installs (or replaces) the running binary at a permanent
// location and adds it to PATH. Typical use: `make build` produces a fresh
// binary in bin/, then `bin/anchor install` places it where `anchor` will
// resolve from anywhere.
func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install (or replace) this binary and add it to PATH",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dst, pathUpdated, err := selfinstall.Install()
			if err != nil {
				return err
			}
			fmt.Printf("installed anchor to %s\n", dst)

			switch {
			case pathUpdated:
				fmt.Println("added it to your PATH — open a new terminal for this to take effect")
			case runtime.GOOS == "windows":
				fmt.Println("already on PATH")
			default:
				fmt.Printf("add %s to your PATH manually (e.g. in your shell profile)\n", filepath.Dir(dst))
			}

			switch msg, err := selfinstall.SetupCompletion(); {
			case err != nil:
				fmt.Printf("note: could not set up shell completion: %v\n", err)
			case msg != "":
				fmt.Println(msg)
			default:
				fmt.Println("for shell completion, see: anchor completion --help")
			}

			switch msg, err := selfinstall.SetupRecurTask(dst); {
			case err != nil:
				fmt.Printf("note: could not schedule `recur run`: %v\n", err)
			case msg != "":
				fmt.Println(msg)
			default:
				fmt.Println("no automatic scheduler support on this platform — run `anchor recur run` periodically yourself (e.g. via cron)")
			}
			return nil
		},
	}
}
