//go:build !windows

package selfinstall

// addToPath is a no-op on non-Windows platforms: there's no single safe,
// shell-agnostic way to persist a PATH change from inside a program (it
// depends on which shell and which profile file the user's session
// reads). The caller prints a manual instruction instead.
func addToPath(dir string) (bool, error) {
	return false, nil
}

// setupCompletion is a no-op on non-Windows platforms for the same reason:
// which rc file to edit (~/.bashrc, ~/.zshrc, fish config, ...) depends on
// a shell this program can't reliably infer. The caller prints manual
// instructions pointing at `anchor completion <shell>` instead.
func setupCompletion() (string, error) {
	return "", nil
}

// setupRecurTask is a no-op outside Windows: anchor-spec.md scopes
// `recur run`'s scheduler integration to Windows Task Scheduler
// specifically. The caller prints a manual cron suggestion instead.
func setupRecurTask(anchorPath string) (string, error) {
	return "", nil
}
