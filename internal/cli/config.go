package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/config"
)

// configKeys lists the settings `anchor config` can change, with help text.
var configKeys = []struct{ name, help string }{
	{"editor", "command to write text in, e.g. `goland -e --wait` (the file is appended last)"},
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show and change global settings (~/.anchor/config.json)",
	}
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigUnsetCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show all settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return err
			}
			editor := "(not set: type into the terminal)"
			if len(c.Editor) > 0 {
				editor = formatCommand(c.Editor)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "editor  %s\n", editor)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	var keyHelp []string
	for _, k := range configKeys {
		keyHelp = append(keyHelp, fmt.Sprintf("  %-8s %s", k.name, k.help))
	}
	return &cobra.Command{
		Use:   "set <key> <value...>",
		Short: "Change a setting",
		Long: "Change a setting. Keys:\n\n" + strings.Join(keyHelp, "\n") + `

Everything after the key is the value, flags included, so an editor's own
options pass straight through:

  anchor config set editor goland -e --wait
  anchor config set editor "C:\Program Files\Some Editor\editor.exe" --wait`,
		// The value may contain flags meant for the editor (-e, --wait);
		// don't let cobra claim them.
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		ValidArgsFunction:  completeConfigKey,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			if len(args) < 2 {
				return fmt.Errorf("usage: anchor config set <key> <value...>")
			}
			c, err := config.Load()
			if err != nil {
				return err
			}
			switch args[0] {
			case "editor":
				c.Editor = args[1:]
			default:
				return unknownConfigKey(args[0])
			}
			return c.Save()
		},
	}
}

func newConfigUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "unset <key>",
		Short:             "Reset a setting to its default",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeConfigKey,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return err
			}
			switch args[0] {
			case "editor":
				c.Editor = nil
			default:
				return unknownConfigKey(args[0])
			}
			return c.Save()
		},
	}
}

func unknownConfigKey(key string) error {
	var names []string
	for _, k := range configKeys {
		names = append(names, k.name)
	}
	return fmt.Errorf("unknown setting %q: must be one of %v", key, names)
}

// completeConfigKey completes the <key> argument; values are free-form.
func completeConfigKey(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, k := range configKeys {
		out = append(out, k.name+"\t"+k.help)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// formatCommand renders argv for display, quoting any part with spaces.
func formatCommand(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		if strings.ContainsAny(a, " \t") || a == "" {
			a = strconv.Quote(a)
		}
		parts[i] = a
	}
	return strings.Join(parts, " ")
}
