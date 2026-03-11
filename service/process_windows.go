//go:build windows

package service

import (
	"strings"
	"syscall"
	"unsafe"
)

const TH32CS_SNAPPROCESS = 0x00000002

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procIsHungAppWindow          = user32.NewProc("IsHungAppWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

type enumCtx struct {
	pidMap    map[uint32]string
	statusMap map[string]string
}

// GetRunningProcesses returns a map of process name (lowercase) to its status.
// It detects "Not Responding" status by enumerating windows and checking IsHungAppWindow.
func GetRunningProcesses() (map[string]string, error) {
	// 1. Snapshot all running processes
	snapshot, err := syscall.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(snapshot)

	var pe32 syscall.ProcessEntry32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	pidMap := make(map[uint32]string)
	statusMap := make(map[string]string)

	if err := syscall.Process32First(snapshot, &pe32); err == nil {
		for {
			name := strings.ToLower(syscall.UTF16ToString(pe32.ExeFile[:]))
			pidMap[pe32.ProcessID] = name
			statusMap[name] = "Running" // Default assumption
			if err := syscall.Process32Next(snapshot, &pe32); err != nil {
				break
			}
		}
	}

	// 2. Iterate Windows to find any that are hung
	ctx := &enumCtx{
		pidMap:    pidMap,
		statusMap: statusMap,
	}

	// Callback function for EnumWindows
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		myCtx := (*enumCtx)(unsafe.Pointer(lParam))

		var pid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

		if name, exists := myCtx.pidMap[pid]; exists {
			// Check if hung
			ret, _, _ := procIsHungAppWindow.Call(uintptr(hwnd))
			if ret != 0 {
				myCtx.statusMap[name] = "Not Responding" // Localized later in GUI
			}
		}
		return 1 // Continue enumeration
	})

	procEnumWindows.Call(cb, uintptr(unsafe.Pointer(ctx)))

	return statusMap, nil
}
