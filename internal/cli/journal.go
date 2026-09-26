package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/config"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

// The journal is deliberately unstructured: one append-only markdown file
// per day (journal/YYYY-MM-DD.md), each entry under a "## HH:MM" heading,
// so a day reads as a timeline. `journal complete` appends a "reviewed"
// marker line for the human; nothing else in anchor reads or depends on it.

func newJournalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Freeform daily journal: timestamped entries in one markdown file per day",
	}
	cmd.AddCommand(newJournalAddCmd())
	cmd.AddCommand(newJournalShowCmd())
	cmd.AddCommand(newJournalCompleteCmd())
	return cmd
}

func newJournalAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [text...]",
		Short: "Add a timestamped entry to today's journal",
		Long: `Add a timestamped entry to today's journal.

  anchor journal add some thoughts     words are joined; quotes optional
  <text> | anchor journal add          piped input is read as the entry
  anchor journal add -                 read the entry from stdin explicitly
  anchor journal add                   write it in the configured editor
                                       (anchor config set editor ...), or,
                                       with none set, type it right here and
                                       finish with Ctrl+Z then Enter

An empty entry adds nothing.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := journalEntryText(cmd, args)
			if err != nil {
				return err
			}
			text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
			if text == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "empty entry; nothing added")
				return nil
			}

			now := time.Now()
			return updateJournal(now.Format(validate.DateLayout), func(existing, date string) string {
				return appendJournalEntry(existing, date, now, text)
			})
		},
	}
}

func newJournalShowCmd() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a day's journal (default: today)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			day, err := journalDate(date)
			if err != nil {
				return err
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			content, err := s.LoadJournal(day)
			if err != nil {
				return err
			}
			if content == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "no journal entries for %s\n", day)
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), ensureTrailingNewline(content))
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "day to show, YYYY-MM-DD (default: today)")
	return cmd
}

func newJournalCompleteCmd() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "complete",
		Short: `Mark a day's journal as reviewed (appends "--- reviewed at HH:MM ---")`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			day, err := journalDate(date)
			if err != nil {
				return err
			}
			now := time.Now()
			return updateJournal(day, func(existing, date string) string {
				return appendReviewMarker(existing, date, now)
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "day to mark reviewed, YYYY-MM-DD (default: today)")
	return cmd
}

// journalDate resolves a --date flag: empty means today, else a strict
// YYYY-MM-DD (the journal looks back, so the forward-looking shorthands
// --due accepts don't apply).
func journalDate(flag string) (string, error) {
	if flag == "" {
		return validate.Today(), nil
	}
	if _, err := time.Parse(validate.DateLayout, flag); err != nil {
		return "", fmt.Errorf("invalid --date %q: must be YYYY-MM-DD", flag)
	}
	return flag, nil
}

// updateJournal loads a day's journal, applies change, and saves it.
func updateJournal(day string, change func(existing, date string) string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	existing, err := s.LoadJournal(day)
	if err != nil {
		return err
	}
	return s.SaveJournal(day, change(existing, day))
}

// appendJournalEntry adds text under a "## HH:MM" heading. A new day's
// file starts with a title line.
func appendJournalEntry(existing, date string, at time.Time, text string) string {
	return journalWithBlock(existing, date, fmt.Sprintf("## %s\n\n%s\n", at.Format("15:04"), text))
}

// appendReviewMarker adds the end-of-day "reviewed" marker line.
func appendReviewMarker(existing, date string, at time.Time) string {
	return journalWithBlock(existing, date, fmt.Sprintf("--- reviewed at %s ---\n", at.Format("15:04")))
}

func journalWithBlock(existing, date, block string) string {
	if strings.TrimSpace(existing) == "" {
		existing = "# Journal — " + date + "\n"
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// journalEntryText works out where a new entry comes from: the arguments;
// stdin when asked for with "-" or when something's piped in; else the
// configured editor; else typed into the terminal.
func journalEntryText(cmd *cobra.Command, args []string) (string, error) {
	switch {
	case len(args) == 1 && args[0] == "-":
		return readNotes(cmd, "-")
	case len(args) > 0:
		return strings.Join(args, " "), nil
	case !stdinIsTerminal(cmd):
		return readNotes(cmd, "-")
	}

	c, err := config.Load()
	if err != nil {
		return "", err
	}
	if len(c.Editor) > 0 {
		return editText(cmd, c.Editor)
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Type the entry; finish with Ctrl+Z then Enter (Ctrl+D on macOS/Linux), or Ctrl+C to cancel.")
	fmt.Fprintln(cmd.ErrOrStderr(), "(To write in an editor instead: anchor config set editor <command>)")
	return readNotes(cmd, "-")
}

// stdinIsTerminal reports whether the command's input is an interactive
// console, as opposed to a pipe or file.
func stdinIsTerminal(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// editText opens editor (program and arguments; the file is appended
// last) on an empty temp file and returns what was saved in it.
func editText(cmd *cobra.Command, editor []string) (string, error) {
	f, err := os.CreateTemp("", "anchor-journal-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	ed := exec.Command(editor[0], append(editor[1:], path)...)
	ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr()
	if err := ed.Run(); err != nil {
		return "", fmt.Errorf("running editor %s: %w", formatCommand(editor), err)
	}

	text, err := readEdited(path)
	if err != nil || strings.TrimSpace(text) != "" {
		return text, err
	}
	// Some editors return before the user is done — GoLand without
	// --wait, or an editor that opens the file in an already-running
	// window. Rather than read the still-empty file, wait for the user.
	fmt.Fprintf(cmd.ErrOrStderr(), "Nothing saved yet. If the editor is still open, save the entry there and press Enter;\n"+
		"otherwise press Enter to cancel. (An editor that waits avoids this, e.g. `goland -e --wait`.) ")
	if _, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n'); err != nil && err != io.EOF {
		return "", err
	}
	return readEdited(path)
}

func readEdited(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(data), utf8BOM), nil // some editors save a BOM
}
