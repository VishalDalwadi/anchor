//go:build windows

package selfinstall

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// addToPath adds dir to the current user's persistent PATH (HKCU\Environment)
// if it isn't already there, then broadcasts WM_SETTINGCHANGE so processes
// launched after this point (e.g. from Explorer) pick it up without a
// reboot. Terminals already open still need to be restarted — that's
// inherent to how PATH is inherited on process creation, not something
// this can work around.
func addToPath(dir string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()

	current, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return false, err
	}

	for _, p := range strings.Split(current, ";") {
		if strings.EqualFold(strings.TrimSpace(p), dir) {
			return false, nil
		}
	}

	newPath := dir
	if current != "" {
		newPath = strings.TrimSuffix(current, ";") + ";" + dir
	}
	if err := key.SetExpandStringValue("Path", newPath); err != nil {
		return false, err
	}

	broadcastEnvChange()
	return true, nil
}

func broadcastEnvChange() {
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001A
		smtoAbortIfHung = 0x0002
	)
	proc := syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW")
	param, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	var result uintptr
	_, _, _ = proc.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(param)),
		smtoAbortIfHung,
		1000,
		uintptr(unsafe.Pointer(&result)),
	)
}

const completionMarker = "# anchor shell completion"

// setupCompletion wires `anchor completion powershell` into the user's
// PowerShell profile so tab-completion works in every new session,
// without needing them to source it by hand. Idempotent: does nothing if
// the marker line is already present.
func setupCompletion() (string, error) {
	profilePath, err := powershellProfilePath()
	if err != nil {
		return "", err
	}
	return addCompletionLine(profilePath)
}

// addCompletionLine appends the completion-sourcing line to the profile
// at path, unless it's already there. Split out from setupCompletion so
// the write/idempotency logic can be tested against a throwaway file
// instead of a real PowerShell profile.
func addCompletionLine(profilePath string) (string, error) {
	existing, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("reading %s: %w", profilePath, err)
	}
	if strings.Contains(string(existing), completionMarker) {
		return "PowerShell completion already configured in " + profilePath, nil
	}

	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(profilePath), err)
	}
	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", profilePath, err)
	}
	defer f.Close()

	line := fmt.Sprintf("\n%s\nif (Get-Command anchor -ErrorAction SilentlyContinue) { anchor completion powershell | Out-String | Invoke-Expression }\n", completionMarker)
	if _, err := f.WriteString(line); err != nil {
		return "", fmt.Errorf("writing %s: %w", profilePath, err)
	}

	return "added PowerShell completion to " + profilePath + " — open a new terminal for it to take effect", nil
}

// setupRecurTask registers (or replaces) a Windows Scheduled Task that
// runs `<anchorPath> recur run` once daily, catching up if the machine
// was asleep/off at the scheduled time. Registering a task in the current
// user's own context (no -RunLevel Highest, no stored password) doesn't
// require elevation.
func setupRecurTask(anchorPath string) (string, error) {
	script := recurTaskScript(anchorPath, RecurTaskName)
	out, err := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("registering scheduled task %s: %w: %s", RecurTaskName, err, strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("scheduled task %q registered to run `recur run` daily", RecurTaskName), nil
}

// recurTaskScript builds the PowerShell script that (re)registers the
// scheduled task. Split out as a pure function so the quoting logic can
// be unit tested without touching the real Task Scheduler.
func recurTaskScript(anchorPath, taskName string) string {
	return fmt.Sprintf(`$Action = New-ScheduledTaskAction -Execute %s -Argument 'recur run'
$Trigger = New-ScheduledTaskTrigger -Daily -At 9am
$Settings = New-ScheduledTaskSettingsSet -StartWhenAvailable
Register-ScheduledTask -TaskName %s -Action $Action -Trigger $Trigger -Settings $Settings -Force | Out-Null`,
		psQuote(anchorPath), psQuote(taskName))
}

// psQuote wraps s in single quotes for embedding in a PowerShell script,
// doubling any embedded single quotes per PowerShell's escaping rule.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// powershellProfilePath asks the live PowerShell host for $PROFILE rather
// than guessing between the Windows PowerShell 5.1 and PowerShell 7+
// locations, so it matches whatever the user's actual shell reads.
func powershellProfilePath() (string, error) {
	out, err := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-Command", "$PROFILE").Output()
	if err != nil {
		return "", fmt.Errorf("locating PowerShell profile: %w", err)
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", fmt.Errorf("PowerShell returned an empty $PROFILE path")
	}
	return path, nil
}
