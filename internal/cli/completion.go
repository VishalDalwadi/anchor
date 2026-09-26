package cli

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/model"
)

// powershellCompletionExtras is appended to cobra's generated PowerShell
// completion script to paper over two PowerShell behaviors.
//
// Bare "-" / "--": PowerShell treats a lone dash word as the start of a
// parameter name and does its own parameter-name completion instead of
// calling the native argument completer cobra registers, so typing "--"
// then Tab lists nothing (while "--d" works). TabExpansion2 — the function
// every completion goes through — is wrapped so that when it finds nothing
// for a bare dash word on an anchor line, it asks cobra's completer
// directly. Guarded against wrapping twice if the script is re-sourced.
//
// Tab: makes Tab show the full candidate menu for anchor
// command lines. PSReadLine's default Tab binding on Windows is
// TabCompleteNext, which cycles through candidates one at a time and
// hides their descriptions — useless for picking an id out of a list.
// Rebinding Tab to MenuComplete globally would change every other
// command's completion too, so instead Tab dispatches on the line: anchor
// lines get MenuComplete, everything else keeps whatever Tab was bound to
// before. Skipped when Tab is already MenuComplete (nothing to fix) or
// already this handler (the script was sourced twice).
const powershellCompletionExtras = `
# anchor: complete flags for a bare "-" or "--", which PowerShell never
# hands to native argument completers on its own.
$global:__anchorCompleter = ${__anchorCompleterBlock}
if (-not (Test-Path Function:\__anchorOrigTabExpansion2)) {
    ${function:global:__anchorOrigTabExpansion2} = ${function:TabExpansion2}
    function global:TabExpansion2 {
        [CmdletBinding(DefaultParameterSetName = 'ScriptInputSet')]
        param(
            [Parameter(ParameterSetName = 'ScriptInputSet', Mandatory = $true, Position = 0)]
            [string] $inputScript,
            [Parameter(ParameterSetName = 'ScriptInputSet', Position = 1)]
            [int] $cursorColumn = $inputScript.Length,
            [Parameter(ParameterSetName = 'AstInputSet', Mandatory = $true, Position = 0)]
            [System.Management.Automation.Language.Ast] $ast,
            [Parameter(ParameterSetName = 'AstInputSet', Mandatory = $true, Position = 1)]
            [System.Management.Automation.Language.Token[]] $tokens,
            [Parameter(ParameterSetName = 'AstInputSet', Mandatory = $true, Position = 2)]
            [System.Management.Automation.Language.IScriptPosition] $positionOfCursor,
            [Parameter(ParameterSetName = 'ScriptInputSet', Position = 2)]
            [Parameter(ParameterSetName = 'AstInputSet', Position = 3)]
            [Hashtable] $options = $null
        )
        $result = __anchorOrigTabExpansion2 @PSBoundParameters
        if ($PSCmdlet.ParameterSetName -ne 'ScriptInputSet' -or $result.CompletionMatches.Count -gt 0) {
            return $result
        }

        $parsed = [System.Management.Automation.Language.Parser]::ParseInput($inputScript, [ref]$null, [ref]$null)
        $cmd = $parsed.FindAll({
            $args[0] -is [System.Management.Automation.Language.CommandAst] -and
            $args[0].Extent.StartOffset -le $cursorColumn -and $args[0].Extent.EndOffset -ge $cursorColumn
        }, $true) | Select-Object -Last 1
        if (-not $cmd -or $cmd.GetCommandName() -notmatch '^(.*[\\/])?anchor(\.exe)?$') {
            return $result
        }
        $word = $cmd.CommandElements | Where-Object {
            $_.Extent.EndOffset -eq $cursorColumn -and ($_.Extent.Text -eq '-' -or $_.Extent.Text -eq '--')
        } | Select-Object -First 1
        if (-not $word) {
            return $result
        }

        $found = [System.Collections.ObjectModel.Collection[System.Management.Automation.CompletionResult]]::new()
        foreach ($m in (& $global:__anchorCompleter $word.Extent.Text $cmd $cursorColumn)) {
            if ($m -is [System.Management.Automation.CompletionResult]) {
                $found.Add($m)
            } elseif ("$m") {
                $found.Add([System.Management.Automation.CompletionResult]::new("$m"))
            }
        }
        if ($found.Count -eq 0) {
            return $result
        }
        return [System.Management.Automation.CommandCompletion]::new($found, -1, $word.Extent.StartOffset, $word.Extent.Text.Length)
    }
}

# anchor: menu completion on Tab for anchor command lines only.
if (Get-Module PSReadLine) {
    $__anchorPrevTab = (Get-PSReadLineKeyHandler | Where-Object { $_.Key -eq 'Tab' }).Function
    if ($__anchorPrevTab -ne 'MenuComplete' -and $__anchorPrevTab -ne 'AnchorTab') {
        $__anchorFallback = $null
        try { $__anchorFallback = [Microsoft.PowerShell.PSConsoleReadLine].GetMethod($__anchorPrevTab) } catch {}
        if (-not $__anchorFallback) {
            $__anchorFallback = [Microsoft.PowerShell.PSConsoleReadLine].GetMethod('TabCompleteNext')
        }
        Set-PSReadLineKeyHandler -Key Tab -BriefDescription 'AnchorTab' -Description 'MenuComplete for anchor commands, previous Tab behavior otherwise' -ScriptBlock ({
            param($key, $arg)
            $line = $null
            $cursor = $null
            [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
            if ($line -match '^\s*(\S*[\\/])?anchor(\.exe)?\s') {
                [Microsoft.PowerShell.PSConsoleReadLine]::MenuComplete($key, $arg)
            } else {
                $__anchorFallback.Invoke($null, @($key, $arg))
            }
        }.GetNewClosure())
    }
    Remove-Variable __anchorPrevTab, __anchorFallback -ErrorAction SilentlyContinue
}
`

// addPowerShellCompletionExtras appends powershellCompletionExtras to the output of
// cobra's built-in `completion powershell` command. Hooking the generated
// script (rather than the profile line `anchor install` writes) means
// existing installs pick the fix up on their next shell start, since
// their profile already sources this command's output.
func addPowerShellCompletionExtras(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	for _, c := range root.Commands() {
		if c.Name() != "completion" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "powershell" {
				continue
			}
			// Regenerate cobra's script here rather than calling the
			// original RunE: that one writes to the output stream captured
			// when the command was built, not cmd.OutOrStdout().
			sub.RunE = func(cmd *cobra.Command, args []string) error {
				out := cmd.OutOrStdout()
				gen := cmd.Root().GenPowerShellCompletionWithDesc
				if noDesc, _ := cmd.Flags().GetBool("no-descriptions"); noDesc {
					gen = cmd.Root().GenPowerShellCompletion
				}
				if err := gen(out); err != nil {
					return err
				}
				_, err := io.WriteString(out, powershellCompletionExtras)
				return err
			}
		}
	}
}

// isCompletionRequest reports whether args (os.Args[1:]) is a shell asking
// cobra for completions rather than a real invocation.
func isCompletionRequest(args []string) bool {
	return len(args) > 0 && (args[0] == cobra.ShellCompRequestCmd || args[0] == cobra.ShellCompNoDescRequestCmd)
}

// addFlagCompletionAfterArgs makes Tab on an empty word suggest a leaf
// command's flags once its positional args are filled in (and right away
// for commands that take none), while still suggesting only ids — or
// nothing, for free text — where a positional arg is still expected.
//
// Cobra by itself only suggests flags for a word starting with "-", except
// that it always tacks unset required flags onto every completion, even
// where free text is expected: `task add <Tab>` would offer (and, in a
// one-item menu, insert) --aspect before the task text. So for completion
// requests only, the required-flag markers are stripped, leaving these
// ValidArgsFunctions fully in charge. Real invocations keep them, so
// required-flag validation is unaffected.
func addFlagCompletionAfterArgs(root *cobra.Command, completing bool) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			walk(sub)
		}
		if c.HasSubCommands() || c.Hidden {
			return
		}
		if completing {
			c.Flags().VisitAll(func(f *pflag.Flag) {
				delete(f.Annotations, cobra.BashCompOneRequiredFlag)
			})
		}
		positional := c.ValidArgsFunction
		c.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if cmd.ValidateArgs(append(args, toComplete)) == nil {
				if positional == nil {
					return nil, cobra.ShellCompDirectiveNoFileComp // free text expected
				}
				return positional(cmd, args, toComplete)
			}
			return flagCompletions(cmd), cobra.ShellCompDirectiveNoFileComp
		}
	}
	walk(root)
}

// flagCompletions lists cmd's visible flags that haven't been given yet,
// as "--name<TAB>usage" candidates.
func flagCompletions(cmd *cobra.Command) []string {
	var out []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Changed {
			return
		}
		out = append(out, "--"+f.Name+"\t"+f.Usage)
	})
	return out
}

// completeAspect completes --aspect with the six fixed life aspects.
func completeAspect(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return aspect.Valid, cobra.ShellCompDirectiveNoFileComp
}

// completeTier completes --tier with the four goal tiers.
func completeTier(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return model.ValidTiers, cobra.ShellCompDirectiveNoFileComp
}

// registerAspectFlag wires --aspect completion onto cmd. Call after the
// flag has been defined.
func registerAspectFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("aspect", completeAspect)
}

// registerTierFlag wires --tier completion onto cmd.
func registerTierFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("tier", completeTier)
}

// registerParentFlag wires --parent completion onto cmd with goal IDs.
func registerParentFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("parent", completeGoalIDs)
}

// registerTaskParentFlag wires --parent completion onto a task command
// with the IDs of open tasks.
func registerTaskParentFlag(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("parent", completeOpenTaskIDs)
}

// idCompletion formats a completion candidate as "<id>\t(<text>)": the id
// is what actually gets inserted as the argument, and the parenthesized
// text is the description shells display alongside it (so the id doesn't
// have to be memorized to tell entries apart).
func idCompletion(id, text string) string {
	return id + "\t(" + text + ")"
}

// The id completers below serve both positional <id> arguments and flags
// that take an id (like --parent). They don't check which positional
// argument is being completed: addFlagCompletionAfterArgs only calls a
// command's ValidArgsFunction while a positional arg is still expected.

// completeTaskIDs completes a task <id> argument.
func completeTaskIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return taskIDCompletions(func(model.Task) bool { return true })
}

// completeOpenTaskIDs completes --parent for tasks: only open tasks can
// take subtasks.
func completeOpenTaskIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return taskIDCompletions(func(t model.Task) bool { return t.Status == model.TaskOpen })
}

func taskIDCompletions(keep func(model.Task) bool) ([]string, cobra.ShellCompDirective) {
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	tasks, err := s.LoadTasks()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if keep(t) {
			out = append(out, idCompletion(t.ID, t.Text))
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeGoalIDs completes a goal <id> positional argument (and the
// --parent flag, which also expects a goal ID).
func completeGoalIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	goals, err := s.LoadGoals()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(goals))
	for _, g := range goals {
		out = append(out, idCompletion(g.ID, g.Text))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeWatchIDs completes a watchlist <id> positional argument.
func completeWatchIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	items, err := s.LoadWatchItems()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(items))
	for _, w := range items {
		out = append(out, idCompletion(w.ID, w.Text))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeRecurringIDs completes a recurring template <id> positional
// argument.
func completeRecurringIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	templates, err := s.LoadRecurring()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(templates))
	for _, r := range templates {
		out = append(out, idCompletion(r.ID, r.Text))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
