package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 7, cfg.CacheTTLDays)
	assert.Equal(t, 1000, cfg.MaxHistoryEntries)
	assert.Equal(t, "command", cfg.DefaultAction)
	assert.Equal(t, 300, cfg.SearchDebounceMS)
	assert.NotEmpty(t, cfg.CacheDir)
	assert.NotEmpty(t, cfg.HistoryFile)
}

func TestConfigSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create config
	cfg := DefaultConfig()
	cfg.CacheTTLDays = 14
	cfg.MaxHistoryEntries = 500

	// Save config
	err := cfg.Save()
	require.NoError(t, err)

	// Verify file exists
	configPath := filepath.Join(tempDir, ".config", "gopick", "config.json")
	assert.FileExists(t, configPath)

	// Load config
	loaded, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, loaded)

	// Verify loaded values match (note: paths will be expanded)
	assert.Equal(t, cfg.CacheTTLDays, loaded.CacheTTLDays)
	assert.Equal(t, cfg.MaxHistoryEntries, loaded.MaxHistoryEntries)
	assert.Equal(t, cfg.DefaultAction, loaded.DefaultAction)
	assert.Equal(t, cfg.SearchDebounceMS, loaded.SearchDebounceMS)
}

func TestConfigExpandPaths(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde expansion",
			input:    "~/.config/gopick",
			expected: filepath.Join(homeDir, ".config", "gopick"),
		},
		{
			name:     "no expansion needed",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input, homeDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigEnsureDirectories(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &Config{
		CacheDir:    filepath.Join(tempDir, "cache"),
		HistoryFile: filepath.Join(tempDir, "history", ".gopick_history"),
	}

	err := cfg.ensureDirectories()
	require.NoError(t, err)

	// Check directories exist
	assert.DirExists(t, cfg.CacheDir)
	assert.DirExists(t, filepath.Dir(cfg.HistoryFile))
}

func TestConfigGetDebounceTime(t *testing.T) {
	cfg := &Config{
		SearchDebounceMS: 500,
	}

	duration := cfg.GetDebounceTime()
	assert.Equal(t, "500ms", duration.String())
}

func TestConfigJSON(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CacheTTLDays = 10
	cfg.MaxHistoryEntries = 2000

	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	// Unmarshal back
	var loaded Config
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, cfg.CacheTTLDays, loaded.CacheTTLDays)
	assert.Equal(t, cfg.MaxHistoryEntries, loaded.MaxHistoryEntries)
	assert.Equal(t, cfg.DefaultAction, loaded.DefaultAction)
}

func TestConfigLoadWithInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create invalid config file
	configDir := filepath.Join(tempDir, ".config", "gopick")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.json")
	err := os.WriteFile(configPath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// Try to load
	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
