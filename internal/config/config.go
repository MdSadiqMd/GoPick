package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	CacheDir          string `json:"cache_dir"`
	HistoryFile       string `json:"history_file"`
	CacheTTLDays      int    `json:"cache_ttl_days"`
	MaxHistoryEntries int    `json:"max_history_entries"`
	DefaultAction     string `json:"default_action"`
	SearchDebounceMS  int    `json:"search_debounce_ms"`
	GoModCachePath    string `json:"gomodcache_path"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "gopick")

	// Get GOMODCACHE path
	goModCache := getGoModCachePath()

	return &Config{
		CacheDir:          filepath.Join(configDir, "cache"),
		HistoryFile:       filepath.Join(configDir, ".gopick_history"),
		CacheTTLDays:      7,
		MaxHistoryEntries: 1000,
		DefaultAction:     "command",
		SearchDebounceMS:  300,
		GoModCachePath:    goModCache,
	}
}

// loads the configuration from file or creates default
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "gopick", "config.json")

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if err := cfg.Save(); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.expandPaths()
	if err := cfg.ensureDirectories(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// saves the configuration to file
func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "gopick", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temp file first (atomic write)
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Rename to actual config file
	if err := os.Rename(tempPath, configPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// expands ~ and environment variables in paths
func (c *Config) expandPaths() {
	homeDir, _ := os.UserHomeDir()

	c.CacheDir = expandPath(c.CacheDir, homeDir)
	c.HistoryFile = expandPath(c.HistoryFile, homeDir)
	c.GoModCachePath = expandPath(c.GoModCachePath, homeDir)
}

// creates necessary directories
func (c *Config) ensureDirectories() error {
	dirs := []string{
		c.CacheDir,
		filepath.Dir(c.HistoryFile),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// expands ~ and environment variables in a path
func expandPath(path, homeDir string) string {
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir, path[2:])
	}
	return os.ExpandEnv(path)
}

// gets the GOMODCACHE path from go env
func getGoModCachePath() string {
	// Try to get GOMODCACHE
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(output))
		if path != "" {
			return path
		}
	}

	// Fallback to GOPATH/pkg/mod
	cmd = exec.Command("go", "env", "GOPATH")
	output, err = cmd.Output()
	if err == nil {
		gopath := strings.TrimSpace(string(output))
		if gopath != "" {
			return filepath.Join(gopath, "pkg", "mod")
		}
	}

	// Last resort default
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "go", "pkg", "mod")
}

func (c *Config) GetDebounceTime() time.Duration {
	return time.Duration(c.SearchDebounceMS) * time.Millisecond
}
