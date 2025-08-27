package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"ui_example/ui"
)

func main() {
	// Use the new reusable app with default configuration
	app := ui.NewDefaultApp()
	
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		log.Fatal(err)
		os.Exit(1)
	}
}