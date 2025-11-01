package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type ActionType string

const (
	ActionViewed    ActionType = "viewed"
	ActionInstalled ActionType = "installed"
)

type Entry struct {
	Timestamp  time.Time  `json:"timestamp"`
	Package    string     `json:"package"`
	ImportPath string     `json:"import_path"`
	Action     ActionType `json:"action"`
}

type History struct {
	file       string
	maxEntries int
	mu         sync.Mutex
}

func New(historyFile string, maxEntries int) (*History, error) {
	h := &History{
		file:       historyFile,
		maxEntries: maxEntries,
	}

	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		file, err := os.Create(historyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create history file: %w", err)
		}
		file.Close()
	}

	return h, nil
}

func (h *History) Add(packageName, importPath string, action ActionType) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isDuplicate(packageName, importPath, action) {
		return nil
	}

	entry := Entry{
		Timestamp:  time.Now(),
		Package:    packageName,
		ImportPath: importPath,
		Action:     action,
	}

	entries, err := h.readEntries()
	if err != nil {
		return err
	}

	entries = append(entries, entry)
	if len(entries) > h.maxEntries {
		entries = entries[len(entries)-h.maxEntries:]
	}

	return h.writeEntries(entries)
}

func (h *History) GetRecent(n int) ([]Entry, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries, err := h.readEntries()
	if err != nil {
		return nil, err
	}

	if len(entries) <= n {
		return entries, nil
	}

	return entries[len(entries)-n:], nil
}

func (h *History) GetAll() ([]Entry, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.readEntries()
}

func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	file, err := os.OpenFile(h.file, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}
	defer file.Close()

	return nil
}

func (h *History) Search(query string) ([]Entry, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries, err := h.readEntries()
	if err != nil {
		return nil, err
	}

	var matches []Entry
	for _, entry := range entries {
		if contains(entry.Package, query) || contains(entry.ImportPath, query) {
			matches = append(matches, entry)
		}
	}

	return matches, nil
}

func (h *History) readEntries() ([]Entry, error) {
	file, err := os.Open(h.file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read history: %w", err)
	}

	return entries, nil
}

func (h *History) writeEntries(entries []Entry) error {
	tempFile := h.file + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp history file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}

		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write history entry: %w", err)
		}

		if _, err := writer.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush history: %w", err)
	}

	if err := os.Rename(tempFile, h.file); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

func (h *History) isDuplicate(packageName, importPath string, action ActionType) bool {
	entries, err := h.readEntries()
	if err != nil {
		return false
	}

	start := len(entries) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(entries); i++ {
		entry := entries[i]
		if entry.Package == packageName &&
			entry.ImportPath == importPath &&
			entry.Action == action {
			if time.Since(entry.Timestamp) < time.Hour {
				return true
			}
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || contains(s[1:], substr) || contains(s[:len(s)-1], substr))
}
