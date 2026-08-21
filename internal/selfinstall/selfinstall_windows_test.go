//go:build windows

package selfinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddCompletionLine(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "nested", "Microsoft.PowerShell_profile.ps1")

	msg, err := addCompletionLine(profile)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !strings.Contains(msg, "added PowerShell completion") {
		t.Errorf("first call message = %q, want it to say completion was added", msg)
	}

	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("reading profile: %v", err)
	}
	if !strings.Contains(string(data), completionMarker) {
		t.Errorf("profile does not contain marker:\n%s", data)
	}
	if !strings.Contains(string(data), "anchor completion powershell") {
		t.Errorf("profile does not source anchor completion:\n%s", data)
	}

	msg2, err := addCompletionLine(profile)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !strings.Contains(msg2, "already configured") {
		t.Errorf("second call message = %q, want it to say already configured", msg2)
	}

	data2, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("reading profile after second call: %v", err)
	}
	if strings.Count(string(data2), completionMarker) != 1 {
		t.Errorf("marker appears %d times after two calls, want 1 (not idempotent)", strings.Count(string(data2), completionMarker))
	}
}

func TestAddCompletionLinePreservesExistingContent(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "Microsoft.PowerShell_profile.ps1")
	const existing = "Set-Alias ll Get-ChildItem\n"
	if err := os.WriteFile(profile, []byte(existing), 0o644); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}

	if _, err := addCompletionLine(profile); err != nil {
		t.Fatalf("addCompletionLine: %v", err)
	}

	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("reading profile: %v", err)
	}
	if !strings.HasPrefix(string(data), existing) {
		t.Errorf("existing profile content was not preserved:\n%s", data)
	}
}
