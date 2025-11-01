package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHistory(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)
	assert.NotNil(t, h)
	assert.FileExists(t, historyFile)
}

func TestHistoryAdd(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add entry
	err = h.Add("testpkg", "github.com/test/testpkg", ActionInstalled)
	require.NoError(t, err)

	// Read back
	entries, err := h.GetAll()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "testpkg", entries[0].Package)
	assert.Equal(t, "github.com/test/testpkg", entries[0].ImportPath)
	assert.Equal(t, ActionInstalled, entries[0].Action)
}

func TestHistoryDuplicateDetection(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add same entry multiple times
	h.Add("testpkg", "github.com/test/testpkg", ActionViewed)
	h.Add("testpkg", "github.com/test/testpkg", ActionViewed)
	h.Add("testpkg", "github.com/test/testpkg", ActionViewed)

	// Should only have one entry (duplicates ignored)
	entries, err := h.GetAll()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestHistoryCircularBuffer(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	// Small buffer for testing
	h, err := New(historyFile, 5)
	require.NoError(t, err)

	// Add more entries than max
	for i := 0; i < 10; i++ {
		pkg := fmt.Sprintf("pkg%d", i)
		path := fmt.Sprintf("github.com/test/pkg%d", i)
		h.Add(pkg, path, ActionInstalled)
		// Sleep to avoid duplicate detection, this is only way i found lol
		time.Sleep(10 * time.Millisecond)
	}

	// Should only keep last 5
	entries, err := h.GetAll()
	require.NoError(t, err)
	assert.Len(t, entries, 5)

	// Should be the last 5 entries (pkg5 through pkg9)
	assert.Equal(t, "pkg5", entries[0].Package)
	assert.Equal(t, "pkg9", entries[4].Package)
}

func TestHistoryGetRecent(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add entries
	for i := 0; i < 10; i++ {
		pkg := fmt.Sprintf("pkg%d", i)
		path := fmt.Sprintf("github.com/test/pkg%d", i)
		h.Add(pkg, path, ActionViewed)
		time.Sleep(10 * time.Millisecond)
	}

	// Get recent 3
	recent, err := h.GetRecent(3)
	require.NoError(t, err)
	assert.Len(t, recent, 3)

	// Should be the last 3 (pkg7, pkg8, pkg9)
	assert.Equal(t, "pkg7", recent[0].Package)
	assert.Equal(t, "pkg9", recent[2].Package)
}

func TestHistorySearch(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add various entries
	h.Add("cobra", "github.com/spf13/cobra", ActionInstalled)
	h.Add("viper", "github.com/spf13/viper", ActionInstalled)
	h.Add("gin", "github.com/gin-gonic/gin", ActionViewed)
	h.Add("echo", "github.com/labstack/echo", ActionViewed)

	// Search for "spf13"
	results, err := h.Search("spf13")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Search for "gin"
	results, err = h.Search("gin")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "gin", results[0].Package)
}

func TestHistoryClear(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add entries
	h.Add("pkg1", "path1", ActionInstalled)
	h.Add("pkg2", "path2", ActionViewed)

	// Clear
	err = h.Clear()
	require.NoError(t, err)

	// Should be empty
	entries, err := h.GetAll()
	require.NoError(t, err)
	assert.Len(t, entries, 0)

	// File should still exist but be empty
	assert.FileExists(t, historyFile)
	stat, _ := os.Stat(historyFile)
	assert.Equal(t, int64(0), stat.Size())
}

func TestHistoryJSONLFormat(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add entries
	h.Add("pkg1", "github.com/test/pkg1", ActionInstalled)
	h.Add("pkg2", "github.com/test/pkg2", ActionViewed)

	// Read file directly
	file, err := os.Open(historyFile)
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		lineCount++

		// Each line should be valid JSON
		var entry Entry
		err := json.Unmarshal(scanner.Bytes(), &entry)
		assert.NoError(t, err)
		assert.NotEmpty(t, entry.Package)
		assert.NotEmpty(t, entry.ImportPath)
		assert.NotZero(t, entry.Timestamp)
	}

	assert.Equal(t, 2, lineCount)
}

func TestHistoryConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, ".gopick_history")

	h, err := New(historyFile, 100)
	require.NoError(t, err)

	// Add entries concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			pkg := fmt.Sprintf("pkg%d", n)
			path := fmt.Sprintf("github.com/test/pkg%d", n)
			h.Add(pkg, path, ActionInstalled)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All entries should be present
	entries, err := h.GetAll()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 10)
}

func TestActionTypes(t *testing.T) {
	assert.Equal(t, ActionType("viewed"), ActionViewed)
	assert.Equal(t, ActionType("installed"), ActionInstalled)
}
