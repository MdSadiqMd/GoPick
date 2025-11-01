package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	tempDir := t.TempDir()

	c, err := New(tempDir, 7)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, tempDir, c.dir)
	assert.Equal(t, 7, c.ttlDays)
	assert.DirExists(t, tempDir)
}

func TestCacheSetAndGet(t *testing.T) {
	tempDir := t.TempDir()
	c, err := New(tempDir, 7)
	require.NoError(t, err)

	packages := []Package{
		{
			Name:        "testpkg",
			ImportPath:  "github.com/test/testpkg",
			Description: "Test package",
			Version:     "v1.0.0",
		},
		{
			Name:        "anotherpkg",
			ImportPath:  "github.com/test/anotherpkg",
			Description: "Another test package",
		},
	}

	// Set cache
	err = c.Set("test query", packages)
	require.NoError(t, err)

	// Get cache
	entry, found := c.Get("test query")
	assert.True(t, found)
	assert.NotNil(t, entry)
	assert.Equal(t, "test query", entry.Query)
	assert.Len(t, entry.Results, 2)
	assert.Equal(t, packages[0].Name, entry.Results[0].Name)
	assert.Equal(t, packages[1].ImportPath, entry.Results[1].ImportPath)
}

func TestCacheGetNotFound(t *testing.T) {
	tempDir := t.TempDir()
	c, err := New(tempDir, 7)
	require.NoError(t, err)

	entry, found := c.Get("nonexistent query")
	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestCacheExpiration(t *testing.T) {
	tempDir := t.TempDir()
	c, err := New(tempDir, 0) // 0 days TTL
	require.NoError(t, err)

	// Create an entry with old timestamp
	entry := CacheEntry{
		Query:     "test",
		Results:   []Package{{Name: "pkg"}},
		Timestamp: time.Now().Add(-24 * time.Hour), // 1 day ago
	}

	// Write directly to file
	filename := c.getFilename("test")
	path := filepath.Join(c.dir, filename)
	data, _ := json.Marshal(entry)
	err = os.WriteFile(path, data, 0644)
	require.NoError(t, err)

	// Try to get
	cached, found := c.Get("test")
	assert.False(t, found)
	assert.Nil(t, cached)

	// File should be deleted
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestCacheClear(t *testing.T) {
	tempDir := t.TempDir()
	c, err := New(tempDir, 7)
	require.NoError(t, err)

	// Set multiple cache entries
	c.Set("query1", []Package{{Name: "pkg1"}})
	c.Set("query2", []Package{{Name: "pkg2"}})
	c.Set("query3", []Package{{Name: "pkg3"}})

	// Verify files exist
	files, _ := os.ReadDir(c.dir)
	assert.GreaterOrEqual(t, len(files), 3)

	// Clear cache
	err = c.Clear()
	require.NoError(t, err)

	// Verify files are deleted
	files, _ = os.ReadDir(c.dir)
	assert.Equal(t, 0, len(files))
}

func TestCacheCleanExpired(t *testing.T) {
	tempDir := t.TempDir()
	c, err := New(tempDir, 1) // 1 day TTL
	require.NoError(t, err)

	// Create valid entry
	validEntry := CacheEntry{
		Query:     "valid",
		Results:   []Package{{Name: "valid"}},
		Timestamp: time.Now(),
	}

	// Create expired entry
	expiredEntry := CacheEntry{
		Query:     "expired",
		Results:   []Package{{Name: "expired"}},
		Timestamp: time.Now().Add(-48 * time.Hour), // 2 days ago
	}

	// Write both entries
	validPath := filepath.Join(c.dir, c.getFilename("valid"))
	expiredPath := filepath.Join(c.dir, c.getFilename("expired"))

	validData, _ := json.Marshal(validEntry)
	expiredData, _ := json.Marshal(expiredEntry)

	os.WriteFile(validPath, validData, 0644)
	os.WriteFile(expiredPath, expiredData, 0644)

	// Clean expired
	err = c.CleanExpired()
	require.NoError(t, err)

	// Valid should exist
	assert.FileExists(t, validPath)

	// Expired should be deleted
	_, err = os.Stat(expiredPath)
	assert.True(t, os.IsNotExist(err))
}

func TestCacheFilename(t *testing.T) {
	c := &Cache{}

	// Test deterministic filename generation
	filename1 := c.getFilename("test query")
	filename2 := c.getFilename("test query")
	assert.Equal(t, filename1, filename2)

	// Different queries should have different filenames
	filename3 := c.getFilename("different query")
	assert.NotEqual(t, filename1, filename3)

	// Should end with .json
	assert.Contains(t, filename1, ".json")
}

func TestCacheAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	c, err := New(tempDir, 7)
	require.NoError(t, err)

	packages := []Package{{Name: "test"}}

	// Set cache multiple times concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			c.Set("concurrent", packages)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still be able to read
	entry, found := c.Get("concurrent")
	assert.True(t, found)
	assert.NotNil(t, entry)
}

func TestPackageStruct(t *testing.T) {
	pkg := Package{
		Name:        "example",
		ImportPath:  "github.com/example/pkg",
		Description: "Example package",
		Version:     "v1.2.3",
		IsInstalled: true,
	}

	// Test JSON marshaling
	data, err := json.Marshal(pkg)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled Package
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, pkg, unmarshaled)
}
