package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Benny93/kafui/pkg/ui//ui"
	tea "github.com/charmbracelet/bubbletea"
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
