package service

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"shutdown_automan/config"
)

// StartMonitor monitors the processes defined in the configuration.
// It uses a context for cancellation.
func StartMonitor(ctx context.Context, cfg *config.Config) {
	// Initial create ticker. Note: if interval changes, we might need a way to update ticker.
	// For simplicity, checking config on each tick is fine, but interval update requires restart of monitor or smarter loop.
	// We'll use a dynamic sleep approach instead of ticker for interval changes.
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		currentCfg := cfg.Get()
		interval := currentCfg.MonitorInterval
		if interval < 1 {
			interval = 60 // Default fallback
		}

		if currentCfg.MonitorEnabled {
			checkProcesses(cfg)
		}

		// Wait for the interval or context cancellation
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(interval) * time.Second):
			// Continue loop
		}
	}
}

func checkProcesses(cfg *config.Config) {
	// Re-fetch config to ensure latest process list
	list := cfg.Get().ProcessList
	for _, proc := range list {
		exists, err := isProcessRunning(proc.Name)
		if err != nil {
			log.Printf("Monitor: Error checking process %s: %v", proc.Name, err)
			continue
		}
		if !exists {
			log.Printf("Monitor: Process %s missing! Triggering restart...", proc.Name)
			// Trigger restart in a separate goroutine to not block monitor loop (though monitor loop is about to be useless after restart)
			go TriggerRestart(cfg)
			return // Stop monitoring as restart sequence has begun
		}
	}
}

func isProcessRunning(name string) (bool, error) {
	// tasklist /NH /FI "IMAGENAME eq name"
	// /NH = No Header
	cmd := exec.Command("tasklist", "/NH", "/FI", fmt.Sprintf("IMAGENAME eq %s", name))
	// On Windows, create fails if not set properly? Usually fine if system calls.
	// For cross-compilation safety (since I am writing on Mac), I assume this code runs on Windows.

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	outStr := string(output)
	
	// If output contains "No tasks are running" (in English Windows)
	// Or check if the process name appears in the output.
	// A more robust way might be checking for exact name match, but simple contains is usually okay.
	if strings.Contains(outStr, "No tasks are running") {
		return false, nil
	}
	// Case-insensitive check
	if strings.Contains(strings.ToLower(outStr), strings.ToLower(name)) {
		return true, nil
	}
	
	return false, nil
}
