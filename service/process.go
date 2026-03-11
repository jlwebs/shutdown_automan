//go:build !windows
// +build !windows

package service

type ProcessStateInfo struct {
	Status string // "Running", "Not Responding", etc.
	IsHung bool
}

// GetRunningProcesses is a stub for non-Windows platforms.
// It returns an empty map.
func GetRunningProcesses() (map[string]ProcessStateInfo, error) {
	return make(map[string]ProcessStateInfo), nil
}
