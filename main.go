package main

import (
	"fmt"
	"os"

	"mkm/internal/config"
	"mkm/internal/parser"
	"mkm/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	// First run: auto-discover projects in cwd and add to a "local" group.
	if len(cfg.Groups) == 0 {
		root, _ := os.Getwd()
		cfg.AddGroup("local")
		projects, _ := parser.ScanProjects(root)
		for _, p := range projects {
			cfg.AddProject(0, p.Dir)
		}
		_ = cfg.Save()
	}

	model := ui.New(cfg)

	p := tea.NewProgram(
		&model,
		tea.WithAltScreen(),
	)

	model.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running TUI:", err)
		os.Exit(1)
	}
}
