# Kafui Technical Implementation Plan

This document provides a detailed technical roadmap for implementing the UI improvements outlined in the UI Improvements Plan. It focuses on specific changes to the current codebase structure and implementation.

## 1. Theme and Styling System

### 1.1. Create Theme Package

Create a new theme package to manage styling consistently:

```go
// pkg/ui/theme/theme.go
package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme and styling for the application
type Theme struct {
    Primary   string
    Secondary string
    Accent    string
    Background string
    Foreground string
    Success   string
    Warning   string
    Error     string
    Info      string
}

// Default themes
var (
    LightTheme = Theme{
        Primary:    "#2E8BC0",
        Secondary:  "#145DA0",
        Accent:     "#B1D4E0",
        Background: "#FFFFFF",
        Foreground: "#000000",
        Success:    "#28A745",
        Warning:    "#FFC107",
        Error:      "#DC3545",
        Info:       "#17A2B8",
    }
    
    DarkTheme = Theme{
        Primary:    "#6CA6C1",
        Secondary:  "#87CEEB",
        Accent:     "#B0E0E6",
        Background: "#1E1E1E",
        Foreground: "#FFFFFF",
        Success:    "#90EE90",
        Warning:    "#FFD700",
        Error:      "#FF6347",
        Info:       "#87CEEB",
    }
)

// Styles defines reusable style components
type Styles struct {
    Header      lipgloss.Style
    Footer      lipgloss.Style
    Sidebar     lipgloss.Style
    MainPanel   lipgloss.Style
    Title       lipgloss.Style
    Subtitle    lipgloss.Style
    InfoText    lipgloss.Style
    SuccessText lipgloss.Style
    WarningText lipgloss.Style
    ErrorText   lipgloss.Style
}

// CreateStyles creates a new Styles instance based on a theme
func CreateStyles(theme Theme) *Styles {
    return &Styles{
        Header: lipgloss.NewStyle().
            Background(lipgloss.Color(theme.Primary)).
            Foreground(lipgloss.Color(theme.Background)).
            Padding(0, 1).
            Bold(true),
            
        Footer: lipgloss.NewStyle().
            Background(lipgloss.Color(theme.Secondary)).
            Foreground(lipgloss.Color(theme.Background)).
            Padding(0, 1),
            
        Sidebar: lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color(theme.Primary)).
            Padding(1),
            
        MainPanel: lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color(theme.Secondary)).
            Padding(1),
            
        Title: lipgloss.NewStyle().
            Foreground(lipgloss.Color(theme.Primary)).
            Bold(true),
            
        Subtitle: lipgloss.NewStyle().
            Foreground(lipgloss.Color(theme.Secondary)).
            Bold(true),
            
        InfoText: lipgloss.NewStyle().
            Foreground(lipgloss.Color(theme.Info)),
            
        SuccessText: lipgloss.NewStyle().
            Foreground(lipgloss.Color(theme.Success)),
            
        WarningText: lipgloss.NewStyle().
            Foreground(lipgloss.Color(theme.Warning)),
            
        ErrorText: lipgloss.NewStyle().
            Foreground(lipgloss.Color(theme.Error)),
    }
}
```

### 1.2. Update Existing Components to Use Theme

Update the detail page to use the new theme system:

```go
// pkg/ui/pages/detail/components.go

// View handles rendering for the detail page
type View struct {
    dimensions core.Dimensions
    theme      theme.Theme
    styles     *theme.Styles
}

// NewView creates a new View instance
func NewView() *View {
    theme := theme.DarkTheme // or detect from settings
    return &View{
        theme:  theme,
        styles: theme.CreateStyles(theme),
    }
}

```

## 2. Enhanced Page Interface

### 2.1. Update Core Page Interface

```go
// pkg/ui/core/interfaces.go
package core

import (
    "github.com/charmbracelet/bubbles/key"
    tea "github.com/charmbracelet/bubbletea"
)

// Page represents a UI page in the application
type Page interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetDimensions(width, height int)
    GetID() string
    
    // New methods for enhanced navigation
    GetTitle() string
    GetHelp() []key.Binding
    HandleNavigation(msg tea.Msg) (Page, tea.Cmd)
}
```

### 2.2. Update Existing Pages to Implement New Interface

```go
// pkg/ui/pages/detail/detail_page.go

// GetID implements the Page interface
func (m *Model) GetID() string {
    return "detail"
}

// GetTitle implements the Page interface
func (m *Model) GetTitle() string {
    return fmt.Sprintf("Message Detail: %s", m.topicName)
}

// GetHelp implements the Page interface
func (m *Model) GetHelp() []key.Binding {
    return []key.Binding{
        m.keys.bindings.Back,
        m.keys.bindings.ToggleFormat,
        m.keys.bindings.ToggleHeaders,
        m.keys.bindings.ToggleMetadata,
    }
}

// HandleNavigation implements the Page interface
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
    // Handle page-specific navigation
    return m, nil
}
```

## 3. Page Router Implementation

### 3.1. Create Router Package

```go
// pkg/ui/router/router.go
package router

import (
    "github.com/Benny93/kafui/pkg/api"
    "github.com/Benny93/kafui/pkg/ui/core"
    detailpage "github.com/Benny93/kafui/pkg/ui/pages/detail"
    mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
    topicpage "github.com/Benny93/kafui/pkg/ui/pages/topic"
    tea "github.com/charmbracelet/bubbletea"
)

// Router manages page navigation and state
type Router struct {
    dataSource api.KafkaDataSource
    pages      map[string]core.Page
    history    []string
    currentPage string
}

// NewRouter creates a new Router instance
func NewRouter(dataSource api.KafkaDataSource) *Router {
    return &Router{
        dataSource: dataSource,
        pages:      make(map[string]core.Page),
        history:    make([]string, 0),
        currentPage: "main",
    }
}

// NavigateTo switches to a specific page
func (r *Router) NavigateTo(pageID string, data interface{}) tea.Cmd {
    // Add current page to history
    if r.currentPage != "" {
        r.history = append(r.history, r.currentPage)
    }
    
    // Initialize page if needed
    if _, exists := r.pages[pageID]; !exists {
        r.pages[pageID] = r.createPage(pageID, data)
    }
    
    r.currentPage = pageID
    return r.pages[pageID].Init()
}

// Back navigates to the previous page
func (r *Router) Back() tea.Cmd {
    if len(r.history) > 0 {
        lastPage := r.history[len(r.history)-1]
        r.history = r.history[:len(r.history)-1]
        r.currentPage = lastPage
        return nil
    }
    return nil
}

// GetCurrentPage returns the current page
func (r *Router) GetCurrentPage() core.Page {
    return r.pages[r.currentPage]
}

// createPage creates a new page instance
func (r *Router) createPage(pageID string, data interface{}) core.Page {
    switch pageID {
    case "main":
        return mainpage.NewModel(r.dataSource)
    case "topic":
        // Extract topic data
        if topicData, ok := data.(map[string]interface{}); ok {
            name := topicData["name"].(string)
            topic := topicData["topic"].(api.Topic)
            return topicpage.NewModel(r.dataSource, name, topic)
        }
        return topicpage.NewModel(r.dataSource, "unknown", api.Topic{})
    case "detail":
        // Extract message data
        if messageData, ok := data.(api.Message); ok {
            return detailpage.NewModel(r.dataSource, "unknown", messageData)
        }
        return detailpage.NewModel(r.dataSource, "unknown", api.Message{})
    default:
        return mainpage.NewModel(r.dataSource)
    }
}
```

## 4. Enhanced Navigation System

### 4.1. Global Key Bindings

```go
// pkg/ui/core/keys.go
package core

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global key bindings
type GlobalKeyMap struct {
    Help      key.Binding
    Quit      key.Binding
    Back      key.Binding
    NextPage  key.Binding
    PrevPage  key.Binding
}

// DefaultGlobalKeys provides default global key bindings
var DefaultGlobalKeys = GlobalKeyMap{
    Help: key.NewBinding(
        key.WithKeys("?"),
        key.WithHelp("?", "help"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("ctrl+c", "q"),
        key.WithHelp("q/ctrl+c", "quit"),
    ),
    Back: key.NewBinding(
        key.WithKeys("esc"),
        key.WithHelp("esc", "back"),
    ),
    NextPage: key.NewBinding(
        key.WithKeys("tab"),
        key.WithHelp("tab", "next page"),
    ),
    PrevPage: key.NewBinding(
        key.WithKeys("shift+tab"),
        key.WithHelp("shift+tab", "previous page"),
    ),
}
```

### 4.2. Help System

```go
// pkg/ui/components/help/help.go
package help

import (
    "github.com/Benny93/kafui/pkg/ui/core"
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/lipgloss"
)

// Model represents the help view
type Model struct {
    width     int
    height    int
    isVisible bool
    keys      []key.Binding
}

// NewModel creates a new help model
func NewModel() *Model {
    return &Model{
        isVisible: false,
    }
}

// Show displays the help view
func (m *Model) Show(keys []key.Binding) {
    m.keys = keys
    m.isVisible = true
}

// Hide hides the help view
func (m *Model) Hide() {
    m.isVisible = false
}

// View renders the help view
func (m *Model) View() string {
    if !m.isVisible {
        return ""
    }
    
    // Create help content
    var helpLines []string
    helpLines = append(helpLines, "HELP")
    helpLines = append(helpLines, "====")
    helpLines = append(helpLines, "")
    
    for _, binding := range m.keys {
        helpLines = append(helpLines, 
            lipgloss.NewStyle().Bold(true).Render(binding.Help().Key)+": "+
            binding.Help().Desc)
    }
    
    // Style the help panel
    helpStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#6CA6C1")).
        Padding(1).
        Width(m.width - 4)
    
    return helpStyle.Render(lipgloss.JoinVertical(lipgloss.Left, helpLines...))
}
```

## 5. Status Bar Component

### 5.1. Create Status Bar

```go
// pkg/ui/components/statusbar/statusbar.go
package statusbar

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

// Model represents the status bar
type Model struct {
    width          int
    connectionStatus string
    messageCount   int
    selectedItems  int
    theme          Theme // Assuming Theme is defined elsewhere
}

// NewModel creates a new status bar model
func NewModel() *Model {
    return &Model{
        connectionStatus: "disconnected",
        messageCount:     0,
        selectedItems:    0,
    }
}

// SetConnectionStatus updates the connection status
func (m *Model) SetConnectionStatus(status string) {
    m.connectionStatus = status
}

// SetMessageCount updates the message count
func (m *Model) SetMessageCount(count int) {
    m.messageCount = count
}

// SetSelectedItems updates the selected items count
func (m *Model) SetSelectedItems(count int) {
    m.selectedItems = count
}

// View renders the status bar
func (m *Model) View() string {
    // Left section - connection status
    left := lipgloss.NewStyle().
        Foreground(lipgloss.Color(getStatusColor(m.connectionStatus))).
        Render(fmt.Sprintf("● %s", m.connectionStatus))
    
    // Right section - counts
    right := lipgloss.NewStyle().
        Align(lipgloss.Right).
        Render(fmt.Sprintf("Messages: %d | Selected: %d", 
            m.messageCount, m.selectedItems))
    
    // Combine sections
    statusBar := lipgloss.NewStyle().
        Background(lipgloss.Color("#145DA0")).
        Foreground(lipgloss.Color("#FFFFFF")).
        Padding(0, 1).
        Width(m.width)
    
    return statusBar.Render(
        lipgloss.Place(
            m.width, 1,
            lipgloss.Left, lipgloss.Center,
            left+" "+right,
        ),
    )
}

// Helper function to get color based on status
func getStatusColor(status string) string {
    switch status {
    case "connected":
        return "#28A745" // Green
    case "connecting":
        return "#FFC107" // Yellow
    case "disconnected":
        return "#6C757D" // Gray
    case "error":
        return "#DC3545" // Red
    default:
        return "#6C757D" // Gray
    }
}
```

## 6. Implementation Steps

### Phase 1: Foundation (Week 1-2)

1. **Create Theme System**
   - Implement `pkg/ui/theme` package
   - Define light/dark themes
   - Create styling utilities

2. **Update Page Interface**
   - Extend `core.Page` interface with new methods
   - Update existing pages to implement new interface
   - Maintain backward compatibility

3. **Implement Router**
   - Create `pkg/ui/router` package
   - Implement page navigation and history
   - Add page lifecycle management

### Phase 2: Navigation and UX (Week 3-4)

1. **Global Key Bindings**
   - Implement global key binding system
   - Add context-sensitive help
   - Create help component

2. **Status Bar**
   - Implement status bar component
   - Add connection status indicators
   - Display message and selection counts

3. **Breadcrumb Navigation**
   - Add breadcrumb component
   - Implement navigation history
   - Create visual navigation trail

### Phase 3: Component Architecture (Week 5-6)

1. **Refactor Components**
   - Organize components into logical packages
   - Create reusable UI components
   - Implement component state management

2. **Modal System**
   - Create modal dialog component
   - Implement confirmation dialogs
   - Add settings and error modals

3. **Loading States**
   - Add loading indicators
   - Implement progress bars
   - Create skeleton screens

### Phase 4: Polish and Advanced Features (Week 7-8)

1. **Visual Polish**
   - Add animations and transitions
   - Improve typography and spacing
   - Add visual feedback for interactions

2. **Advanced Search**
   - Implement fuzzy search
   - Add search history
   - Create filter presets


## 7. Testing Strategy

### Unit Tests
- Test theme and styling functions
- Verify page interface compliance
- Test router navigation logic
- Validate component rendering

### Integration Tests
- Test page transitions
- Verify navigation history
- Check global key bindings
- Validate help system

### Visual Regression Tests
- Capture UI state screenshots
- Compare against baseline images
- Test different terminal sizes
- Validate theme switching

## 8. Migration Plan

### Backward Compatibility
- Maintain existing API contracts
- Provide deprecation warnings
- Support legacy configuration
- Gradual migration of components

### Deprecation Timeline
- Phase 1: Introduce new systems alongside old ones
- Phase 2: Mark old systems as deprecated
- Phase 3: Remove old systems after stabilization

## 9. Performance Considerations

### Memory Management
- Implement component caching
- Use object pooling for frequently created objects
- Optimize rendering performance
- Monitor memory usage

### Rendering Optimization
- Implement virtual rendering for large lists
- Use viewport components for scrollable content
- Optimize layout calculations
- Minimize redraw operations

## 10. Documentation Updates

### User Documentation
- Update README with new features
- Create user guide for navigation
- Document key bindings and shortcuts
- Provide migration guide

### Developer Documentation
- Update architecture documentation
- Document new component APIs
- Provide implementation examples
- Create contribution guidelines

This technical implementation plan provides a detailed roadmap for enhancing the Kafui UI while maintaining code quality and user experience.