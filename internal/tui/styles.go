package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor   = lipgloss.Color("#00D9FF") // Cyan
	secondaryColor = lipgloss.Color("#FF79C6") // Pink
	accentColor    = lipgloss.Color("#50FA7B") // Green
	warningColor   = lipgloss.Color("#FFB86C") // Orange
	errorColor     = lipgloss.Color("#FF5555") // Red

	bgColor        = lipgloss.Color("#0D1117") // Dark background
	fgColor        = lipgloss.Color("#C9D1D9") // Light gray text
	borderColor    = lipgloss.Color("#30363D") // Border gray
	selectedBg     = lipgloss.Color("#161B22") // Selected background
	dimmedColor    = lipgloss.Color("#8B949E") // Dimmed text
	highlightColor = lipgloss.Color("#58A6FF") // Link blue
)

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1).
			MarginBottom(1)

	searchLabelStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				MarginRight(1)

	resultsHeaderStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				MarginBottom(1).
				MarginTop(1)

	packageNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(highlightColor)

	packageDescStyle = lipgloss.NewStyle().
				Foreground(dimmedColor).
				MarginLeft(2)

	packagePathStyle = lipgloss.NewStyle().
				Foreground(fgColor).
				MarginLeft(2)

	selectedPackageStyle = lipgloss.NewStyle().
				Background(selectedBg).
				Foreground(primaryColor).
				Bold(true).
				PaddingLeft(1)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			MarginRight(1)

	uncheckedBoxStyle = lipgloss.NewStyle().
				Foreground(dimmedColor).
				MarginRight(1)

	installedBadge = lipgloss.NewStyle().
			Background(accentColor).
			Foreground(bgColor).
			Padding(0, 1).
			MarginLeft(1).
			Bold(true)

	cachedBadge = lipgloss.NewStyle().
			Background(warningColor).
			Foreground(bgColor).
			Padding(0, 1).
			MarginLeft(1).
			Bold(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				MarginTop(1).
				MarginBottom(1)

	progressTextStyle = lipgloss.NewStyle().
				Foreground(dimmedColor).
				MarginLeft(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimmedColor).
			MarginTop(1).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(dimmedColor)

	footerStyle = lipgloss.NewStyle().
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			MarginTop(1).
			Padding(1, 0)

	successMessageStyle = lipgloss.NewStyle().
				Background(accentColor).
				Foreground(bgColor).
				Padding(0, 2).
				MarginTop(1).
				Bold(true)

	errorMessageStyle = lipgloss.NewStyle().
				Background(errorColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 2).
				MarginTop(1).
				Bold(true)

	infoMessageStyle = lipgloss.NewStyle().
				Background(highlightColor).
				Foreground(bgColor).
				Padding(0, 2).
				MarginTop(1)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(50).
			Align(lipgloss.Center)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				MarginBottom(1).
				Align(lipgloss.Center)

	emptyStateStyle = lipgloss.NewStyle().
			Foreground(dimmedColor).
			Italic(true).
			MarginTop(2).
			MarginBottom(2).
			Align(lipgloss.Center)
)

func RenderProgressBar(percent float64, width int) string {
	if width <= 0 {
		width = 40
	}

	filled := int(float64(width) * (percent / 100))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return progressBarStyle.Render(bar) + progressTextStyle.Render(fmt.Sprintf(" %.0f%%", percent))
}

func RenderCheckbox(selected bool) string {
	if selected {
		return checkboxStyle.Render("[✓]")
	}
	return uncheckedBoxStyle.Render("[ ]")
}

func RenderBadge(text string, style lipgloss.Style) string {
	return style.Render(text)
}

func TruncateText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return text[:maxWidth]
	}
	return text[:maxWidth-3] + "..."
}
