package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ProcessItem struct {
	Name  string `json:"name"`
	Delay int    `json:"delay"` // Seconds
}

type Config struct {
	Port            string        `json:"port"`
	ProcessList     []ProcessItem `json:"process_list"`
	SecretKey       string        `json:"secret_key"` // Optional security key
	MonitorEnabled  bool          `json:"monitor_enabled"`
	MonitorInterval int           `json:"monitor_interval"` // Seconds
	Language        string        `json:"language"`         // "en" or "zh"

	// Internal synchronization
	mu sync.RWMutex
}

const ConfigFileName = "config.json"

func DefaultConfig() *Config {
	return &Config{
		Port:            "8080",
		ProcessList:     []ProcessItem{},
		MonitorEnabled:  false,
		MonitorInterval: 60,
		Language:        "zh",
	}
}

func LoadConfig() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(filepath.Dir(exePath), ConfigFileName)

	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := DefaultConfig()

	// Create a temporary struct to handle potential migration or extra fields
	type Alias Config
	aux := &struct {
		OldProcessList string `json:"processes_old,omitempty"` // Example field for migration
		*Alias
	}{
		Alias: (*Alias)(cfg),
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(aux); err != nil {
		return nil, err
	}

	// Migration logic: if OldProcessList is present and ProcessList is empty
	if aux.OldProcessList != "" && len(cfg.ProcessList) == 0 {
		parts := strings.Split(aux.OldProcessList, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.ProcessList = append(cfg.ProcessList, ProcessItem{Name: p, Delay: 0})
			}
		}
	}

	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	configPath := filepath.Join(filepath.Dir(exePath), ConfigFileName)

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}

func (c *Config) Update(newConfig Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Port = newConfig.Port
	c.ProcessList = newConfig.ProcessList
	c.MonitorEnabled = newConfig.MonitorEnabled
	c.MonitorInterval = newConfig.MonitorInterval
	c.SecretKey = newConfig.SecretKey
	c.Language = newConfig.Language
}

func (c *Config) Get() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c
}
