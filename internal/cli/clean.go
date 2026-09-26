package cli

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/store"
)

// cleanConfirmation is what must be typed to confirm `anchor clean`: a
// deliberate word rather than y/n, given what it does.
const cleanConfirmation = "DELETE"

func newCleanCmd() *cobra.Command {
	var yes, noBackup bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete ALL anchor data (backed up first unless --no-backup). Cannot be undone.",
		Long: `Delete ALL anchor data: tasks, goals, watchlist, recurring templates, lists,
focus, and every journal and log file. This cannot be undone.

Before deleting, the data folder is backed up to
~/.anchor/backups/anchor-backup-<timestamp>.zip, unless --no-backup; restoring
is unzipping it back into ~/.anchor/data. Settings (~/.anchor/config.json)
are kept.

You're asked to type ` + cleanConfirmation + ` to confirm; --yes skips that, for scripts.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			dataDir, err := anchorDir("data")
			if err != nil {
				return err
			}
			files, err := dataFiles(dataDir)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Fprintln(out, "no anchor data to delete")
				return nil
			}

			if !yes {
				fmt.Fprintf(out, "This deletes ALL anchor data in %s (%d file(s)): tasks, goals, watchlist,\n", dataDir, len(files))
				fmt.Fprintln(out, "recurring templates, lists, focus, journal and logs. It cannot be undone.")
				if noBackup {
					fmt.Fprintln(out, "No backup will be made (--no-backup).")
				} else {
					fmt.Fprintln(out, "A backup zip is written to ~/.anchor/backups first.")
				}
				fmt.Fprintf(out, "Type %s to confirm: ", cleanConfirmation)
				answer, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				if err != nil && err != io.EOF {
					return err
				}
				// Strip a BOM too: PowerShell adds one when the answer is piped.
				if strings.TrimSpace(strings.TrimPrefix(answer, utf8BOM)) != cleanConfirmation {
					fmt.Fprintln(out, "\nnot confirmed; nothing was deleted")
					return nil
				}
			}

			if !noBackup {
				backupsDir, err := anchorDir("backups")
				if err != nil {
					return err
				}
				zipPath, err := backupData(dataDir, backupsDir, time.Now())
				if err != nil {
					return fmt.Errorf("backing up before clean (nothing was deleted): %w", err)
				}
				fmt.Fprintf(out, "backed up to %s\n", zipPath)
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			if err := s.Clean(); err != nil {
				return err
			}
			fmt.Fprintln(out, "all anchor data deleted")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask for confirmation (for scripts)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "don't write a backup zip first")
	return cmd
}

// dataFiles lists the files `clean` would delete, as paths relative to
// dataDir.
func dataFiles(dataDir string) ([]string, error) {
	var files []string
	for _, p := range store.DataPaths {
		root := filepath.Join(dataDir, filepath.FromSlash(p))
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(dataDir, path)
			files = append(files, filepath.ToSlash(rel))
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// backupData zips everything `clean` would delete from dataDir into a new
// timestamped file in backupsDir, and returns its path. Entries keep their
// paths relative to dataDir, so unzipping into dataDir restores them.
func backupData(dataDir, backupsDir string, now time.Time) (string, error) {
	files, err := dataFiles(dataDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		return "", err
	}
	zipPath := filepath.Join(backupsDir, "anchor-backup-"+now.Format("20060102-150405")+".zip")
	f, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}

	zw := zip.NewWriter(f)
	for _, rel := range files {
		if err = addToZip(zw, filepath.Join(dataDir, filepath.FromSlash(rel)), rel); err != nil {
			break
		}
	}
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(zipPath) // don't leave a partial backup that looks whole
		return "", err
	}
	return zipPath, nil
}

func addToZip(zw *zip.Writer, path, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}
