// Package selfinstall implements `anchor install`: copying the running
// binary into a permanent location and making sure that location is on
// PATH.
package selfinstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// BinaryName is the platform-appropriate executable name.
func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "anchor.exe"
	}
	return "anchor"
}

// Dir returns the directory anchor installs its binary into: ~/.anchor/bin.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".anchor", "bin"), nil
}

// Install copies the currently running binary into the install directory
// (creating it if needed, replacing whatever binary was already there) and
// ensures that directory is on PATH. Returns the installed path and
// whether PATH was newly updated (false if it was already present, or if
// this platform has no supported way to update it persistently).
func Install() (installedPath string, pathUpdated bool, err error) {
	src, err := os.Executable()
	if err != nil {
		return "", false, fmt.Errorf("locating current binary: %w", err)
	}
	src, err = filepath.EvalSymlinks(src)
	if err != nil {
		return "", false, fmt.Errorf("resolving current binary: %w", err)
	}

	dir, err := Dir()
	if err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, fmt.Errorf("creating %s: %w", dir, err)
	}
	dst := filepath.Join(dir, BinaryName())

	// If we're already running the installed copy, there's nothing to
	// replace it with.
	if filepath.Clean(src) != filepath.Clean(dst) {
		if err := copyFile(src, dst); err != nil {
			return "", false, err
		}
	}

	updated, err := addToPath(dir)
	if err != nil {
		return dst, false, fmt.Errorf("installed to %s but failed to update PATH: %w", dst, err)
	}
	return dst, updated, nil
}

// copyFile copies src to dst via a temp file + rename, so a failed copy
// never leaves a truncated binary at dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("replacing %s: %w", dst, err)
	}
	return nil
}
