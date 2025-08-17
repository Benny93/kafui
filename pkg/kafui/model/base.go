package model

import (
    "github.com/charmbracelet/lipgloss"
)

// BaseModel represents the base state for all models
type BaseModel struct {
    styles  *Styles
    width   int
    height  int
}

// Styles holds the common styles used across models
type Styles struct {
    Title       lipgloss.Style
    Border      lipgloss.Style
    Selected    lipgloss.Style
    Normal      lipgloss.Style
    Error       lipgloss.Style
    StatusBar   lipgloss.Style
}

// NewStyles initializes default styles
func NewStyles() *Styles {
    return &Styles{
        Title: lipgloss.NewStyle().
            Bold(true).
            Foreground(lipgloss.Color("15")),
        Border: lipgloss.NewStyle().
            BorderStyle(lipgloss.RoundedBorder()),
        Selected: lipgloss.NewStyle().
            Background(lipgloss.Color("69")),
        Normal: lipgloss.NewStyle().
            Foreground(lipgloss.Color("15")),
        Error: lipgloss.NewStyle().
            Foreground(lipgloss.Color("9")),
        StatusBar: lipgloss.NewStyle().
            Background(lipgloss.Color("17")),
    }
}