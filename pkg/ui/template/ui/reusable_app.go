package ui

import (
	"github.com/Benny93/kafui/pkg/ui/template/ui/components"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/Benny93/kafui/pkg/ui/template/ui/styles"
	"github.com/charmbracelet/bubbles/help"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	DefaultCompactModeWidthBreakpoint  = 120 // Width at which the app switches to compact mode
	DefaultCompactModeHeightBreakpoint = 30  // Height at which the app switches to compact mode
	DefaultSideBarWidth                = 31  // Fixed width of the sidebar (CRUSH standard)
	DefaultHeaderHeight                = 1   // Height of the header
)

// ReusableApp is the configurable version of the CRUSH UI framework
type ReusableApp struct {
	width, height int

	header  components.Header
	content components.Content
	sidebar components.Sidebar
	footer  *components.Footer

	config      *providers.AppConfig
	sizeMode    styles.SizeMode
	showSidebar bool
	showDebug   bool
	showHelp    bool
}

// NewReusableApp creates a new app with the provided configuration
func NewReusableApp(config *providers.AppConfig) *ReusableApp {
	// Set defaults if not provided
	if config.CompactModeWidthBreakpoint == 0 {
		config.CompactModeWidthBreakpoint = DefaultCompactModeWidthBreakpoint
	}
	if config.CompactModeHeightBreakpoint == 0 {
		config.CompactModeHeightBreakpoint = DefaultCompactModeHeightBreakpoint
	}

	// Create components with providers
	var header components.Header
	if config.HeaderDataProvider != nil {
		header = components.NewHeaderWithProvider(config.HeaderDataProvider)
	} else {
		header = components.NewHeader()
	}

	var content components.Content
	if config.ContentProvider != nil {
		content = components.NewContentWithProvider(config.ContentProvider)
	} else {
		content = components.NewContent()
	}

	var sidebar components.Sidebar
	if len(config.SidebarSections) > 0 {
		sidebar = components.NewSidebarWithSections(config.SidebarSections)
	} else {
		sidebar = components.NewSidebar()
	}

	return &ReusableApp{
		header:      header,
		content:     content,
		sidebar:     sidebar,
		footer:      components.NewFooter(),
		config:      config,
		showSidebar: config.ShowSidebarByDefault,
		sizeMode:    styles.SizeModeNormal,
		showDebug:   false,
		showHelp:    false,
	}
}

// NewDefaultApp creates an app with default providers (same as original example)
func NewDefaultApp() *ReusableApp {
	config := &providers.AppConfig{
		ContentProvider:    providers.NewDefaultContentProvider(),
		HeaderDataProvider: providers.NewDefaultHeaderDataProvider(),
		SidebarSections: []providers.SidebarSection{
			providers.NewFilesSection(),
			providers.NewServersSection(),
			providers.NewStatusSection(),
		},
		ShowSidebarByDefault:        true,
		CompactModeWidthBreakpoint:  DefaultCompactModeWidthBreakpoint,
		CompactModeHeightBreakpoint: DefaultCompactModeHeightBreakpoint,
	}

	return NewReusableApp(config)
}

func (a *ReusableApp) Init() tea.Cmd {
	return tea.Batch(
		a.header.Init(),
		a.content.Init(),
		a.sidebar.Init(),
		a.footer.Init(),
	)
}

func (a *ReusableApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Determine size mode based on window dimensions
		a.sizeMode = styles.GetSizeMode(a.width, a.height)

		// Hide sidebar in small modes or if screen is too narrow
		if a.sizeMode <= styles.SizeModeCompact || a.width < (DefaultSideBarWidth+50) {
			a.showSidebar = false
		} else if a.config.ShowSidebarByDefault {
			a.showSidebar = true
		}

		cmds = append(cmds, a.header.SetSize(a.width, DefaultHeaderHeight))
		cmds = append(cmds, a.updateContentSize())
		cmds = append(cmds, a.updateSidebarSize())
		cmds = append(cmds, a.footer.SetSize(a.width, 1)) // Footer height is 1

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "ctrl+s", "t":
			// Only allow sidebar toggle in normal/big modes
			if a.sizeMode >= styles.SizeModeNormal {
				a.showSidebar = !a.showSidebar
				cmds = append(cmds, a.updateContentSize())
				cmds = append(cmds, a.updateSidebarSize())
			}
		case "ctrl+r":
			if refreshable, ok := a.sidebar.(components.Refreshable); ok {
				cmds = append(cmds, refreshable.Refresh())
			}
		case "ctrl+d":
			a.showDebug = !a.showDebug
		case "?":
			a.showHelp = !a.showHelp
			a.footer.ToggleShowAll()
		}
	}

	// Update components
	var cmd tea.Cmd
	headerComponent, cmd := a.header.Update(msg)
	a.header = headerComponent.(components.Header)
	cmds = append(cmds, cmd)

	contentComponent, cmd := a.content.Update(msg)
	a.content = contentComponent.(components.Content)
	cmds = append(cmds, cmd)

	sidebarComponent, cmd := a.sidebar.Update(msg)
	a.sidebar = sidebarComponent.(components.Sidebar)
	cmds = append(cmds, cmd)

	footerComponent, cmd := a.footer.Update(msg)
	a.footer = footerComponent.(*components.Footer)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a *ReusableApp) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Handle minimum size mode - show "Window too small!" message
	if a.sizeMode == styles.SizeModeMinimum {
		return a.renderMinimumSizeView()
	}

	var components []string

	// Add debug info if enabled
	if a.showDebug {
		debugInfo := styles.DebugInfo("App", a.width, a.height)
		components = append(components, debugInfo)
	}

	// Header (skip in small mode)
	if a.sizeMode > styles.SizeModeSmall {
		header := a.header.View()
		components = append(components, header)
	}

	// Main content area
	var mainContent string
	if a.showSidebar && a.sizeMode >= styles.SizeModeNormal {
		// CRUSH layout: Content on left, fixed-width sidebar on right
		contentView := a.content.View()
		sidebarView := a.sidebar.View()

		// Create the main layout with content on left, sidebar on right
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Bottom,
			contentView,
			sidebarView,
		)
	} else {
		// Full width content
		mainContent = a.content.View()
	}

	components = append(components, mainContent)

	// Footer
	footer := a.footer.View()
	if footer != "" {
		components = append(components, footer)
	}

	// Combine all components
	return lipgloss.JoinVertical(
		lipgloss.Left,
		components...,
	)
}

// renderMinimumSizeView renders the "Window too small!" message
func (a *ReusableApp) renderMinimumSizeView() string {
	t := styles.CurrentTheme()

	message := t.S().Base.
		Padding(1, 4).
		Foreground(t.White).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Render("Window too small!")

	return t.S().Base.
		Width(a.width).
		Height(a.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(message)
}

func (a *ReusableApp) updateContentSize() tea.Cmd {
	contentWidth := a.width
	if a.showSidebar {
		contentWidth = a.width - DefaultSideBarWidth
	}
	contentHeight := a.height - DefaultHeaderHeight

	return a.content.SetSize(contentWidth, contentHeight)
}

func (a *ReusableApp) updateSidebarSize() tea.Cmd {
	if !a.showSidebar {
		return nil
	}

	sidebarHeight := a.height - DefaultHeaderHeight - 2
	if a.showDebug {
		sidebarHeight -= 1 // Account for debug info
	}

	cmds := []tea.Cmd{
		a.sidebar.SetSize(DefaultSideBarWidth, sidebarHeight),
		a.sidebar.SetCompactMode(a.sizeMode <= styles.SizeModeCompact),
	}

	return tea.Batch(cmds...)
}

// SetKeyMap sets the key map for the footer help display
func (a *ReusableApp) SetKeyMap(keyMap help.KeyMap) {
	a.footer.SetKeyMap(keyMap)
}

// ToggleHelp toggles the help display
func (a *ReusableApp) ToggleHelp() {
	a.showHelp = !a.showHelp
	a.footer.ToggleShowAll()
}
