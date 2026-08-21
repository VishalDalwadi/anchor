//go:build windows

package selfinstall

import (
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
