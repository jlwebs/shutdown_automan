//go:build !windows
// +build !windows

package service

// GetRunningProcesses is a stub for non-Windows platforms.
// It returns an empty map.
func GetRunningProcesses() (map[string]string, error) {
	return make(map[string]string), nil
}
