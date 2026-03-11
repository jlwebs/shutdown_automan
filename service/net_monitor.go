package service

import (
	"sync"
	"time"
)

var (
	netHistoryMux sync.Mutex
	netHistory    [10]struct {
		Timestamp time.Time
		BytesIn   uint64
		BytesOut  uint64
	}
	netHistoryIdx int
	netHistoryLen int
	netMonitorStarted bool
)

func init() {
	go trackNetworkStats()
}

func trackNetworkStats() {
	netHistoryMux.Lock()
	if netMonitorStarted {
		netHistoryMux.Unlock()
		return
	}
	netMonitorStarted = true
	// get initial
	in, out, _ := GetSystemNetworkStats()
	netHistory[0] = struct {
		Timestamp time.Time
		BytesIn   uint64
		BytesOut  uint64
	}{time.Now(), in, out}
	netHistoryIdx = 0
	netHistoryLen = 1
	netHistoryMux.Unlock()

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		in, out, _ := GetSystemNetworkStats()
		netHistoryMux.Lock()
		netHistoryIdx = (netHistoryIdx + 1) % 10
		netHistory[netHistoryIdx] = struct {
			Timestamp time.Time
			BytesIn   uint64
			BytesOut  uint64
		}{time.Now(), in, out}
		if netHistoryLen < 10 {
			netHistoryLen++
		}
		netHistoryMux.Unlock()
	}
}

// GetNetworkSpeed10Min returns the average network speed over the last 10 minutes in Bytes per second.
func GetNetworkSpeed10Min() (inSpeed, outSpeed float64) {
	netHistoryMux.Lock()
	defer netHistoryMux.Unlock()

	if netHistoryLen < 2 {
		return 0, 0
	}

	oldestIdx := (netHistoryIdx - netHistoryLen + 1 + 10) % 10
	latest := netHistory[netHistoryIdx]
	oldest := netHistory[oldestIdx]

	duration := latest.Timestamp.Sub(oldest.Timestamp).Seconds()
	if duration <= 0 {
		return 0, 0
	}

	// Handle counter wrap-around (if any, uint64 shouldn't wrap soon, but just in case)
	var inDiff, outDiff uint64
	if latest.BytesIn >= oldest.BytesIn {
		inDiff = latest.BytesIn - oldest.BytesIn
	}
	if latest.BytesOut >= oldest.BytesOut {
		outDiff = latest.BytesOut - oldest.BytesOut
	}

	inSpeed = float64(inDiff) / duration
	outSpeed = float64(outDiff) / duration
	return
}
