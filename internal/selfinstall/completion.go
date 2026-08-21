package selfinstall

// SetupCompletion configures shell completion for the current platform's
// primary shell, where anchor knows how (PowerShell on Windows). It
// returns a human-readable message describing what happened, safe to
// print as-is; the caller doesn't need to know which shell it was.
//
// anchor's own completion scripts (bash/zsh/fish/powershell) come free
// from cobra via `anchor completion <shell>` — this just decides whether
// and how to wire that into a shell that starts automatically.
func SetupCompletion() (string, error) {
	return setupCompletion()
}
