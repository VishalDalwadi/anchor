//go:build !windows

package selfinstall

// addToPath is a no-op on non-Windows platforms: there's no single safe,
// shell-agnostic way to persist a PATH change from inside a program (it
// depends on which shell and which profile file the user's session
// reads). The caller prints a manual instruction instead.
func addToPath(dir string) (bool, error) {
	return false, nil
}
