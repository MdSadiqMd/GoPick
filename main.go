package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MdSadiqMd/gopick/internal/cache"
	"github.com/MdSadiqMd/gopick/internal/config"
	"github.com/MdSadiqMd/gopick/internal/history"
	"github.com/MdSadiqMd/gopick/internal/packages"
	"github.com/MdSadiqMd/gopick/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	c, err := cache.New(cfg.CacheDir, cfg.CacheTTLDays)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing cache: %v\n", err)
		os.Exit(1)
	}

	h, err := history.New(cfg.HistoryFile, cfg.MaxHistoryEntries)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing history: %v\n", err)
		os.Exit(1)
	}

	pm := packages.New(cfg.GoModCachePath)

	go c.CleanExpired()

	model := tui.New(cfg, c, h, pm)

	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running gopick: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(*tui.Model); ok {
		if m.ShouldPrintCommands() {
			commands := m.GetCommandsToPrint()

			if m.ShouldAutoRun() {
				fmt.Println("\n📦 Run this command to install packages:")
				fmt.Println()
				fullCmd := strings.Join(commands, " && ")
				fmt.Println("  " + fullCmd)
				fmt.Println()
				fmt.Println("Copy and run the command above ☝️")
			} else {
				fmt.Println("\n📦 Installation Commands:")
				fmt.Println()
				for _, cmd := range commands {
					fmt.Println("  " + cmd)
				}
				fmt.Println()
				fmt.Println("Copy and run these commands in your terminal.")
			}
		}
	}
}
