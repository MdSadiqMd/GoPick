package tui

import (
	"fmt"

	"github.com/MdSadiqMd/gopick/internal/history"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleSearchKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit

	case tea.KeyEsc:
		if m.searchInput.Value() == "" {
			return tea.Quit
		}
		// clear search
		m.searchInput.SetValue("")
		m.lastQuery = ""
		m.packages = nil
		m.message = ""
		m.cursor = 0
		return nil

	case tea.KeyUp:
		if len(m.packages) > 0 && m.cursor > 0 {
			m.cursor--
		}
		return nil

	case tea.KeyDown:
		if len(m.packages) > 0 && m.cursor < len(m.packages)-1 {
			m.cursor++
		}
		return nil

	case tea.KeyTab:
		if len(m.packages) > 0 && m.cursor < len(m.packages) {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
		return nil

	case tea.KeyEnter:
		if m.firstRun {
			m.firstRun = false
			return nil
		}

		selected := m.getSelectedPackages()
		if len(selected) == 0 && m.cursor < len(m.packages) {
			// auto-select current item if nothing selected
			m.selected[m.cursor] = true
			selected = m.getSelectedPackages()
		}

		if len(selected) > 0 {
			m.viewState = ViewOptions
			for _, pkg := range selected {
				m.history.Add(pkg.Name, pkg.ImportPath, history.ActionViewed)
			}
		}
		return nil

	case tea.KeyCtrlH:
		m.showHelp = !m.showHelp
		return nil

	case tea.KeyCtrlQ:
		return tea.Quit

	case tea.KeyCtrlA:
		if len(m.packages) > 0 {
			for i := range m.packages {
				m.selected[i] = true
			}
		}
		return nil

	case tea.KeyCtrlN:
		m.selected = make(map[int]bool)
		return nil

	case tea.KeyRunes:
		// shift+letter commands
		runes := msg.Runes
		if len(runes) == 1 {
			switch runes[0] {
			case 'Q':
				return tea.Quit
			case 'H':
				m.showHelp = !m.showHelp
				return nil
			case 'A':
				if len(m.packages) > 0 {
					for i := range m.packages {
						m.selected[i] = true
					}
				}
				return nil
			case 'N':
				m.selected = make(map[int]bool)
				return nil
			case 'C':
				if err := m.cache.Clear(); err == nil {
					m.message = "Cache cleared successfully"
					m.messageType = "success"
				} else {
					m.message = fmt.Sprintf("Failed to clear cache: %v", err)
					m.messageType = "error"
				}
				return nil
			}
		}

		return nil
	}

	if m.message != "" && msg.String() != "" {
		m.message = ""
	}

	return nil
}

func (m *Model) handleOptionsKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc:
		m.viewState = ViewSearch
		m.searchInput.Focus()
		return nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "g", "G":
			selected := m.getSelectedPackages()
			command := m.pkgManager.GetInstallCommand(selected)
			if command != "" {
				m.quitWithCommands = true
				m.commandsToPrint = []string{command}
				m.autoRun = false
				return tea.Quit
			} else {
				m.message = "All selected packages are already installed"
				m.messageType = "info"
				m.viewState = ViewSearch
			}
			return nil

		case "d", "D":
			selected := m.getSelectedPackages()
			command := m.pkgManager.GetInstallCommand(selected)
			if command != "" {
				m.quitWithCommands = true
				m.commandsToPrint = []string{command}
				m.autoRun = true

				for _, pkg := range selected {
					m.history.Add(pkg.Name, pkg.ImportPath, history.ActionInstalled)
				}

				return tea.Quit
			}
			return nil

		case "c", "C":
			m.viewState = ViewSearch
			m.searchInput.Focus()
			return nil
		}
	}

	return nil
}

func (m *Model) handleCommandsKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyEnter:
		m.viewState = ViewSearch
		m.commands = nil
		m.searchInput.Focus()
		return nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q", "Q":
			m.viewState = ViewSearch
			m.commands = nil
			m.searchInput.Focus()
			return nil
		}
	}

	return nil
}
