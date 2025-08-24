# Creating Good-Looking Bubble Tea Layouts: A Guide Inspired by Crush

This guide provides practical advice and patterns for creating visually appealing terminal user interfaces using Bubble Tea and Lip Gloss, inspired by the design principles seen in the Crush TUI.

## 1. Understanding Layout Fundamentals

### 1.1. Component-Based Architecture

Good Bubble Tea layouts are built using a component-based architecture where each part of the UI is a self-contained component:

```go
// Define components as separate structs
type Header struct {
    title string
    style lipgloss.Style
}

type Sidebar struct {
    items []string
    style lipgloss.Style
}

type ChatView struct {
    messages []Message
    style    lipgloss.Style
}

type Editor struct {
    content string
    style   lipgloss.Style
}
```

### 1.2. Layout Constants

Define layout constants at the package level for consistency:

```go
const (
    CompactModeWidthBreakpoint  = 120
    CompactModeHeightBreakpoint = 30
    EditorHeight                = 5
    SideBarWidth                = 31
    HeaderHeight                = 1
    
    // Layout constants for borders and padding
    BorderWidth      = 1
    LeftRightBorders = 2
    TopBottomBorders = 2
)
```

## 2. Using Lip Gloss for Styling

### 2.1. Style Definitions

Define styles at the package level for reusability:

```go
var (
    normal    = lipgloss.Color("#EEEEEE")
    subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
    highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
    special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

    // Component styles
    headerStyle = lipgloss.NewStyle().
        Background(highlight).
        Foreground(lipgloss.Color("#FFF7DB")).
        Padding(0, 1).
        Bold(true)

    sidebarStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(highlight).
        Padding(1).
        Width(SideBarWidth)

    chatStyle = lipgloss.NewStyle().
        Padding(1, 2)

    editorStyle = lipgloss.NewStyle().
        Border(lipgloss.HiddenBorder()).
        Background(lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#252525"}).
        Padding(1)
)
```

### 2.2. Adaptive Colors

Use adaptive colors to support both light and dark terminal themes:

```go
// Use AdaptiveColor for text that should work on both light and dark backgrounds
statusText := lipgloss.NewStyle().
    Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
    Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})
```

## 3. Layout Composition Patterns

### 3.1. Horizontal Layouts with JoinHorizontal

Use `lipgloss.JoinHorizontal` to create side-by-side layouts:

```go
func (m *Model) View() string {
    // Create sidebar content
    sidebarView := sidebarStyle.Render(m.sidebar.View())
    
    // Create main content (chat + editor)
    chatView := chatStyle.Render(m.chat.View())
    editorView := editorStyle.Render(m.editor.View())
    mainContent := lipgloss.JoinVertical(lipgloss.Left, chatView, editorView)
    
    // Combine sidebar and main content horizontally
    return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainContent)
}
```

### 3.2. Vertical Layouts with JoinVertical

Use `lipgloss.JoinVertical` to stack components:

```go
func (m *Model) View() string {
    headerView := headerStyle.Render(m.header.View())
    contentView := contentStyle.Render(m.content.View())
    statusBarView := statusBarStyle.Render(m.statusBar.View())
    
    return lipgloss.JoinVertical(lipgloss.Left, headerView, contentView, statusBarView)
}
```

### 3.3. Flexible Layout with Place Functions

Use `lipgloss.Place` for more precise positioning:

```go
dialog := lipgloss.Place(width, 9,
    lipgloss.Center, lipgloss.Center,
    dialogBoxStyle.Render(ui),
    lipgloss.WithWhitespaceChars("猫咪"),
    lipgloss.WithWhitespaceForeground(subtle),
)
```

## 4. Responsive Design Patterns

### 4.1. Compact Mode Handling

Implement different layouts based on terminal size:

```go
type Model struct {
    width, height int
    compact       bool
}

func (m *Model) SetSize(width, height int) tea.Cmd {
    m.width, m.height = width, height
    
    // Switch to compact mode based on breakpoints
    m.compact = width < CompactModeWidthBreakpoint || 
                height < CompactModeHeightBreakpoint
    
    // Update component sizes
    m.sidebar.SetCompactMode(m.compact)
    
    return nil
}

func (m *Model) View() string {
    if m.compact {
        // Stack components vertically in compact mode
        return lipgloss.JoinVertical(lipgloss.Left,
            m.header.View(),
            m.chat.View(),
            m.editor.View(),
        )
    }
    
    // Use horizontal layout in regular mode
    return lipgloss.JoinHorizontal(lipgloss.Top,
        m.sidebar.View(),
        lipgloss.JoinVertical(lipgloss.Left,
            m.chat.View(),
            m.editor.View(),
        ),
    )
}
```

### 4.2. Dynamic Component Sizing

Calculate component sizes based on available space:

```go
func (m *Model) updateComponentSizes() {
    if m.compact {
        m.chat.SetSize(m.width, m.height-HeaderHeight-EditorHeight)
        m.editor.SetSize(m.width, EditorHeight)
    } else {
        sidebarWidth := SideBarWidth
        contentWidth := m.width - sidebarWidth
        m.sidebar.SetSize(sidebarWidth, m.height)
        m.chat.SetSize(contentWidth, m.height-EditorHeight)
        m.editor.SetSize(contentWidth, EditorHeight)
    }
}
```

## 5. Initial Sizing and Layout Calculation

### 5.1. Handling the First Window Size Message

When a Bubble Tea application starts, it's crucial to properly handle the initial window size to ensure your UI renders correctly from the first frame. The key is to process the `tea.WindowSizeMsg` that Bubble Tea sends automatically when the application starts:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // This message is sent automatically when the application starts
        // and whenever the terminal is resized
        m.width = msg.Width
        m.height = msg.Height
        
        // Update component sizes based on new dimensions
        m.updateComponentSizes()
        
        // Return nil as no additional command is needed
        return m, nil
    // ... other message handlers
    }
    // ... rest of update logic
}
```

### 5.2. Ensuring Proper Initial Render

To ensure your application renders correctly on the first frame, you should:

1. Initialize your model with reasonable default dimensions
2. Handle the case where dimensions are not yet known
3. Update dimensions as soon as the first `tea.WindowSizeMsg` arrives

```go
type Model struct {
    width, height int
    // ... other fields
}

func NewModel() *Model {
    return &Model{
        // Initialize with default dimensions or zero
        // Zero values will be updated by the first WindowSizeMsg
        width:  0,
        height: 0,
        // ... initialize other fields
    }
}

func (m *Model) View() string {
    // Handle case where dimensions are not yet known
    if m.width == 0 || m.height == 0 {
        // Return a minimal loading view or empty string
        return "Loading..."
    }
    
    // Render your full UI with proper sizing
    return m.renderFullView()
}

func (m *Model) renderFullView() string {
    // Now you can safely use m.width and m.height for layout calculations
    headerHeight := 3
    footerHeight := 2
    contentHeight := m.height - headerHeight - footerHeight
    
    header := m.renderHeader(m.width, headerHeight)
    content := m.renderContent(m.width, contentHeight)
    footer := m.renderFooter(m.width, footerHeight)
    
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

### 5.3. Component Initialization with Proper Sizing

When initializing components, ensure they can handle being rendered before dimensions are known:

```go
type MessageList struct {
    messages []string
    width    int
    height   int
    style    lipgloss.Style
}

func NewMessageList() *MessageList {
    return &MessageList{
        messages: []string{},
        width:    0, // Will be set by parent component
        height:   0, // Will be set by parent component
        style:    lipgloss.NewStyle().Padding(1),
    }
}

func (ml *MessageList) SetSize(width, height int) {
    ml.width = width
    ml.height = height
    // Update style with new dimensions if needed
    ml.style = ml.style.Width(width).Height(height)
}

func (ml *MessageList) View() string {
    if ml.width == 0 || ml.height == 0 {
        // Return minimal view when dimensions unknown
        return ""
    }
    
    // Render with proper sizing
    return ml.style.Render(ml.renderMessages())
}
```

### 5.4. Dealing with Terminal Resize Events

Bubble Tea automatically sends `tea.WindowSizeMsg` whenever the terminal is resized. Your application should gracefully handle these events:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        
        // Recalculate layout for all components
        m.updateLayout()
        
        // In some cases, you might want to trigger a refresh of data
        // that depends on the new dimensions
        return m, m.refreshContentIfNeeded()
        
    // ... other message handlers
    }
    // ... rest of update logic
}

func (m *Model) updateLayout() {
    // Update all components with new dimensions
    m.header.SetSize(m.width, HeaderHeight)
    
    if m.compact {
        contentHeight := m.height - HeaderHeight - FooterHeight
        m.content.SetSize(m.width, contentHeight)
    } else {
        sidebarWidth := SideBarWidth
        contentWidth := m.width - sidebarWidth
        contentHeight := m.height - HeaderHeight - FooterHeight
        
        m.sidebar.SetSize(sidebarWidth, m.height)
        m.content.SetSize(contentWidth, contentHeight)
    }
    
    m.footer.SetSize(m.width, FooterHeight)
}
```

### 5.5. Best Practices for Initial Sizing

1. **Always check for zero dimensions**: Before using width/height in calculations, check if they're greater than zero.

2. **Provide fallback views**: Show a loading message or minimal UI when dimensions are unknown.

3. **Initialize with reasonable defaults**: While dimensions will be updated by `tea.WindowSizeMsg`, having reasonable defaults can prevent errors.

4. **Update all components**: When dimensions change, ensure all components in your layout are notified.

5. **Consider content constraints**: Some components may have minimum size requirements that should be respected.

```go
func (m *Model) updateLayout() {
    // Ensure minimum dimensions
    minWidth := 40
    minHeight := 10
    
    width := m.width
    if width < minWidth {
        width = minWidth
    }
    
    height := m.height
    if height < minHeight {
        height = minHeight
    }
    
    // Apply dimensions to components
    m.applyDimensions(width, height)
}
```

By following these patterns, your Bubble Tea application will render correctly from the first frame and gracefully handle terminal resizing events.

## 6. Visual Hierarchy and Typography

### 6.1. Establishing Visual Hierarchy

Use styling to create clear visual hierarchy:

```go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#874BFD")).
        MarginBottom(1)

    subtitleStyle = lipgloss.NewStyle().
        Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#626262"})

    bodyStyle = lipgloss.NewStyle().
        Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#DDDDDD"})
)
```

### 6.2. Consistent Spacing

Use consistent padding and margins throughout your application:

```go
// Define standard spacing constants
const (
    SmallPadding  = 1
    MediumPadding = 2
    LargePadding  = 3
    
    SmallMargin  = 1
    MediumMargin = 2
    LargeMargin  = 3
)

// Apply consistent spacing
componentStyle := lipgloss.NewStyle().
    Padding(MediumPadding).
    Margin(MediumMargin, SmallMargin)
```

## 7. Advanced Layout Techniques

### 7.1. Tab-Based Navigation

Create tabbed interfaces similar to Crush:

```go
var (
    activeTabBorder = lipgloss.Border{
        Top:         "─",
        Bottom:      " ",
        Left:        "│",
        Right:       "│",
        TopLeft:     "╭",
        TopRight:    "╮",
        BottomLeft:  "┘",
        BottomRight: "└",
    }

    tabBorder = lipgloss.Border{
        Top:         "─",
        Bottom:      "─",
        Left:        "│",
        Right:       "│",
        TopLeft:     "╭",
        TopRight:    "╮",
        BottomLeft:  "┴",
        BottomRight: "┴",
    }

    tabStyle = lipgloss.NewStyle().
        Border(tabBorder, true).
        BorderForeground(highlight).
        Padding(0, 1)

    activeTabStyle = tabStyle.Border(activeTabBorder, true)
)

func renderTabs(activeTab string, tabs []string) string {
    var tabViews []string
    for _, tab := range tabs {
        if tab == activeTab {
            tabViews = append(tabViews, activeTabStyle.Render(tab))
        } else {
            tabViews = append(tabViews, tabStyle.Render(tab))
        }
    }
    return lipgloss.JoinHorizontal(lipgloss.Bottom, tabViews...)
}
```

### 7.2. Status Bars

Create informative status bars:

```go
func renderStatusBar(width int, status, encoding, fishCake string) string {
    statusKey := statusStyle.Render("STATUS")
    encodingView := encodingStyle.Render(encoding)
    fishCakeView := fishCakeStyle.Render(fishCake)
    
    statusVal := statusText.
        Width(width - lipgloss.Width(statusKey) - lipgloss.Width(encodingView) - lipgloss.Width(fishCakeView)).
        Render(status)

    bar := lipgloss.JoinHorizontal(lipgloss.Top,
        statusKey,
        statusVal,
        encodingView,
        fishCakeView,
    )

    return statusBarStyle.Width(width).Render(bar)
}
```

## 8. Best Practices

### 8.1. Performance Considerations

- Cache rendered components when possible
- Only re-render components that have changed
- Use `lipgloss.Width()` and `lipgloss.Height()` to measure rendered content

```go
func (m *Model) View() string {
    // Only re-render if content has changed
    if m.chatView == "" {
        m.chatView = chatStyle.Render(m.chat.View())
    }
    
    return m.chatView
}
```

### 8.2. Accessibility

- Ensure sufficient color contrast
- Provide keyboard navigation
- Use semantic styling (headers, content, etc.)

### 8.3. Consistency

- Maintain consistent styling across components
- Use the same spacing and padding patterns
- Keep interaction patterns consistent

## 9. Example Implementation

Here's a complete example showing these patterns in action:

```go
package main

import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    width, height int
    compact       bool
    messages      []string
    input         textinput.Model
}

func (m model) Init() tea.Cmd {
    return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        m.compact = msg.Width < 80
        m.input.Width = msg.Width - 4
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            m.messages = append(m.messages, m.input.Value())
            m.input.SetValue("")
        }
    }
    
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m model) View() string {
    if m.width == 0 || m.height == 0 {
        return "Loading..."
    }
    
    if m.compact {
        return m.renderCompactView()
    }
    return m.renderFullView()
}

func (m model) renderFullView() string {
    // Create sidebar
    sidebar := sidebarStyle.Render(
        lipgloss.JoinVertical(lipgloss.Left,
            "Channels",
            "#general",
            "#random",
            "#dev",
        ),
    )
    
    // Create chat area
    var messageViews []string
    for _, msg := range m.messages {
        messageViews = append(messageViews, msg)
    }
    chat := chatStyle.Render(
        lipgloss.JoinVertical(lipgloss.Left, messageViews...),
    )
    
    // Create input area
    input := editorStyle.Render(m.input.View())
    
    // Combine main content vertically
    mainContent := lipgloss.JoinVertical(lipgloss.Left, chat, input)
    
    // Combine sidebar and main content horizontally
    return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent)
}

func (m model) renderCompactView() string {
    // Stack everything vertically in compact mode
    var messageViews []string
    for _, msg := range m.messages {
        messageViews = append(messageViews, msg)
    }
    
    return lipgloss.JoinVertical(lipgloss.Left,
        "Channels: #general",
        lipgloss.JoinVertical(lipgloss.Left, messageViews...),
        m.input.View(),
    )
}
```

This guide provides a foundation for creating beautiful, responsive terminal user interfaces with Bubble Tea and Lip Gloss, following patterns established by successful applications like Crush.