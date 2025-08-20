package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ResourceType represents different types of resources
type ResourceType int

const (
	TopicResourceType ResourceType = iota
	ConsumerGroupResourceType
	SchemaResourceType
	ContextResourceType
)

func (r ResourceType) String() string {
	switch r {
	case TopicResourceType:
		return "topics"
	case ConsumerGroupResourceType:
		return "consumer-groups"
	case SchemaResourceType:
		return "schemas"
	case ContextResourceType:
		return "contexts"
	default:
		return "unknown"
	}
}

// SidebarConfig holds configuration for the sidebar
type SidebarConfig struct {
	Context         string
	CurrentResource ResourceType
	ShowResources   bool
	ShowShortcuts   bool
	CustomSections  []SidebarSection
}

// SidebarSection represents a custom section in the sidebar
type SidebarSection struct {
	Title   string
	Content string
}

// Sidebar represents a reusable sidebar component
type Sidebar struct {
	config SidebarConfig
}

// NewSidebar creates a new sidebar component
func NewSidebar(config SidebarConfig) *Sidebar {
	return &Sidebar{config: config}
}

// RenderContext renders the context section
func (s *Sidebar) RenderContext() string {
	if s.config.Context == "" {
		return ""
	}
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		TitleStyle.Render("CONTEXT"),
		InfoStyle.Render(s.config.Context),
		lipgloss.NewStyle().MarginTop(2).Render(""),
	)
}

// RenderResourceButtons renders the resource selection buttons
func (s *Sidebar) RenderResourceButtons() string {
	if !s.config.ShowResources {
		return ""
	}
	
	resources := []struct {
		name string
		key  string
		typ  ResourceType
	}{
		{"Topics", "F1", TopicResourceType},
		{"Consumer Groups", "F2", ConsumerGroupResourceType},
		{"Schemas", "F3", SchemaResourceType},
		{"Contexts", "F4", ContextResourceType},
	}

	buttons := make([]string, len(resources))
	for i, res := range resources {
		style := InfoStyle
		if s.config.CurrentResource == res.typ {
			style = lipgloss.NewStyle().
				Foreground(Special).
				Bold(true)
		}
		
		buttons[i] = style.Render(fmt.Sprintf("%s %s", res.key, res.name))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		SubtitleStyle.Render("RESOURCES"),
		lipgloss.NewStyle().MarginBottom(1).Render("Press to switch:"),
		lipgloss.JoinVertical(lipgloss.Left, buttons...),
		lipgloss.NewStyle().MarginTop(2).Render(""),
	)
}

// RenderShortcuts renders the keyboard shortcuts section
func (s *Sidebar) RenderShortcuts() string {
	if !s.config.ShowShortcuts {
		return ""
	}
	
	shortcuts := []string{
		"↑/↓   Navigate items",
		"Enter   Select item",
		"/       Search",
		"Esc     Cancel search",
		"q       Quit",
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		SubtitleStyle.Render("SHORTCUTS"),
		lipgloss.JoinVertical(lipgloss.Left, shortcuts...),
	)
}

// RenderCustomSections renders any custom sections
func (s *Sidebar) RenderCustomSections() string {
	if len(s.config.CustomSections) == 0 {
		return ""
	}
	
	var sections []string
	for _, section := range s.config.CustomSections {
		sectionContent := lipgloss.JoinVertical(
			lipgloss.Left,
			SubtitleStyle.Render(strings.ToUpper(section.Title)),
			section.Content,
			lipgloss.NewStyle().MarginTop(1).Render(""),
		)
		sections = append(sections, sectionContent)
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// Render renders the complete sidebar
func (s *Sidebar) Render() string {
	var sections []string
	
	// Add context section
	if contextSection := s.RenderContext(); contextSection != "" {
		sections = append(sections, contextSection)
	}
	
	// Add resource buttons section
	if resourceSection := s.RenderResourceButtons(); resourceSection != "" {
		sections = append(sections, resourceSection)
	}
	
	// Add shortcuts section
	if shortcutsSection := s.RenderShortcuts(); shortcutsSection != "" {
		sections = append(sections, shortcutsSection)
	}
	
	// Add custom sections
	if customSections := s.RenderCustomSections(); customSections != "" {
		sections = append(sections, customSections)
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// UpdateConfig updates the sidebar configuration
func (s *Sidebar) UpdateConfig(config SidebarConfig) {
	s.config = config
}

// GetConfig returns the current sidebar configuration
func (s *Sidebar) GetConfig() SidebarConfig {
	return s.config
}

// SetCurrentResource updates the current resource type
func (s *Sidebar) SetCurrentResource(resourceType ResourceType) {
	s.config.CurrentResource = resourceType
}

// SetContext updates the context
func (s *Sidebar) SetContext(context string) {
	s.config.Context = context
}