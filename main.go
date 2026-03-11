package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"shutdown_automan/config"
	"shutdown_automan/gui"
)

func main() {
	// 1. Setup logging to file with absolute path
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	logPath := filepath.Join(filepath.Dir(exePath), "log.txt")

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		defer f.Close()
		log.SetOutput(f)
	}
	log.Println("--- App Starting (Version: 2008R2-Compatible) ---")
	pwd, _ := os.Getwd()
	log.Printf("Working Directory: %s", pwd)
	log.Printf("Executable Path: %s", exePath)

	// Panic Recovery
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("CRITICAL ERROR: %v\n\nStack Trace:\n%s", r, debug.Stack())
			log.Println(panicMsg)
			_ = os.WriteFile(filepath.Join(filepath.Dir(exePath), "CRASH_REPORT.txt"), []byte(panicMsg), 0666)
		}
	}()

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("CRITICAL: Failed to load config: %v", err)
	} else {
		log.Println("Config loaded successfully")
	}

	// 2. Initialize GUI
	appGUI := gui.NewGUI(cfg)

	// 3. Run Application
	log.Println("Starting GUI Run Loop...")
	appGUI.Run()
	log.Println("--- App Exited (Normally) ---")
}
