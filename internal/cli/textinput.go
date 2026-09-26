package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/config"
)

// Multiline text (journal entries, notes given as "-") comes from, in
// order: piped stdin; else the editor set with `anchor config set editor`;
// else typed straight into the terminal.

// utf8BOM is the byte-order mark some tools (PowerShell when piping,
// some editors when saving) put at the start of text.
const utf8BOM = string(rune(0xFEFF))

// readTextInput gets multiline text from wherever the user can give it
// (see above). current is what the editor starts with, e.g. the notes
// being replaced; it's ignored for stdin and typing, which start empty.
// The result has Windows line endings normalized and trailing newlines
// trimmed.
func readTextInput(cmd *cobra.Command, current string) (string, error) {
	if !stdinIsTerminal(cmd) {
		return readStdin(cmd)
	}
	c, err := config.Load()
	if err != nil {
		return "", err
	}
	if len(c.Editor) > 0 {
		text, err := editText(cmd, c.Editor, current)
		return normalizeText(text), err
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Type the text; finish with Ctrl+Z then Enter (Ctrl+D on macOS/Linux), or Ctrl+C to cancel.")
	fmt.Fprintln(cmd.ErrOrStderr(), "(To write in an editor instead: anchor config set editor <command>)")
	return readStdin(cmd)
}

// readStdin reads all of stdin, normalized like readTextInput's result.
func readStdin(cmd *cobra.Command) (string, error) {
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	return normalizeText(string(data)), nil
}

func normalizeText(s string) string {
	s = strings.TrimPrefix(s, utf8BOM)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimRight(s, "\n")
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
// last) on a temp file holding initial, and returns what's saved in it.
func editText(cmd *cobra.Command, editor []string, initial string) (string, error) {
	f, err := os.CreateTemp("", "anchor-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)
	if initial != "" {
		_, err = f.WriteString(initial + "\n")
	}
	f.Close()
	if err != nil {
		return "", err
	}
	before, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	ed := exec.Command(editor[0], append(editor[1:], path)...)
	ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr()
	if err := ed.Run(); err != nil {
		return "", fmt.Errorf("running editor %s: %w", formatCommand(editor), err)
	}

	if after, err := os.Stat(path); err == nil && after.ModTime().Equal(before.ModTime()) {
		// Never saved. Some editors return before the user is done —
		// GoLand without --wait, or one that opens the file in an
		// already-running window — so rather than take the untouched
		// file, wait for the user.
		fmt.Fprint(cmd.ErrOrStderr(), "Nothing saved yet. If the editor is still open, save there and press Enter;\n"+
			"otherwise press Enter to leave it unchanged. (An editor that waits avoids this, e.g. `goland -e --wait`.) ")
		if _, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n'); err != nil && err != io.EOF {
			return "", err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(data), utf8BOM), nil
}
