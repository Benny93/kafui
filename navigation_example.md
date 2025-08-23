# Kafui Improved Navigation Example

This example demonstrates how the improved navigation system would work in Kafui, inspired by the patterns seen in Crush and other Charm tools.

## Page Interface Implementation

```go
// pkg/ui/core/page.go
package core

import (
    "github.com/charmbracelet/bubbles/key"
    tea "github.com/charmbracelet/bubbletea"
)

// Page represents a navigable UI page
type Page interface {
    // Standard Bubble Tea methods
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetDimensions(width, height int)
    
    // Page identification
    GetID() string
    GetTitle() string
    
    // Navigation support
    GetHelp() []key.Binding
    HandleNavigation(msg tea.Msg) (Page, tea.Cmd)
    
    // Lifecycle methods
    OnFocus() tea.Cmd
    OnBlur() tea.Cmd
}
```

## Router Implementation

```go
// pkg/ui/router/router.go
package router

import (
    "github.com/Benny93/kafui/pkg/ui/core"
    tea "github.com/charmbracelet/bubbletea"
)

// Router manages page navigation
type Router struct {
    pages       map[string]core.Page
    history     []string
    currentPage string
    width       int
    height      int
}

// NavigateTo switches to a specific page
func (r *Router) NavigateTo(pageID string, data interface{}) tea.Cmd {
    // Add current page to history if it exists
    if r.currentPage != "" {
        r.history = append(r.history, r.currentPage)
    }
    
    // Create page if it doesn't exist
    if _, exists := r.pages[pageID]; !exists {
        page := r.createPage(pageID, data)
        if page != nil {
            r.pages[pageID] = page
        }
    }
    
    // Set current page and focus
    oldPage := r.pages[r.currentPage]
    newPage := r.pages[pageID]
    
    r.currentPage = pageID
    
    var cmds []tea.Cmd
    
    // Blur old page
    if oldPage != nil {
        cmds = append(cmds, oldPage.OnBlur())
    }
    
    // Focus new page
    if newPage != nil {
        newPage.SetDimensions(r.width, r.height)
        cmds = append(cmds, newPage.OnFocus())
    }
    
    return tea.Batch(cmds...)
}

// Back navigates to the previous page
func (r *Router) Back() tea.Cmd {
    if len(r.history) > 0 {
        lastPage := r.history[len(r.history)-1]
        r.history = r.history[:len(r.history)-1]
        return r.NavigateTo(lastPage, nil)
    }
    return nil
}

// GetCurrentPage returns the current page
func (r *Router) GetCurrentPage() core.Page {
    return r.pages[r.currentPage]
}
```

## Main Application Model

```go
// pkg/ui/app.go
package ui

import (
    "github.com/Benny93/kafui/pkg/ui/core"
    "github.com/Benny93/kafui/pkg/ui/router"
    tea "github.com/charmbracelet/bubbletea"
)

// Model represents the main application state
type Model struct {
    router      *router.Router
    showHelp    bool
    helpPage    core.Page // Help page instance
    globalKeys  core.GlobalKeyMap
}

// NewModel creates a new application model
func NewModel(dataSource api.KafkaDataSource) Model {
    r := router.NewRouter(dataSource)
    
    return Model{
        router:     r,
        globalKeys: core.DefaultGlobalKeys,
    }
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
    return m.router.NavigateTo("main", nil)
}

// Update handles application messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    // Handle global key bindings first
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, m.globalKeys.Help):
            m.showHelp = !m.showHelp
            return m, nil
        case key.Matches(msg, m.globalKeys.Quit):
            return m, tea.Quit
        case key.Matches(msg, m.globalKeys.Back):
            if !m.showHelp {
                return m, m.router.Back()
            }
        }
    case tea.WindowSizeMsg:
        m.router.SetDimensions(msg.Width, msg.Height)
    }
    
    // Handle current page updates
    if !m.showHelp {
        currentPage := m.router.GetCurrentPage()
        if currentPage != nil {
            updatedPage, cmd := currentPage.Update(msg)
            if updatedPage != nil {
                // Handle page changes from within the page
                // (e.g., when a page wants to navigate to another page)
                m.router.UpdatePage(currentPage.GetID(), updatedPage)
            }
            cmds = append(cmds, cmd)
        }
    }
    
    return m, tea.Batch(cmds...)
}

// View renders the application
func (m Model) View() string {
    if m.showHelp {
        // Show help overlay
        return m.renderHelp()
    }
    
    currentPage := m.router.GetCurrentPage()
    if currentPage != nil {
        return currentPage.View()
    }
    
    return "Loading..."
}

// renderHelp shows the help overlay
func (m Model) renderHelp() string {
    currentPage := m.router.GetCurrentPage()
    if currentPage != nil {
        helpKeys := currentPage.GetHelp()
        // Render help using the help component
        // This is a simplified example
        return m.helpPage.View()
    }
    return ""
}
```

## Topic Page Implementation with Navigation

```go
// pkg/ui/pages/topic/topic_page.go
package topic

import (
    "github.com/Benny93/kafui/pkg/api"
    "github.com/Benny93/kafui/pkg/ui/core"
    tea "github.com/charmbracelet/bubbletea"
)

// Model represents the topic page
type Model struct {
    topicName string
    message   *api.Message
    // ... other fields
}

// GetID returns the page ID
func (m *Model) GetID() string {
    return "topic"
}

// GetTitle returns the page title
func (m *Model) GetTitle() string {
    return "Topic: " + m.topicName
}

// GetHelp returns the page-specific help
func (m *Model) GetHelp() []key.Binding {
    return []key.Binding{
        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view message details")),
        key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "pause/resume")),
        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry connection")),
        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search messages")),
    }
}

// HandleNavigation handles page-specific navigation
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            if m.message != nil {
                // Navigate to detail page with message data
                return nil, func() tea.Msg {
                    return core.PageNavigationMsg{
                        TargetPage: "detail",
                        Data:       *m.message,
                    }
                }
            }
        }
    }
    return m, nil
}

// OnFocus is called when the page gains focus
func (m *Model) OnFocus() tea.Cmd {
    // Resume message consumption or other focus-related actions
    return nil
}

// OnBlur is called when the page loses focus
func (m *Model) OnBlur() tea.Cmd {
    // Pause message consumption or other blur-related actions
    return nil
}
```

## Message Passing for Navigation

```go
// pkg/ui/core/messages.go
package core

import tea "github.com/charmbracelet/bubbletea"

// PageNavigationMsg represents a request to navigate to another page
type PageNavigationMsg struct {
    TargetPage string
    Data       interface{}
}

// PageChangeMsg represents a page change notification
type PageChangeMsg struct {
    PageID string
    Data   interface{}
}
```

## Benefits of This Approach

1. **Consistent Navigation**: All pages implement the same interface, making navigation predictable
2. **Page Lifecycle Management**: Pages can respond to focus/blur events
3. **History Management**: Built-in back navigation with history stack
4. **Global Key Bindings**: Consistent key bindings across all pages
5. **Help System Integration**: Each page can provide context-sensitive help
6. **Modular Design**: Pages are self-contained and can be developed independently
7. **Performance**: Pages can optimize their behavior based on focus state

This pattern provides a solid foundation for building a professional, maintainable TUI application that follows the best practices seen in successful Bubble Tea applications.