package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"leetcli/internal/config"
	"leetcli/internal/ui"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}
	if cfg == nil {
		cfg = config.Default()
		cfg.ResolveBaseDir()
	}

	model := ui.New(cfg)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
