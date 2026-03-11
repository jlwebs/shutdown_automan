package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"shutdown_automan/config"
	"shutdown_automan/gui"
)

func main() {
	// Setup logging to file with absolute path to ensure we find it
	exePath, _ := os.Executable()
	logPath := "log.txt"
	if exePath != "" {
		logPath = filepath.Join(filepath.Dir(exePath), "log.txt")
	}

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		defer f.Close()
		log.SetOutput(f)
	} else {
		// If we can't open log file, try to print to stdout (though invisible in GUI app)
		log.Printf("Failed to open log file: %v", err)
	}

	// Panic Recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC RECOVERED: %v\nStack: %s", r, debug.Stack())
		}
	}()

	log.Println("--- App Starting (Version: ARM64-Fix-2) ---")
	pwd, _ := os.Getwd()
	log.Printf("Working Directory: %s", pwd)

	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
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
