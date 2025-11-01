package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CacheEntry struct {
	Query     string    `json:"query"`
	Results   []Package `json:"results"`
	Timestamp time.Time `json:"timestamp"`
}

type Package struct {
	Name        string `json:"name"`
	ImportPath  string `json:"import_path"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	IsInstalled bool   `json:"is_installed,omitempty"`
}

type Cache struct {
	dir     string
	ttlDays int
}

func New(cacheDir string, ttlDays int) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{
		dir:     cacheDir,
		ttlDays: ttlDays,
	}, nil
}

func (c *Cache) Get(query string) (*CacheEntry, bool) {
	filename := c.getFilename(query)
	path := filepath.Join(c.dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	if c.isExpired(entry.Timestamp) {
		os.Remove(path)
		return nil, false
	}

	return &entry, true
}

func (c *Cache) Set(query string, packages []Package) error {
	entry := CacheEntry{
		Query:     query,
		Results:   packages,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	filename := c.getFilename(query)
	path := filepath.Join(c.dir, filename)

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(c.dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove cache file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

func (c *Cache) CleanExpired() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(c.dir, entry.Name())

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var cacheEntry CacheEntry
			if err := json.Unmarshal(data, &cacheEntry); err != nil {
				continue
			}

			if c.isExpired(cacheEntry.Timestamp) {
				os.Remove(path)
			}
		}
	}

	return nil
}

func (c *Cache) getFilename(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:]) + ".json"
}

func (c *Cache) isExpired(timestamp time.Time) bool {
	ttl := time.Duration(c.ttlDays) * 24 * time.Hour
	return time.Since(timestamp) > ttl
}

func (c *Cache) GetTTL() int {
	return c.ttlDays
}
