//go:build !windows
// +build !windows

package service

func GetSystemNetworkStats() (bytesRecv, bytesSent uint64, err error) {
	return 0, 0, nil
}
