# Kafui Navigation and Key Input Handling Improvement Plan

This document outlines a comprehensive plan to enhance the navigation and key input handling systems in Kafui, focusing on robustness and maintainability. The plan builds upon the existing modular architecture and follows the patterns established in the UI improvements plan.

## 1. Current State Analysis

### 1.1. Navigation System
The current navigation system uses a page-based approach with a simple enum-based state machine in the root UI controller:

```go
type pageType int

const (
    mainPageType pageType = iota
    topicPageType
    detailPageType
    resourceDetailPageType
)
```

Navigation flow:
1. **Main Page** - Shows available Kafka resources (topics, consumer groups, etc.)
2. **Topic Page** - Shows messages within a specific topic
3. **Message Detail Page** - Shows details of a selected message
4. **Resource Detail Page** - Shows details of other Kafka resources (consumer groups, etc.)

Navigation is handled through direct state changes and page initialization in the main UI model's Update function.

### 1.2. Key Input Handling
Key input handling is distributed across individual page models with some global key bindings defined in the root UI controller:

```go
var keys = keyMap{
    Search: key.NewBinding(
        key.WithKeys(":"),
        key.WithHelp(":", "search"),
    ),
    TopicMode: key.NewBinding(
        key.WithKeys("/"),
        key.WithHelp("/", "filter topics"),
    ),
    Back: key.NewBinding(
        key.WithKeys("esc"),
        key.WithHelp("esc", "back"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("ctrl+c"),
        key.WithHelp("ctrl+c", "quit"),
    ),
}
```

### 1.3. Identified Issues
1. **Tight Coupling**: Pages are tightly coupled to the root UI controller
2. **Limited Navigation History**: No proper back stack implementation
3. **Inconsistent Key Handling**: Global and local key bindings are not well integrated
4. **No Context-Sensitive Help**: Pages don't provide help information
5. **No Focus Management**: No concept of focused components or focus navigation
6. **Confusing Naming**: The "detail" page actually shows message details, not topic details

## 2. Improvement Goals

### 2.1. Robustness Goals
- Implement a reliable navigation system with proper history management
- Create a consistent key input handling mechanism
- Add error recovery for navigation failures
- Implement proper focus management for components

### 2.2. Maintainability Goals
- Decouple pages from the root UI controller
- Create a standardized page interface for navigation
- Implement a centralized router for page management
- Establish clear patterns for key binding definitions

## 3. Proposed Architecture

### 3.1. Enhanced Page Interface
Extend the existing `Page` interface with navigation-specific methods:

```go
// pkg/ui/core/interfaces.go
type Page interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetDimensions(width, height int)
    GetID() string
    
    // New navigation methods
    GetTitle() string
    GetHelp() []key.Binding
    HandleNavigation(msg tea.Msg) (Page, tea.Cmd)
    OnFocus() tea.Cmd
    OnBlur() tea.Cmd
}
```

### 3.2. Page Types Clarification
To improve clarity and maintainability, we should rename the pages to better reflect their purpose:

1. **Main Page** (`main`) - Shows available Kafka resources (topics, consumer groups, etc.)
2. **Topic Page** (`topic`) - Shows messages within a specific topic
3. **Message Detail Page** (`message_detail`) - Shows details of a selected message
4. **Resource Detail Page** (`resource_detail`) - Shows details of other Kafka resources (consumer groups, etc.)

The current "detail" page actually shows details of individual Kafka messages, not topics. This naming is confusing because:
- Users navigate from Main Page → Topic Page → Message Detail Page
- The Topic Page shows a list of messages within a topic
- The Message Detail Page shows the details of a specific message

By renaming "detail" to "message_detail", we make the navigation flow more intuitive:
- **Main** (resource list) → **Topic** (message list) → **Message Detail** (message details)

This renaming will make the codebase more intuitive and easier to understand.

### 3.3. Router Implementation
Create a centralized router to manage page navigation and history:

```go
// pkg/ui/router/router.go
type Router struct {
    dataSource  api.KafkaDataSource
    pages       map[string]core.Page
    history     []string
    currentPage string
    width       int
    height      int
}

func (r *Router) NavigateTo(pageID string, data interface{}) tea.Cmd
func (r *Router) Back() tea.Cmd
func (r *Router) GetCurrentPage() core.Page
func (r *Router) SetDimensions(width, height int)
```

### 3.4. Global Key Binding System
Implement a centralized key binding system:

```go
// pkg/ui/core/keys.go
type GlobalKeyMap struct {
    Help      key.Binding
    Quit      key.Binding
    Back      key.Binding
    NextPage  key.Binding
    PrevPage  key.Binding
}

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
}
```

## 4. Implementation Plan

### Phase 1: Foundation (Week 1)

#### 4.1. Enhanced Page Interface
1. Update the `core.Page` interface with new navigation methods
2. Implement default empty implementations for new methods in existing pages
3. Update all existing pages to implement the new interface methods

#### 4.2. Page Naming Refactor
1. Rename the `detail` package to `message_detail` to better reflect its purpose
2. Update all references to the detail page throughout the codebase
3. Update page IDs to use descriptive names:
   - `main` for the main page
   - `topic` for the topic page
   - `message_detail` for the message detail page
   - `resource_detail` for the resource detail page

#### 4.3. Router Implementation
1. Create the `pkg/ui/router` package
2. Implement the Router struct with basic navigation functionality
3. Add page creation logic for all page types
4. Implement history management

#### 4.4. Global Key Binding System
1. Create the global key binding system in `pkg/ui/core/keys.go`
2. Define standard global key bindings
3. Implement help system component

### Phase 2: Integration (Week 2)

#### 4.5. Root UI Controller Refactor
1. Update the root UI controller to use the new Router
2. Implement global key binding handling
3. Add help system integration
4. Maintain backward compatibility during transition

#### 4.6. Page Lifecycle Management
1. Implement focus/blur handling in all pages
2. Add proper cleanup when pages are destroyed
3. Implement page state persistence where appropriate

### Phase 3: Advanced Features (Week 3)

#### 4.7. Context-Sensitive Help
1. Implement help system that shows page-specific key bindings
2. Add help overlay component
3. Integrate help system with global key bindings

#### 4.8. Focus Management
1. Implement focus management for components within pages
2. Add keyboard navigation between components
3. Implement visual focus indicators

#### 4.9. Error Handling and Recovery
1. Add error handling for navigation failures
2. Implement fallback navigation mechanisms
3. Add user-friendly error messages

### Phase 4: Testing and Polish (Week 4)

#### 4.10. Testing
1. Write unit tests for the router
2. Test navigation history functionality
3. Test global key binding system
4. Test focus management

#### 4.11. Documentation
1. Update architecture documentation
2. Document the new navigation patterns
3. Create examples for new page implementations

## 5. Detailed Implementation Steps

### 5.1. Enhanced Page Interface Implementation

Update `pkg/ui/core/interfaces.go`:

```go
// Page represents a UI page in the application
type Page interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetDimensions(width, height int)
    GetID() string
    
    // Navigation methods
    GetTitle() string
    GetHelp() []key.Binding
    HandleNavigation(msg tea.Msg) (Page, tea.Cmd)
    OnFocus() tea.Cmd
    OnBlur() tea.Cmd
}
```

Update existing pages to implement the new methods:

```go
// In pkg/ui/pages/main/main_page.go
func (m *Model) GetTitle() string {
    return "Kafui - Kafka TUI"
}

func (m *Model) GetHelp() []key.Binding {
    return m.keys.GetKeyBindings()
}

func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
    // Handle page-specific navigation
    return m, nil
}

func (m *Model) OnFocus() tea.Cmd {
    // Handle focus gain
    return nil
}

func (m *Model) OnBlur() tea.Cmd {
    // Handle focus loss
    return nil
}
```

### 5.2. Page Naming Refactor

1. Rename the `detail` package to `message_detail`:
   ```bash
   mv pkg/ui/pages/detail pkg/ui/pages/message_detail
   ```

2. Update import paths in all files that reference the detail package:
   ```go
   // Old import
   detailpage "github.com/Benny93/kafui/pkg/ui/pages/detail"
   
   // New import
   messageDetailPage "github.com/Benny93/kafui/pkg/ui/pages/message_detail"
   ```

3. Update page IDs to use descriptive names:
   - `main` for the main page
   - `topic` for the topic page
   - `message_detail` for the message detail page
   - `resource_detail` for the resource detail page

### 5.3. Router Implementation

Create `pkg/ui/router/router.go`:

```go
package router

import (
    "github.com/Benny93/kafui/pkg/api"
    "github.com/Benny93/kafui/pkg/ui/core"
    messagedetailpage "github.com/Benny93/kafui/pkg/ui/pages/message_detail"
    mainpage "github.com/Benny93/kafui/pkg/ui/pages/main"
    resourcedetailpage "github.com/Benny93/kafui/pkg/ui/pages/resource_detail"
    topicpage "github.com/Benny93/kafui/pkg/ui/pages/topic"
    tea "github.com/charmbracelet/bubbletea"
)

// Router manages page navigation and state
type Router struct {
    dataSource  api.KafkaDataSource
    pages       map[string]core.Page
    history     []string
    currentPage string
    width       int
    height      int
}

// NewRouter creates a new Router instance
func NewRouter(dataSource api.KafkaDataSource) *Router {
    return &Router{
        dataSource:  dataSource,
        pages:       make(map[string]core.Page),
        history:     make([]string, 0),
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
        page := r.createPage(pageID, data)
        if page != nil {
            r.pages[pageID] = page
        }
    }
    
    r.currentPage = pageID
    
    // Focus the new page
    if page, exists := r.pages[pageID]; exists {
        return page.OnFocus()
    }
    
    return nil
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

// SetDimensions updates the dimensions for all pages
func (r *Router) SetDimensions(width, height int) {
    r.width = width
    r.height = height
    
    // Update dimensions for all pages
    for _, page := range r.pages {
        page.SetDimensions(width, height)
    }
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
    case "message_detail":
        // Extract message data
        if messageData, ok := data.(api.Message); ok {
            return messagedetailpage.NewModel(r.dataSource, "unknown", messageData)
        }
        return messagedetailpage.NewModel(r.dataSource, "unknown", api.Message{})
    case "resource_detail":
        // Extract resource data
        if resourceItem, ok := data.(shared.ResourceItem); ok {
            return resourcedetailpage.NewModel(resourceItem, "unknown")
        }
        return resourcedetailpage.NewModel(nil, "unknown")
    default:
        return mainpage.NewModel(r.dataSource)
    }
}
```

### 5.3. Global Key Binding System

Update `pkg/ui/core/keys.go`:

```go
package core

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global key bindings
type GlobalKeyMap struct {
    Help     key.Binding
    Quit     key.Binding
    Back     key.Binding
    NextPage key.Binding
    PrevPage key.Binding
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
}
```

### 5.5. Root UI Controller Refactor

Update `pkg/ui/ui.go`:

```go
// Add router import
import (
    "github.com/Benny93/kafui/pkg/ui/router"
    // ... other imports
)

// Update Model struct
type Model struct {
    dataSource api.KafkaDataSource
    router     *router.Router
    showHelp   bool
    width      int
    height     int
}

// Update initialModel
func initialModel(dataSource api.KafkaDataSource) Model {
    r := router.NewRouter(dataSource)
    return Model{
        dataSource: dataSource,
        router:     r,
    }
}

// Update Init method
func (m Model) Init() tea.Cmd {
    return m.router.NavigateTo("main", nil)
}

// Update Update method
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    // Handle global key bindings first
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, core.DefaultGlobalKeys.Help):
            m.showHelp = !m.showHelp
            return m, nil
        case key.Matches(msg, core.DefaultGlobalKeys.Quit):
            return m, tea.Quit
        case key.Matches(msg, core.DefaultGlobalKeys.Back):
            if !m.showHelp {
                return m, m.router.Back()
            }
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.router.SetDimensions(msg.Width, msg.Height)
    }
    
    // Handle current page updates
    if !m.showHelp {
        currentPage := m.router.GetCurrentPage()
        if currentPage != nil {
            updatedPage, cmd := currentPage.Update(msg)
            if updatedPage != nil {
                // Handle page changes from within the page
                // This would require a method to update pages in the router
            }
            cmds = append(cmds, cmd)
        }
    }
    
    return m, tea.Batch(cmds...)
}

// Update View method
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
```

## 6. Benefits of the New Architecture

### 6.1. Improved Robustness
- **Centralized Navigation**: All navigation is handled by a single router component
- **History Management**: Built-in back navigation with proper history stack
- **Error Recovery**: Better error handling with fallback mechanisms
- **Consistent State**: Pages are properly initialized and cleaned up

### 6.2. Enhanced Maintainability
- **Decoupled Components**: Pages no longer need to know about other pages
- **Standardized Interface**: All pages implement the same interface
- **Clear Separation of Concerns**: Navigation logic is separate from page logic
- **Extensibility**: Easy to add new pages and navigation patterns
- **Clear Naming**: Page names clearly indicate their purpose, making the codebase easier to understand

### 6.3. Better User Experience
- **Context-Sensitive Help**: Each page can provide its own help information
- **Consistent Key Bindings**: Global key bindings work consistently across all pages
- **Focus Management**: Proper focus handling for better keyboard navigation
- **Visual Feedback**: Better feedback for navigation actions

## 7. Migration Strategy

### 7.1. Backward Compatibility
- Maintain existing page interfaces where possible
- Provide adapter patterns for legacy code
- Gradually migrate pages to the new interface

### 7.2. Transition Plan
1. Implement the new router replacing the existing navigation
2. Rename the `detail` package to `message_detail` and update all references
3. Update pages one by one to use the new interface
4. Update page IDs to use descriptive names
5. Remove legacy navigation code once all pages are migrated
6. Update documentation and examples

## 8. Testing Strategy

### 8.1. Unit Tests
- Test router navigation functionality
- Test history management
- Test page creation and initialization
- Test global key binding handling

### 8.2. Integration Tests
- Test end-to-end navigation workflows
- Test error recovery scenarios
- Test focus management
- Test help system integration

### 8.3. Manual Testing
- Test navigation on different terminal sizes
- Test key binding consistency
- Test error scenarios
- Test help system usability

## 9. Success Metrics

### 9.1. Technical Metrics
- All pages implement the enhanced Page interface
- Router handles all navigation correctly
- Global key bindings work consistently
- No memory leaks in page management

### 9.2. User Experience Metrics
- Navigation is intuitive and predictable
- Help system provides useful information
- Error messages are clear and actionable
- Keyboard navigation is smooth and responsive

### 9.3. Maintainability Metrics
- Code complexity is reduced
- New pages can be added easily
- Bug reports related to navigation decrease
- Development time for navigation features decreases

## 10. Risks and Mitigations

### 10.1. Technical Risks
- **Complexity Creep**: Mitigate by implementing changes incrementally
- **Performance Issues**: Monitor performance during each phase
- **Breaking Changes**: Maintain backward compatibility with deprecation warnings

### 10.2. User Experience Risks
- **Learning Curve**: Provide clear documentation and tutorials
- **Inconsistent Behavior**: Maintain strict design guidelines
- **Feature Overload**: Focus on core functionality first

## 11. Conclusion

This improvement plan will significantly enhance Kafui's navigation and key input handling systems, making them more robust and maintainable. By implementing a centralized router, standardized page interface, consistent key binding system, and clearer page naming conventions, we'll create a more professional and user-friendly TUI that follows modern Bubble Tea best practices.

The phased approach ensures we can deliver value incrementally while maintaining stability and backward compatibility. The page renaming from "detail" to "message_detail" will make the codebase more intuitive and easier to understand. The end result will be a navigation system that rivals the best terminal applications in terms of usability and reliability.


Notes: Running the app (make run-mock) will get you stuck since it is an interactive tui app. You can still run it using the timeout tool if you like