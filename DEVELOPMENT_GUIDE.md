# Development Guide

This guide explains how to develop with the improved Kafui architecture.

**Last Updated**: March 15, 2026

---

## Table of Contents

1. [Adding New Pages](#adding-new-pages)
2. [Adding New Components](#adding-new-components)
3. [Adding Key Bindings](#adding-key-bindings)
4. [Using the Common Context](#using-the-common-context)
5. [Using the Layout System](#using-the-layout-system)
6. [Using the Style System](#using-the-style-system)
7. [Error Handling](#error-handling)
8. [Testing](#testing)

---

## Adding New Pages

### Step 1: Create Page Structure

```go
// pkg/ui/pages/mypage/mypage.go
package mypage

import (
    "github.com/Benny93/kafui/pkg/ui/core"
    "github.com/Benny93/kafui/pkg/ui/keys"
    templateui "github.com/Benny93/kafui/pkg/ui/template/ui"
    "github.com/Benny93/kafui/pkg/ui/template/ui/providers"
    tea "github.com/charmbracelet/bubbletea"
)

// Model represents the page state
type Model struct {
    common *core.Common
    // ... page-specific fields
}

// NewModelWithCommon creates a new page model
func NewModelWithCommon(common *core.Common) *Model {
    return &Model{
        common: common,
        // Initialize fields
    }
}

// Init implements the Page interface
func (m *Model) Init() tea.Cmd {
    return nil
}

// Update implements the Page interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages
    return m, nil
}

// View implements the Page interface
func (m *Model) View() string {
    // Render view
    return "My Page"
}

// SetDimensions implements the Page interface
func (m *Model) SetDimensions(width, height int) {
    // Handle dimension changes
}

// GetID implements the Page interface
func (m *Model) GetID() string {
    return "mypage"
}

// GetTitle implements the Page interface
func (m *Model) GetTitle() string {
    return "My Page"
}

// GetHelp implements the Page interface
func (m *Model) GetHelp() []key.Binding {
    km := keys.DefaultKeyMap()
    return []key.Binding{
        km.Global.Help,
        km.Global.Back,
        km.Global.Quit,
    }
}

// HandleNavigation implements the Page interface
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
    // Handle navigation messages
    return m, nil
}

// OnFocus implements the Page interface
func (m *Model) OnFocus() tea.Cmd {
    // Handle focus gain
    return nil
}

// OnBlur implements the Page interface
func (m *Model) OnBlur() tea.Cmd {
    // Handle focus loss
    return nil
}
```

### Step 2: Register Page in Router

```go
// pkg/ui/router/router.go
func (r *Router) createPage(pageID string, data interface{}) core.Page {
    switch baseID {
    case "mypage":
        return mypage.NewModelWithCommon(r.com)
    // ... other cases
    }
}
```

---

## Adding New Components

### Step 1: Create Component with BaseComponent

```go
// pkg/ui/components/mycomponent.go
package components

import (
    "github.com/Benny93/kafui/pkg/ui/core"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// MyComponent represents a custom component
type MyComponent struct {
    core.BaseComponent // Embed for common functionality
    
    // Component-specific fields
    data string
}

// NewMyComponent creates a new component
func NewMyComponent() *MyComponent {
    return &MyComponent{
        BaseComponent: core.NewBaseComponent(0, 0),
    }
}

// Init implements Component interface
func (c *MyComponent) Init() tea.Cmd {
    // Can use c.BaseComponent.Init() for default
    return c.BaseComponent.Init()
}

// Update implements Component interface
func (c *MyComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle component-specific messages
    return c, nil
}

// View implements Component interface
func (c *MyComponent) View() string {
    return lipgloss.NewStyle().Render(c.data)
}
```

---

## Adding Key Bindings

### Step 1: Add to Centralized Key Map

```go
// pkg/ui/keys/keys.go
type MyPageKeyMap struct {
    Action1 key.Binding
    Action2 key.Binding
}

func DefaultMyPageKeyMap() MyPageKeyMap {
    return MyPageKeyMap{
        Action1: key.NewBinding(
            key.WithKeys("a"),
            key.WithHelp("a", "action 1"),
        ),
        Action2: key.NewBinding(
            key.WithKeys("b"),
            key.WithHelp("b", "action 2"),
        ),
    }
}
```

### Step 2: Add to Main KeyMap

```go
// pkg/ui/keys/keys.go
type KeyMap struct {
    // ... existing fields
    MyPage MyPageKeyMap
}

func DefaultKeyMap() KeyMap {
    return KeyMap{
        // ... existing fields
        MyPage: DefaultMyPageKeyMap(),
    }
}
```

### Step 3: Use in Page

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        km := keys.DefaultKeyMap()
        switch {
        case key.Matches(msg, km.MyPage.Action1):
            return m, m.handleAction1()
        case key.Matches(msg, km.MyPage.Action2):
            return m, m.handleAction2()
        }
    }
    return m, nil
}
```

---

## Using the Common Context

### Accessing Dependencies

```go
func (m *Model) loadData() tea.Cmd {
    // Access data source
    topics, err := m.common.DataSource.GetTopics()
    
    // Access styles
    style := m.common.Styles.Header
    
    // Access layout
    layout := m.common.Layout
    height := layout.GetAvailableHeight()
    
    // Access config
    theme := m.common.Config.Theme
    
    return nil
}
```

### Creating Common Context in Tests

```go
func TestMyPage(t *testing.T) {
    mockDS := &MockDataSource{}
    common := core.NewCommon(mockDS)
    
    page := mypage.NewModelWithCommon(common)
    
    // Test page
}
```

---

## Using the Layout System

### Getting Layout Dimensions

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // Layout is automatically updated via common.UpdateLayout()
        layout := m.common.Layout
        
        // Get available dimensions
        height := layout.GetAvailableHeight()
        width := layout.GetAvailableWidth()
        
        // Check layout mode
        if layout.CompactMode {
            // Use compact layout
        }
    }
    return m, nil
}
```

### Using Layout Calculator

```go
import "github.com/Benny93/kafui/pkg/ui/layout"

func calculateTableHeight(layout *layout.Layout) int {
    calc := layout.NewLayoutCalculator(layout.DefaultLayoutConfig())
    return calc.CalculateTableHeight(layout, 5) // Reserve 5 lines
}
```

---

## Using the Style System

### Using Semantic Colors

```go
import stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"

func renderError(err error) string {
    // Use semantic color
    style := lipgloss.NewStyle().Foreground(stylesPkg.Error)
    return style.Render(err.Error())
}
```

### Using Component Styles

```go
func (m *Model) View() string {
    styles := m.common.Styles
    
    // Use predefined component styles
    header := styles.HeaderStyle.Title.Render("My Header")
    footer := styles.FooterStyle.Base.Render("Footer")
    
    return lipgloss.JoinVertical(header, footer)
}
```

---

## Error Handling

### Using Status Messages

```go
func (m *Model) loadData() tea.Cmd {
    data, err := m.common.DataSource.GetData()
    if err != nil {
        // Show error with auto-dismiss (10 seconds)
        return core.NewErrorMsg(err.Error())
    }
    
    // Show success message
    return core.NewSuccessMsg("Data loaded successfully")
}
```

### Using Retry Logic

```go
import "github.com/Benny93/kafui/pkg/ui/core"

func (m *Model) loadDataWithRetry() tea.Cmd {
    config := core.DefaultRetryConfig()
    config.MaxRetries = 5
    config.InitialDelay = 2 * time.Second
    
    cmd := m.loadData()
    return core.RetryWithBackoff(cmd, config)
}
```

---

## Testing

### Unit Tests

```go
func TestMyPage_Update(t *testing.T) {
    mockDS := &MockDataSource{}
    common := core.NewCommon(mockDS)
    
    model := mypage.NewModelWithCommon(common)
    
    msg := tea.KeyMsg{Type: tea.KeyEnter}
    updatedModel, cmd := model.Update(msg)
    
    assert.IsType(t, model, updatedModel)
}
```

### Integration Tests

```go
func TestNavigation(t *testing.T) {
    mockDS := &MockDataSource{}
    common := core.NewCommon(mockDS)
    
    router := router.NewRouter(common)
    
    // Navigate to page
    cmd := router.NavigateTo("mypage", nil)
    
    // Execute navigation command
    msg := cmd()
    
    // Verify navigation occurred
    assert.IsType(t, core.PageChangeMsg{}, msg)
}
```

### Running Tests

```bash
# Run all tests
go test ./pkg/ui/... -v

# Run with race detector
go test ./pkg/ui/... -race

# Run with coverage
go test ./pkg/ui/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package
go test ./pkg/ui/pages/mypage/... -v
```

---

## Best Practices

### 1. Always Use Common Context

```go
// ✅ Good
func NewModel(common *core.Common) *Model {
    return &Model{common: common}
}

// ❌ Bad
func NewModel(dataSource api.KafkaDataSource) *Model {
    return &Model{dataSource: dataSource}
}
```

### 2. Use Centralized Keys

```go
// ✅ Good
km := keys.DefaultKeyMap()
if key.Matches(msg, km.Global.Quit) {
    return m, tea.Quit
}

// ❌ Bad
if msg.String() == "q" {
    return m, tea.Quit
}
```

### 3. Use Semantic Styles

```go
// ✅ Good
style := lipgloss.NewStyle().Foreground(stylesPkg.Error)

// ❌ Bad
style := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
```

### 4. Use Layout System

```go
// ✅ Good
height := m.common.Layout.GetAvailableHeight()

// ❌ Bad
height := m.height - 6
```

### 5. Handle Errors with Status Messages

```go
// ✅ Good
if err != nil {
    return core.NewErrorMsg(err.Error())
}

// ❌ Bad
m.error = err
```

---

## Troubleshooting

### Common Issues

**Issue**: "undefined: core.Common"  
**Solution**: Import `"github.com/Benny93/kafui/pkg/ui/core"`

**Issue**: "cannot use model (type *Model) as type tea.Model"  
**Solution**: Return `m, cmd` not just `m` from Update()

**Issue**: "key binding not defined"  
**Solution**: Add key binding to `pkg/ui/keys/keys.go`

**Issue**: "style not found"  
**Solution**: Check `pkg/ui/styles/styles.go` for available styles

---

## Related Documents

- [BUBBLE_TEA_IMPROVEMENT_PLAN.md](./BUBBLE_TEA_IMPROVEMENT_PLAN.md) - Improvement plan
- [ARCHITECTURE_DECISIONS.md](./ARCHITECTURE_DECISIONS.md) - Architecture decisions
- [UI_ARCHITECTURE.md](./UI_ARCHITECTURE.md) - Architecture documentation

---

**End of Guide**
