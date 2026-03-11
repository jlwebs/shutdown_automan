package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"shutdown_automan/config"
)

// StartHTTPServer starts the HTTP server. It listens on the configured port.
// It uses context for shutdown.
func StartHTTPServer(ctx context.Context, cfg *config.Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		// Allow GET (for easy link access) and POST
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Auth check
		currentCfg := cfg.Get()

		// Debug Log
		requestKey := r.URL.Query().Get("key")
		log.Printf("[HTTP] Received restart request. ConfiguredKey='%s', RequestKey='%s'", currentCfg.SecretKey, requestKey)

		if currentCfg.SecretKey != "" {
			if requestKey != currentCfg.SecretKey {
				log.Printf("[HTTP] Auth failed: keys do not match.")
				http.Error(w, "Invalid Secret Key", http.StatusForbidden)
				return
			}
		}

		go func() {
			if err := TriggerRestart(cfg); err != nil {
				log.Printf("Restart failed: %v", err)
			}
		}()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Restart initiated")
	})

	mux.HandleFunc("/process_status", func(w http.ResponseWriter, r *http.Request) {
		currentCfg := cfg.Get()

		// Auth check
		requestKey := r.URL.Query().Get("key")
		if currentCfg.SecretKey != "" {
			if requestKey != currentCfg.SecretKey {
				http.Error(w, "Invalid Secret Key", http.StatusForbidden)
				return
			}
		}

		runningMap, err := GetRunningProcesses()
		if err != nil {
			http.Error(w, "Failed to get process status: "+err.Error(), http.StatusInternalServerError)
			return
		}

		type ProcessStatus struct {
			Name            string `json:"name"`
			Status          string `json:"status"`
			Delay           int    `json:"delay"`
			IsNotResponding bool   `json:"is_not_responding"`
		}

		type SystemStatus struct {
			Processes          []ProcessStatus `json:"processes"`
			NetworkSpeedInBps  float64         `json:"network_speed_in_bps"`
			NetworkSpeedOutBps float64         `json:"network_speed_out_bps"`
		}

		var statuses []ProcessStatus
		for _, p := range currentCfg.ProcessList {
			procInfo, exists := runningMap[strings.ToLower(p.Name)]
			stat := procInfo.Status
			isHung := procInfo.IsHung
			// Normalize/translate "Unknown" status
			if !exists {
				stat = "Not Started"
				isHung = false
			} else if strings.Contains(stat, "Unknown") {
				stat = "Running"
			}
			statuses = append(statuses, ProcessStatus{
				Name:            p.Name,
				Status:          stat,
				Delay:           p.Delay,
				IsNotResponding: isHung,
			})
		}

		inSpeed, outSpeed := GetNetworkSpeed10Min()
		response := SystemStatus{
			Processes:          statuses,
			NetworkSpeedInBps:  inSpeed,
			NetworkSpeedOutBps: outSpeed,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		// Watch for config changes or context cancellation to manage server restarts if needed
		// For now, simple server starting on configured port.
		// Handling dynamic port change is complex (requires server restart).
		// We'll stick to initial port for now, or implement a restart logic in main loop if config changes.

		port := cfg.Get().Port
		if port == "" {
			port = "8080"
		}
		server.Addr = ":" + port

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP Server error: %v", err)
		}
	}()

	<-ctx.Done()
	return server.Shutdown(context.Background())
}
