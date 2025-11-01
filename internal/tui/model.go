package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MdSadiqMd/gopick/internal/cache"
	"github.com/MdSadiqMd/gopick/internal/config"
	"github.com/MdSadiqMd/gopick/internal/history"
	"github.com/MdSadiqMd/gopick/internal/packages"
	"github.com/MdSadiqMd/gopick/internal/scraper"
)

type ViewState int

const (
	ViewSearch ViewState = iota
	ViewOptions
	ViewInstalling
	ViewCommands
	ViewHelp
)

type Model struct {
	config     *config.Config
	cache      *cache.Cache
	history    *history.History
	scraper    *scraper.Scraper
	pkgManager *packages.Manager

	viewState   ViewState
	searchInput textinput.Model
	packages    []cache.Package
	cursor      int
	selected    map[int]bool
	message     string
	messageType string // "success", "error", "info"

	searching      bool
	searchDebounce *time.Timer
	lastQuery      string
	fromCache      bool

	installing      bool
	installProgress float64
	installMessage  string
	spinner         spinner.Model

	showHelp bool
	commands []string

	width  int
	height int

	recentHistory []history.Entry
	installedPkgs map[string]bool

	firstRun         bool
	quitWithCommands bool
	commandsToPrint  []string
	autoRun          bool
}

func New(cfg *config.Config, c *cache.Cache, h *history.History, pm *packages.Manager) *Model {
	ti := textinput.New()
	ti.Placeholder = "Search for Go packages..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(dimmedColor)
	ti.TextStyle = lipgloss.NewStyle().Foreground(fgColor)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	firstRun := false

	installedPkgs := make(map[string]bool)
	if allHistory, err := h.GetAll(); err == nil {
		for _, entry := range allHistory {
			if entry.Action == history.ActionInstalled {
				installedPkgs[entry.ImportPath] = true
			}
		}
	}

	return &Model{
		config:        cfg,
		cache:         c,
		history:       h,
		scraper:       scraper.New(),
		pkgManager:    pm,
		viewState:     ViewSearch,
		searchInput:   ti,
		selected:      make(map[int]bool),
		spinner:       s,
		firstRun:      firstRun,
		width:         80,
		height:        24,
		installedPkgs: installedPkgs,
	}
}

func (m *Model) Init() tea.Cmd {
	m.searchInput.Focus()

	cmds := []tea.Cmd{
		textinput.Blink,
		m.spinner.Tick,
	}

	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch m.viewState {
		case ViewSearch:
			if m.showHelp {
				m.showHelp = false
				m.searchInput.Focus()
				return m, nil
			}

			cmd := m.handleSearchKeys(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case ViewOptions:
			cmd := m.handleOptionsKeys(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case ViewCommands:
			cmd := m.handleCommandsKeys(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case ViewInstalling:
			// No key handling during installation
		}

	case searchResultsMsg:
		m.handleSearchResults(msg)

	case installProgressMsg:
		m.installProgress = msg.percent
		m.installMessage = msg.message
		if msg.done {
			m.viewState = ViewSearch
			m.installing = false
			m.message = "Installation completed successfully!"
			m.messageType = "success"
			// Clear selected packages
			m.selected = make(map[int]bool)
			// Refresh installed status
			m.packages = m.pkgManager.MarkInstalledPackages(m.packages)
			// Re-focus search input
			m.searchInput.Focus()
		}

	case installErrorMsg:
		m.viewState = ViewSearch
		m.installing = false
		m.message = fmt.Sprintf("Installation failed: %s", msg.err)
		m.messageType = "error"

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.viewState == ViewSearch && !m.showHelp && !m.installing {
		var cmd tea.Cmd
		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		newValue := m.searchInput.Value()

		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		if newValue != oldValue {
			m.lastQuery = newValue
			searchCmd := m.debounceSearch()
			if searchCmd != nil {
				cmds = append(cmds, searchCmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.firstRun {
		return m.renderWelcome()
	}

	switch m.viewState {
	case ViewInstalling:
		return m.renderInstalling()
	case ViewCommands:
		return m.renderCommands()
	case ViewOptions:
		return m.renderOptions()
	default:
		if m.showHelp {
			return m.renderHelp()
		}
		return m.renderSearch()
	}
}

func (m *Model) renderWelcome() string {
	welcome := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("🎯 Welcome to gopick!"),
		"",
		"The interactive Go package search and installation tool",
		"",
		helpStyle.Render("Press any key to start..."),
		"",
		helpStyle.Render("Quick tips:"),
		helpStyle.Render("  • Type to search for packages"),
		helpStyle.Render("  • Use ↑/↓ to navigate results"),
		helpStyle.Render("  • Press [space] to select multiple packages"),
		helpStyle.Render("  • Press [enter] to proceed with selected packages"),
		helpStyle.Render("  • Press [h] for help anytime"),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Width(60).Render(welcome))
}

func (m *Model) renderSearch() string {
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("🚀 gopick - Go Package Search"))
	content.WriteString("\n\n")

	// Search box
	searchBox := lipgloss.JoinHorizontal(lipgloss.Left,
		searchLabelStyle.Render("Search:"),
		m.searchInput.View(),
	)

	if m.searching {
		searchBox += " " + m.spinner.View()
	}

	content.WriteString(searchBox)
	content.WriteString("\n\n")

	// Results header
	if len(m.packages) > 0 {
		header := resultsHeaderStyle.Render(fmt.Sprintf("📦 Results (%d packages)", len(m.packages)))
		content.WriteString(header)
		content.WriteString("\n\n")

		visibleItems := m.getVisibleItems()
		for i, idx := range visibleItems {
			content.WriteString(m.renderPackageItem(idx))
			if i < len(visibleItems)-1 {
				content.WriteString("\n")
			}
		}
	} else if m.lastQuery != "" && !m.searching {
		content.WriteString(emptyStateStyle.Render("No packages found"))
	} else if len(m.recentHistory) > 0 && m.searchInput.Value() == "" {
		content.WriteString(resultsHeaderStyle.Render("📚 Recent History"))
		content.WriteString("\n\n")
		for _, entry := range m.recentHistory {
			histItem := fmt.Sprintf("  %s %s",
				helpStyle.Render(entry.Timestamp.Format("15:04")),
				packageNameStyle.Render(entry.Package))
			if entry.Action == history.ActionInstalled {
				histItem += lipgloss.NewStyle().Foreground(accentColor).Render(" ✓")
			}
			content.WriteString(histItem)
			content.WriteString("\n")
		}
	}

	if m.message != "" {
		content.WriteString("\n")
		switch m.messageType {
		case "success":
			content.WriteString(successMessageStyle.Render(m.message))
		case "error":
			content.WriteString(errorMessageStyle.Render(m.message))
		default:
			content.WriteString(infoMessageStyle.Render(m.message))
		}
	}

	content.WriteString("\n")
	content.WriteString(m.renderFooter())

	return appStyle.Width(m.width - 4).Render(content.String())
}

func (m *Model) renderPackageItem(idx int) string {
	pkg := m.packages[idx]
	isSelected := m.selected[idx]
	isCursor := idx == m.cursor

	var item strings.Builder

	// Selection indicator
	if isCursor {
		item.WriteString(selectedPackageStyle.Render(">"))
	} else {
		item.WriteString(" ")
	}

	// Checkbox
	item.WriteString(" " + RenderCheckbox(isSelected))

	// Package name
	name := packageNameStyle.Render(pkg.Name)
	if isCursor {
		name = selectedPackageStyle.Render(pkg.Name)
	}
	item.WriteString(" " + name)

	// Badges
	if pkg.IsInstalled {
		item.WriteString(installedBadge.Render("installed"))
	}
	if pkg.Version != "" {
		item.WriteString(" " + helpStyle.Render("v"+pkg.Version))
	}
	if m.installedPkgs[pkg.ImportPath] {
		item.WriteString(cachedBadge.Render("cached"))
	}

	item.WriteString("\n")

	// Description
	if pkg.Description != "" {
		desc := TruncateText(pkg.Description, 70)
		item.WriteString(packageDescStyle.Render(desc))
		item.WriteString("\n")
	}

	// Import path
	item.WriteString(packagePathStyle.Render(pkg.ImportPath))

	return item.String()
}

func (m *Model) renderOptions() string {
	selected := m.getSelectedPackages()

	title := dialogTitleStyle.Render(fmt.Sprintf("📦 %d package(s) selected", len(selected)))

	options := []string{
		"[G] Give me the command",
		"[D] Download for me",
		"[C] Cancel",
	}

	var optionList strings.Builder
	for _, opt := range options {
		optionList.WriteString(helpStyle.Render(opt))
		optionList.WriteString("\n")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		"What would you like to do?",
		"",
		optionList.String(),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(content))
}

func (m *Model) renderCommands() string {
	title := dialogTitleStyle.Render("📋 Installation Commands")

	var cmdList strings.Builder
	for _, cmd := range m.commands {
		cmdList.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E1E")).
			Foreground(accentColor).
			Padding(0, 1).
			Render(cmd))
		cmdList.WriteString("\n\n")
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		"Copy and run these commands:",
		"",
		cmdList.String(),
		helpStyle.Render("Press [ESC] to go back"),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Width(70).Render(content))
}

func (m *Model) renderInstalling() string {
	title := titleStyle.Render("📦 Installing Packages")

	progressBar := RenderProgressBar(m.installProgress, 40)

	message := m.installMessage
	if message == "" {
		message = "Preparing installation..."
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		m.spinner.View()+" "+message,
		"",
		progressBar,
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(content))
}

func (m *Model) renderHelp() string {
	helpText := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("⌨️  Keyboard Shortcuts"),
		"",
		lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Navigation & Search:"),
		m.renderHelpItem("Type", "Search packages (any letter/number/space)"),
		m.renderHelpItem("↑/↓", "Navigate results"),
		m.renderHelpItem("Tab", "Select/deselect package"),
		m.renderHelpItem("Enter", "Proceed with selected"),
		m.renderHelpItem("Esc", "Clear search / Quit if empty"),
		"",
		lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Commands (Shift+Key or Ctrl+Key):"),
		m.renderHelpItem("Shift+A", "Select all"),
		m.renderHelpItem("Shift+N", "Deselect all"),
		m.renderHelpItem("Shift+H", "Toggle help"),
		m.renderHelpItem("Shift+C", "Clear cache"),
		m.renderHelpItem("Shift+Q", "Quit"),
		"",
		helpStyle.Render("Press any key to close help..."),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(helpText))
}

func (m *Model) renderHelpItem(key, desc string) string {
	return fmt.Sprintf("%s %s",
		helpKeyStyle.Width(10).Render(key),
		helpDescStyle.Render(desc))
}

func (m *Model) renderFooter() string {
	selectedCount := len(m.getSelectedPackages())

	var footer strings.Builder
	footer.WriteString(footerStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			helpKeyStyle.Render("[↑↓]")+" Navigate  ",
			helpKeyStyle.Render("[Tab]")+" Select  ",
			helpKeyStyle.Render("[Enter]")+" Proceed  ",
			helpKeyStyle.Render("[Shift+H]")+" Help  ",
			helpKeyStyle.Render("[Shift+Q]")+" Quit",
		),
	))

	if selectedCount > 0 {
		footer.WriteString("\n")
		footer.WriteString(lipgloss.NewStyle().Foreground(accentColor).Render(fmt.Sprintf("✓ %d selected", selectedCount)))
	}

	return footer.String()
}

func (m *Model) getVisibleItems() []int {
	start := 0
	end := len(m.packages)

	maxVisible := (m.height - 15) / 4
	if maxVisible < 1 {
		maxVisible = 1
	}

	if end-start > maxVisible {
		start = m.cursor - maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > len(m.packages) {
			end = len(m.packages)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
	}

	result := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, i)
	}
	return result
}

func (m *Model) getSelectedPackages() []cache.Package {
	var selected []cache.Package
	for idx := range m.selected {
		if m.selected[idx] && idx < len(m.packages) {
			selected = append(selected, m.packages[idx])
		}
	}
	return selected
}
