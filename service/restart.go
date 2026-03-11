package service

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"shutdown_automan/config"
)

var (
	restartMutex sync.Mutex
	isRestarting bool
	cancelChan   chan struct{}
)

// TriggerRestart initiates the restart sequence.
// It ensures only one restart sequence runs at a time.
func TriggerRestart(cfg *config.Config) error {
	restartMutex.Lock()
	if isRestarting {
		restartMutex.Lock()
		return fmt.Errorf("restart sequence already in progress")
	}
	isRestarting = true
	c := make(chan struct{})
	cancelChan = c
	restartMutex.Unlock()

	defer func() {
		restartMutex.Lock()
		isRestarting = false
		cancelChan = nil
		restartMutex.Unlock()
	}()

	log.Println("Restart sequence initiated. Waiting 30 seconds before proceeding...")

	// Wait 30 seconds, allowing for cancellation
	select {
	case <-time.After(30 * time.Second):
		log.Println("30 seconds elapsed. Proceeding with restart sequence...")
		restartMutex.Lock()
		cancelChan = nil // Prevents further cancellation attempts during process kill loop
		restartMutex.Unlock()
	case <-c: // Cancelled
		log.Println("Restart sequence cancelled by user.")
		return nil
	}

	// 1. Terminate processes
	for _, proc := range cfg.Get().ProcessList {
		log.Printf(" terminating process: %s", proc.Name)
		if err := killProcess(proc.Name); err != nil {
			log.Printf("Failed to kill process %s: %v", proc.Name, err)
			// Continue even if kill fails, as per requirements to proceed to shutdown
		}

		// 2. Wait for delay
		if proc.Delay > 0 {
			log.Printf("Waiting %d seconds after killing %s...", proc.Delay, proc.Name)
			time.Sleep(time.Duration(proc.Delay) * time.Second)
		}
	}

	// 3. System Restart
	log.Println("All processes handled. Initiating system restart...")
	return systemRestart()
}

// CancelRestart cancels an ongoing restart sequence that is in the 30-second waiting period.
func CancelRestart() error {
	restartMutex.Lock()
	defer restartMutex.Unlock()

	if !isRestarting {
		return fmt.Errorf("no restart sequence in progress")
	}

	if cancelChan != nil {
		close(cancelChan)
		cancelChan = nil
		return nil
	}

	return fmt.Errorf("restart sequence cannot be cancelled at this stage")
}

func killProcess(name string) error {
	// forceful kill (/f) by image name (/im)
	cmd := exec.Command("taskkill", "/F", "/IM", name)
	return cmd.Run()
}

func systemRestart() error {
	// restart (/r) with delay 0 (/t 0)
	cmd := exec.Command("shutdown", "/r", "/t", "0")
	return cmd.Run()
}
