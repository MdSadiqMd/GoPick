package tui

import (
	"time"

	"github.com/MdSadiqMd/gopick/internal/cache"
	tea "github.com/charmbracelet/bubbletea"
)

type searchResultsMsg struct {
	packages  []cache.Package
	fromCache bool
	err       error
}

type installProgressMsg struct {
	percent float64
	message string
	done    bool
}

type installErrorMsg struct {
	err error
}

func (m *Model) debounceSearch() tea.Cmd {
	if m.searchDebounce != nil {
		m.searchDebounce.Stop()
	}

	m.searching = true
	query := m.searchInput.Value()

	if query == "" {
		m.packages = nil
		m.searching = false
		return nil
	}

	m.searchDebounce = time.NewTimer(m.config.GetDebounceTime())

	return func() tea.Msg {
		<-m.searchDebounce.C
		return m.performSearch(query)()
	}
}

func (m *Model) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		if cached, found := m.cache.Get(query); found {
			packages := m.pkgManager.MarkInstalledPackages(cached.Results)
			return searchResultsMsg{
				packages:  packages,
				fromCache: true,
			}
		}

		packages, err := m.scraper.Search(query)
		if err != nil {
			if cached, found := m.cache.Get(query); found {
				packages = cached.Results
			} else {
				return searchResultsMsg{err: err}
			}
		}

		packages = m.pkgManager.MarkInstalledPackages(packages)
		if err == nil {
			m.cache.Set(query, packages)
		}

		return searchResultsMsg{
			packages:  packages,
			fromCache: false,
		}
	}
}

func (m *Model) handleSearchResults(msg searchResultsMsg) {
	m.searching = false
	if msg.err != nil {
		m.message = "Search failed: " + msg.err.Error()
		m.messageType = "error"
		return
	}

	m.packages = msg.packages
	m.fromCache = msg.fromCache
	m.cursor = 0
	m.selected = make(map[int]bool)

	if len(msg.packages) == 0 {
		m.message = "No packages found"
		m.messageType = "info"
	} else {
		m.message = ""
	}
}

func ShowMessage(message, messageType string) tea.Cmd {
	return func() tea.Msg {
		return struct {
			message     string
			messageType string
		}{message, messageType}
	}
}
