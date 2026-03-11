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
)

// TriggerRestart initiates the restart sequence.
// It ensures only one restart sequence runs at a time.
func TriggerRestart(cfg *config.Config) error {
	restartMutex.Lock()
	if isRestarting {
		restartMutex.Unlock()
		return fmt.Errorf("restart sequence already in progress")
	}
	isRestarting = true
	restartMutex.Unlock()

	defer func() {
		restartMutex.Lock()
		isRestarting = false
		restartMutex.Unlock()
	}()

	log.Println("Starting restart sequence...")

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
