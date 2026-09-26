package cli

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// runAnchorIn is runAnchor with stdin.
func runAnchorIn(t *testing.T, home, stdin string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	root := newRootCmd(args)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func seedAnchorData(t *testing.T, home string) {
	t.Helper()
	for _, args := range [][]string{
		{"task", "add", "x", "--aspect", "career"},
		{"goal", "add", "g", "--aspect", "career", "--tier", "life"},
		{"list", "create", "movies"},
		{"focus", "set", "--aspect", "career"},
		{"journal", "add", "an", "entry"},
		{"brief"},
		{"config", "set", "editor", "vim"},
	} {
		if _, err := runAnchor(t, home, args...); err != nil {
			t.Fatalf("seeding %v: %v", args, err)
		}
	}
	// Something in the data folder that anchor doesn't own.
	if err := os.WriteFile(filepath.Join(home, ".anchor", "data", "mine.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func backups(t *testing.T, home string) []string {
	t.Helper()
	zips, _ := filepath.Glob(filepath.Join(home, ".anchor", "backups", "*.zip"))
	return zips
}

func TestCleanNeedsTheConfirmationWord(t *testing.T) {
	home := t.TempDir()
	seedAnchorData(t, home)
	for _, answer := range []string{"y\n", "yes\n", "delete\n", ""} {
		out, err := runAnchorIn(t, home, answer, "clean")
		if err != nil || !strings.Contains(out, "nothing was deleted") {
			t.Errorf("answer %q: out=%q err=%v; want nothing deleted", answer, out, err)
		}
	}
	if !exists(filepath.Join(home, ".anchor", "data", "tasks.json")) || len(backups(t, home)) != 0 {
		t.Error("an unconfirmed clean deleted data or wrote a backup")
	}
}

func TestCleanBacksUpThenDeletesEverything(t *testing.T) {
	home := t.TempDir()
	seedAnchorData(t, home)
	data := filepath.Join(home, ".anchor", "data")

	// With a BOM and CRLF, as PowerShell pipes it.
	out, err := runAnchorIn(t, home, utf8BOM+"DELETE\r\n", "clean")
	if err != nil {
		t.Fatalf("clean: %v\n%s", err, out)
	}

	// The backup holds every data file, at paths relative to data/.
	zips := backups(t, home)
	if len(zips) != 1 {
		t.Fatalf("backups = %v, want exactly one", zips)
	}
	zr, err := zip.OpenReader(zips[0])
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	zr.Close()
	sort.Strings(names)
	for _, want := range []string{"tasks.json", "goals.json", "collections.json", "focus.json", "journal/", "log/"} {
		found := false
		for _, n := range names {
			if n == want || (strings.HasSuffix(want, "/") && strings.HasPrefix(n, want)) {
				found = true
			}
		}
		if !found {
			t.Errorf("backup %v has no %s", names, want)
		}
	}

	// All anchor data is gone...
	for _, p := range []string{"tasks.json", "goals.json", "collections.json", "focus.json", "journal", "log"} {
		if exists(filepath.Join(data, p)) {
			t.Errorf("%s still exists after clean", p)
		}
	}
	// ...but settings and files anchor doesn't own are kept.
	if !exists(filepath.Join(home, ".anchor", "config.json")) {
		t.Error("clean deleted config.json")
	}
	if !exists(filepath.Join(data, "mine.txt")) {
		t.Error("clean deleted a file that isn't anchor's")
	}

	// Nothing left: a second clean says so, without asking.
	if out, _ := runAnchorIn(t, home, "", "clean"); !strings.Contains(out, "no anchor data to delete") {
		t.Errorf("second clean = %q", out)
	}
}

func TestCleanYesNoBackup(t *testing.T) {
	home := t.TempDir()
	seedAnchorData(t, home)
	out, err := runAnchorIn(t, home, "", "clean", "-y", "--no-backup")
	if err != nil || strings.Contains(out, "Type DELETE") {
		t.Fatalf("clean -y --no-backup: out=%q err=%v; want no prompt", out, err)
	}
	if exists(filepath.Join(home, ".anchor", "data", "tasks.json")) {
		t.Error("tasks.json survived clean -y")
	}
	if n := len(backups(t, home)); n != 0 {
		t.Errorf("%d backup(s) written despite --no-backup", n)
	}
}
