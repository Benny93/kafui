# Bubble Tea Layout Patterns for Kafui

This document provides specific implementation patterns for creating beautiful layouts in Kafui, applying the general principles from the Bubble Tea layout guide to the specific needs of a Kafka TUI.

## 1. Page Structure

### 1.1. Standard Page Interface

Implement a consistent page interface across all Kafui pages:

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

### 1.2. Base Page Implementation

Create a base page structure that can be extended:

```go
// pkg/ui/pages/base/base_page.go
package base

import (
    "github.com/Benny93/kafui/pkg/ui/core"
    "github.com/charmbracelet/bubbles/key"
    tea "github.com/charmbracelet/bubbletea"
)

type BasePage struct {
    ID          string
    Title       string
    Width       int
    Height      int
    IsFocused   bool
}

func (p *BasePage) GetID() string {
    return p.ID
}

func (p *BasePage) GetTitle() string {
    return p.Title
}

func (p *BasePage) SetDimensions(width, height int) {
    p.Width = width
    p.Height = height
}

func (p *BasePage) OnFocus() tea.Cmd {
    p.IsFocused = true
    return nil
}

func (p *BasePage) OnBlur() tea.Cmd {
    p.IsFocused = false
    return nil
}

func (p *BasePage) GetHelp() []key.Binding {
    return []key.Binding{}
}

func (p *BasePage) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
    return p, nil
}
```

## 2. Topic Page Layout

### 2.1. Component Structure

Organize the topic page into logical components:

```go
// pkg/ui/pages/topic/components.go
package topic

import (
    "github.com/Benny93/kafui/pkg/api"
    "github.com/charmbracelet/lipgloss"
)

type TopicPageComponents struct {
    Header   Header
    Sidebar  Sidebar
    Messages MessageList
    Status   StatusBar
}

type Header struct {
    topicName string
    style     lipgloss.Style
}

type Sidebar struct {
    message      *api.Message
    schemaInfo   *api.MessageSchemaInfo
    compactMode  bool
    style        lipgloss.Style
}

type MessageList struct {
    messages     []api.Message
    selectedIndex int
    style        lipgloss.Style
}

type StatusBar struct {
    connectionStatus string
    messageCount     int
    selectedItems    int
    style            lipgloss.Style
}
```

### 2.2. Styling Definitions

Define consistent styles for the topic page:

```go
// pkg/ui/theme/topic_theme.go
package theme

import "github.com/charmbracelet/lipgloss"

var (
    // Colors
    Primary   = lipgloss.AdaptiveColor{Light: "#2E8BC0", Dark: "#6CA6C1"}
    Secondary = lipgloss.AdaptiveColor{Light: "#145DA0", Dark: "#87CEEB"}
    Accent    = lipgloss.AdaptiveColor{Light: "#B1D4E0", Dark: "#B0E0E6"}
    Success   = lipgloss.AdaptiveColor{Light: "#28A745", Dark: "#90EE90"}
    Warning   = lipgloss.AdaptiveColor{Light: "#FFC107", Dark: "#FFD700"}
    Error     = lipgloss.AdaptiveColor{Light: "#DC3545", Dark: "#FF6347"}
    Info      = lipgloss.AdaptiveColor{Light: "#17A2B8", Dark: "#87CEEB"}

    // Component styles
    TopicHeaderStyle = lipgloss.NewStyle().
                Background(Primary).
                Foreground(lipgloss.Color("#FFFFFF")).
                Padding(0, 1).
                Bold(true)

    TopicSidebarStyle = lipgloss.NewStyle().
                Border(lipgloss.RoundedBorder()).
                BorderForeground(Primary).
                Padding(1)

    TopicMessageListStyle = lipgloss.NewStyle().
                Padding(0, 1)

    TopicSelectedMessageStyle = lipgloss.NewStyle().
                Background(lipgloss.AdaptiveColor{Light: "#E3F2FD", Dark: "#2C3E50"}).
                Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}).
                Padding(0, 1)

    TopicSchemaInfoStyle = lipgloss.NewStyle().
                Border(lipgloss.NormalBorder()).
                BorderForeground(Secondary).
                Padding(1).
                MarginTop(1)

    TopicStatusBarStyle = lipgloss.NewStyle().
                Background(Secondary).
                Foreground(lipgloss.Color("#FFFFFF")).
                Padding(0, 1)
)
```

### 2.3. Layout Implementation

Implement the topic page layout with responsive design:

```go
// pkg/ui/pages/topic/view.go
package topic

import (
    "fmt"
    "strings"
    
    "github.com/Benny93/kafui/pkg/api"
    "github.com/Benny93/kafui/pkg/ui/theme"
    "github.com/charmbracelet/lipgloss"
)

const (
    CompactModeWidthBreakpoint = 100
    SidebarWidth               = 35
    HeaderHeight               = 1
    StatusBarHeight            = 1
)

func (m *Model) View() string {
    if m.width == 0 || m.height == 0 {
        return "Loading..."
    }

    // Determine layout mode
    compact := m.width < CompactModeWidthBreakpoint

    if compact {
        return m.renderCompactView()
    }
    return m.renderFullView()
}

func (m *Model) renderFullView() string {
    // Calculate dimensions
    contentHeight := m.height - HeaderHeight - StatusBarHeight
    sidebarWidth := SidebarWidth
    messageListWidth := m.width - sidebarWidth

    // Render components
    header := m.renderHeader()
    sidebar := m.renderSidebar(sidebarWidth, contentHeight)
    messageList := m.renderMessageList(messageListWidth, contentHeight)
    statusBar := m.renderStatusBar()

    // Combine components
    content := lipgloss.JoinHorizontal(
        lipgloss.Top,
        messageList,
        sidebar,
    )

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        content,
        statusBar,
    )
}

func (m *Model) renderCompactView() string {
    // In compact mode, stack components vertically
    header := m.renderHeader()
    messageList := m.renderMessageList(m.width, m.height-HeaderHeight-StatusBarHeight-10)
    schemaInfo := m.renderCompactSchemaInfo()
    statusBar := m.renderStatusBar()

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        messageList,
        schemaInfo,
        statusBar,
    )
}

func (m *Model) renderHeader() string {
    title := fmt.Sprintf("Topic: %s", m.topicName)
    return theme.TopicHeaderStyle.Width(m.width).Render(title)
}

func (m *Model) renderSidebar(width, height int) string {
    if m.selectedMessage == nil {
        return theme.TopicSidebarStyle.
            Width(width).
            Height(height).
            Render("No message selected")
    }

    // Render message details
    var details []string
    
    // Basic message info
    details = append(details, theme.TopicSidebarStyle.Bold(true).Render("Message Details"))
    details = append(details, fmt.Sprintf("Offset: %d", m.selectedMessage.Offset))
    details = append(details, fmt.Sprintf("Partition: %d", m.selectedMessage.Partition))
    
    if m.selectedMessage.KeySchemaID != "" {
        details = append(details, fmt.Sprintf("Key Schema ID: %s", m.selectedMessage.KeySchemaID))
    }
    
    if m.selectedMessage.ValueSchemaID != "" {
        details = append(details, fmt.Sprintf("Value Schema ID: %s", m.selectedMessage.ValueSchemaID))
    }
    
    // Schema information
    if m.selectedMessageSchema != nil {
        details = append(details, "")
        details = append(details, theme.TopicSidebarStyle.Bold(true).Render("Schema Information"))
        
        if m.selectedMessageSchema.KeySchema != nil {
            details = append(details, fmt.Sprintf("Key Schema: %s", m.selectedMessageSchema.KeySchema.RecordName))
            details = append(details, fmt.Sprintf("Key ID: %d", m.selectedMessageSchema.KeySchema.ID))
        }
        
        if m.selectedMessageSchema.ValueSchema != nil {
            details = append(details, fmt.Sprintf("Value Schema: %s", m.selectedMessageSchema.ValueSchema.RecordName))
            details = append(details, fmt.Sprintf("Value ID: %d", m.selectedMessageSchema.ValueSchema.ID))
        }
    }
    
    // Headers
    if len(m.selectedMessage.Headers) > 0 {
        details = append(details, "")
        details = append(details, theme.TopicSidebarStyle.Bold(true).Render("Headers"))
        for _, header := range m.selectedMessage.Headers {
            details = append(details, fmt.Sprintf("%s: %s", header.Key, string(header.Value)))
        }
    }

    return theme.TopicSidebarStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, details...))
}

func (m *Model) renderCompactSchemaInfo() string {
    if m.selectedMessage == nil || m.selectedMessageSchema == nil {
        return ""
    }

    var info []string
    info = append(info, theme.TopicSchemaInfoStyle.Bold(true).Render("Schema Info"))
    
    if m.selectedMessageSchema.KeySchema != nil {
        info = append(info, fmt.Sprintf("Key: %s (%d)", 
            m.selectedMessageSchema.KeySchema.RecordName, 
            m.selectedMessageSchema.KeySchema.ID))
    }
    
    if m.selectedMessageSchema.ValueSchema != nil {
        info = append(info, fmt.Sprintf("Value: %s (%d)", 
            m.selectedMessageSchema.ValueSchema.RecordName, 
            m.selectedMessageSchema.ValueSchema.ID))
    }

    return theme.TopicSchemaInfoStyle.Render(lipgloss.JoinVertical(lipgloss.Left, info...))
}

func (m *Model) renderMessageList(width, height int) string {
    if len(m.messages) == 0 {
        return theme.TopicMessageListStyle.
            Width(width).
            Height(height).
            Render("No messages")
    }

    var messageViews []string
    for i, msg := range m.messages {
        style := theme.TopicMessageListStyle
        if i == m.cursor {
            style = theme.TopicSelectedMessageStyle
        }
        
        // Truncate message content for display
        key := truncateString(msg.Key, 20)
        value := truncateString(msg.Value, 50)
        
        messageView := fmt.Sprintf("[%d:%d] %s: %s", 
            msg.Partition, msg.Offset, key, value)
        messageViews = append(messageViews, style.Render(messageView))
    }

    return theme.TopicMessageListStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, messageViews...))
}

func (m *Model) renderStatusBar() string {
    statusText := fmt.Sprintf("Messages: %d | Selected: %d", 
        len(m.messages), m.cursor+1)
    
    if m.selectedMessage != nil && m.selectedMessageSchema != nil {
        schemaCount := 0
        if m.selectedMessageSchema.KeySchema != nil {
            schemaCount++
        }
        if m.selectedMessageSchema.ValueSchema != nil {
            schemaCount++
        }
        if schemaCount > 0 {
            statusText += fmt.Sprintf(" | Schemas: %d", schemaCount)
        }
    }

    return theme.TopicStatusBarStyle.
        Width(m.width).
        Render(statusText)
}

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    if maxLen > 3 {
        return s[:maxLen-3] + "..."
    }
    return s[:maxLen]
}
```

## 3. Detail Page Layout

### 3.1. Component Structure

Organize the detail page with clear sections:

```go
// pkg/ui/pages/detail/components.go
package detail

import (
    "github.com/Benny93/kafui/pkg/api"
    "github.com/charmbracelet/lipgloss"
)

type DetailPageComponents struct {
    Header      Header
    Sidebar     Sidebar
    Content     Content
    StatusBar   StatusBar
}

type Content struct {
    message      api.Message
    displayFormat DisplayFormat
    showHeaders  bool
    showMetadata bool
    style        lipgloss.Style
}

type DisplayFormat struct {
    KeyFormat   string // raw, json, hex
    ValueFormat string // raw, pretty, json, hex
    WrapLines   bool
    ShowBytes   bool
}
```

### 3.2. Layout Implementation

Implement the detail page with a clean, organized layout:

```go
// pkg/ui/pages/detail/view.go
package detail

import (
    "fmt"
    "strings"
    
    "github.com/Benny93/kafui/pkg/ui/theme"
    "github.com/charmbracelet/lipgloss"
)

const (
    DetailCompactWidthBreakpoint = 100
    DetailSidebarWidth           = 30
    DetailHeaderHeight           = 1
    DetailStatusBarHeight        = 1
)

func (m *Model) View() string {
    if m.width == 0 || m.height == 0 {
        return "Loading..."
    }

    compact := m.width < DetailCompactWidthBreakpoint

    if compact {
        return m.renderCompactView()
    }
    return m.renderFullView()
}

func (m *Model) renderFullView() string {
    // Calculate dimensions
    contentHeight := m.height - DetailHeaderHeight - DetailStatusBarHeight
    sidebarWidth := DetailSidebarWidth
    mainContentWidth := m.width - sidebarWidth

    // Render components
    header := m.renderHeader()
    sidebar := m.renderSidebar(sidebarWidth, contentHeight)
    content := m.renderContent(mainContentWidth, contentHeight)
    statusBar := m.renderStatusBar()

    // Combine components
    mainArea := lipgloss.JoinHorizontal(
        lipgloss.Top,
        content,
        sidebar,
    )

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        mainArea,
        statusBar,
    )
}

func (m *Model) renderCompactView() string {
    header := m.renderHeader()
    content := m.renderContent(m.width, m.height/2)
    sidebar := m.renderCompactSidebar()
    statusBar := m.renderStatusBar()

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        content,
        sidebar,
        statusBar,
    )
}

func (m *Model) renderHeader() string {
    title := fmt.Sprintf("Message Detail: %s", m.topicName)
    return theme.TopicHeaderStyle.Width(m.width).Render(title)
}

func (m *Model) renderSidebar(width, height int) string {
    var details []string
    
    // Basic metadata
    details = append(details, theme.TopicSidebarStyle.Bold(true).Render("Metadata"))
    details = append(details, fmt.Sprintf("Topic: %s", m.topicName))
    details = append(details, fmt.Sprintf("Offset: %d", m.message.Offset))
    details = append(details, fmt.Sprintf("Partition: %d", m.message.Partition))
    
    // Timestamp
    if !m.message.Timestamp.IsZero() {
        details = append(details, fmt.Sprintf("Timestamp: %s", m.message.Timestamp.Format("2006-01-02 15:04:05")))
    }
    
    // Schema information
    schemaInfo := m.GetSchemaInfo()
    if schemaInfo != nil {
        details = append(details, "")
        details = append(details, theme.TopicSidebarStyle.Bold(true).Render("Schema Info"))
        
        if schemaInfo.KeySchema != nil {
            details = append(details, fmt.Sprintf("Key Schema: %s", schemaInfo.KeySchema.RecordName))
            details = append(details, fmt.Sprintf("Key ID: %d", schemaInfo.KeySchema.ID))
        }
        
        if schemaInfo.ValueSchema != nil {
            details = append(details, fmt.Sprintf("Value Schema: %s", schemaInfo.ValueSchema.RecordName))
            details = append(details, fmt.Sprintf("Value ID: %d", schemaInfo.ValueSchema.ID))
        }
    }
    
    // Headers toggle
    headersStatus := "Hidden"
    if m.showHeaders {
        headersStatus = "Visible"
    }
    details = append(details, "")
    details = append(details, theme.TopicSidebarStyle.Bold(true).Render("Display Options"))
    details = append(details, fmt.Sprintf("Headers: %s", headersStatus))
    details = append(details, fmt.Sprintf("Metadata: %t", m.showMetadata))

    return theme.TopicSidebarStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, details...))
}

func (m *Model) renderCompactSidebar() string {
    var info []string
    
    info = append(info, fmt.Sprintf("Topic: %s | Offset: %d", m.topicName, m.message.Offset))
    
    schemaInfo := m.GetSchemaInfo()
    if schemaInfo != nil {
        var schemaInfoText []string
        if schemaInfo.KeySchema != nil {
            schemaInfoText = append(schemaInfoText, fmt.Sprintf("Key: %s(%d)", 
                schemaInfo.KeySchema.RecordName, schemaInfo.KeySchema.ID))
        }
        if schemaInfo.ValueSchema != nil {
            schemaInfoText = append(schemaInfoText, fmt.Sprintf("Value: %s(%d)", 
                schemaInfo.ValueSchema.RecordName, schemaInfo.ValueSchema.ID))
        }
        if len(schemaInfoText) > 0 {
            info = append(info, strings.Join(schemaInfoText, " | "))
        }
    }

    return theme.TopicSchemaInfoStyle.Render(lipgloss.JoinVertical(lipgloss.Left, info...))
}

func (m *Model) renderContent(width, height int) string {
    var contentSections []string
    
    // Key section
    keyTitle := "Key"
    if m.message.KeySchemaID != "" {
        keyTitle += fmt.Sprintf(" (Schema ID: %s)", m.message.KeySchemaID)
    }
    contentSections = append(contentSections, theme.TopicSidebarStyle.Bold(true).Render(keyTitle))
    contentSections = append(contentSections, m.GetFormattedKey())
    
    // Value section
    valueTitle := "Value"
    if m.message.ValueSchemaID != "" {
        valueTitle += fmt.Sprintf(" (Schema ID: %s)", m.message.ValueSchemaID)
    }
    contentSections = append(contentSections, "")
    contentSections = append(contentSections, theme.TopicSidebarStyle.Bold(true).Render(valueTitle))
    contentSections = append(contentSections, m.GetFormattedValue())
    
    // Headers section
    if m.showHeaders && len(m.message.Headers) > 0 {
        contentSections = append(contentSections, "")
        contentSections = append(contentSections, theme.TopicSidebarStyle.Bold(true).Render("Headers"))
        for _, header := range m.message.Headers {
            contentSections = append(contentSections, fmt.Sprintf("%s: %s", header.Key, string(header.Value)))
        }
    }

    return theme.TopicMessageListStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, contentSections...))
}

func (m *Model) renderStatusBar() string {
    formatInfo := fmt.Sprintf("Key: %s | Value: %s", 
        m.displayFormat.KeyFormat, m.displayFormat.ValueFormat)
    
    toggleInfo := ""
    if m.showHeaders {
        toggleInfo += " [H]Headers"
    }
    if m.showMetadata {
        toggleInfo += " [M]Metadata"
    }
    
    statusText := formatInfo + toggleInfo

    return theme.TopicStatusBarStyle.
        Width(m.width).
        Render(statusText)
}
```

## 4. Main Page Layout

### 4.1. Dashboard Style Layout

Create a dashboard-style main page:

```go
// pkg/ui/pages/main/view.go
package main

import (
    "fmt"
    "strings"
    
    "github.com/Benny93/kafui/pkg/ui/theme"
    "github.com/charmbracelet/lipgloss"
)

const (
    MainCompactWidthBreakpoint = 80
    MainSidebarWidth           = 25
)

func (m *Model) View() string {
    if m.width == 0 || m.height == 0 {
        return "Loading..."
    }

    compact := m.width < MainCompactWidthBreakpoint

    if compact {
        return m.renderCompactView()
    }
    return m.renderFullView()
}

func (m *Model) renderFullView() string {
    // Calculate dimensions
    contentHeight := m.height - 2 // Account for header and status bar
    
    // Render components
    header := m.renderHeader()
    sidebar := m.renderSidebar(MainSidebarWidth, contentHeight)
    content := m.renderContent(m.width-MainSidebarWidth, contentHeight)
    statusBar := m.renderStatusBar()

    // Combine components
    mainArea := lipgloss.JoinHorizontal(
        lipgloss.Top,
        sidebar,
        content,
    )

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        mainArea,
        statusBar,
    )
}

func (m *Model) renderCompactView() string {
    header := m.renderHeader()
    content := m.renderContent(m.width, m.height-4)
    statusBar := m.renderStatusBar()

    return lipgloss.JoinVertical(
        lipgloss.Left,
        header,
        content,
        statusBar,
    )
}

func (m *Model) renderHeader() string {
    title := "Kafui - Kafka TUI"
    subtitle := "Browse and interact with your Kafka clusters"
    
    headerContent := lipgloss.JoinVertical(
        lipgloss.Left,
        theme.TopicHeaderStyle.Bold(true).Render(title),
        theme.TopicHeaderStyle.Render(subtitle),
    )
    
    return theme.TopicHeaderStyle.Width(m.width).Render(headerContent)
}

func (m *Model) renderSidebar(width, height int) string {
    var menuItems []string
    menuItems = append(menuItems, theme.TopicSidebarStyle.Bold(true).Render("Navigation"))
    
    // Highlight current selection
    topicsStyle := theme.TopicMessageListStyle
    brokersStyle := theme.TopicMessageListStyle
    settingsStyle := theme.TopicMessageListStyle
    
    switch m.cursor {
    case 0:
        topicsStyle = theme.TopicSelectedMessageStyle
    case 1:
        brokersStyle = theme.TopicSelectedMessageStyle
    case 2:
        settingsStyle = theme.TopicSelectedMessageStyle
    }
    
    menuItems = append(menuItems, topicsStyle.Render("Topics"))
    menuItems = append(menuItems, brokersStyle.Render("Brokers"))
    menuItems = append(menuItems, settingsStyle.Render("Settings"))

    return theme.TopicSidebarStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, menuItems...))
}

func (m *Model) renderContent(width, height int) string {
    switch m.cursor {
    case 0:
        return m.renderTopicsView(width, height)
    case 1:
        return m.renderBrokersView(width, height)
    case 2:
        return m.renderSettingsView(width, height)
    default:
        return theme.TopicMessageListStyle.
            Width(width).
            Height(height).
            Render("Select an option from the menu")
    }
}

func (m *Model) renderTopicsView(width, height int) string {
    if len(m.topics) == 0 {
        return theme.TopicMessageListStyle.
            Width(width).
            Height(height).
            Render("No topics found")
    }

    var topicViews []string
    topicViews = append(topicViews, theme.TopicSidebarStyle.Bold(true).Render("Available Topics"))
    
    for _, topic := range m.topics {
        topicViews = append(topicViews, fmt.Sprintf("• %s (%d partitions)", 
            topic.Name, len(topic.Partitions)))
    }

    return theme.TopicMessageListStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, topicViews...))
}

func (m *Model) renderBrokersView(width, height int) string {
    if len(m.brokers) == 0 {
        return theme.TopicMessageListStyle.
            Width(width).
            Height(height).
            Render("No brokers found")
    }

    var brokerViews []string
    brokerViews = append(brokerViews, theme.TopicSidebarStyle.Bold(true).Render("Connected Brokers"))
    
    for _, broker := range m.brokers {
        status := "Connected"
        if !broker.Connected {
            status = "Disconnected"
        }
        brokerViews = append(brokerViews, fmt.Sprintf("• %s:%d (%s)", 
            broker.Host, broker.Port, status))
    }

    return theme.TopicMessageListStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, brokerViews...))
}

func (m *Model) renderSettingsView(width, height int) string {
    settings := []string{
        theme.TopicSidebarStyle.Bold(true).Render("Settings"),
        "• Cluster: " + m.currentCluster,
        "• Refresh Interval: " + fmt.Sprintf("%ds", m.refreshInterval),
        "• Max Messages: " + fmt.Sprintf("%d", m.maxMessages),
    }

    return theme.TopicMessageListStyle.
        Width(width).
        Height(height).
        Render(lipgloss.JoinVertical(lipgloss.Left, settings...))
}

func (m *Model) renderStatusBar() string {
    statusText := fmt.Sprintf("Clusters: %d | Topics: %d | Brokers: %d", 
        len(m.clusters), len(m.topics), len(m.brokers))

    return theme.TopicStatusBarStyle.
        Width(m.width).
        Render(statusText)
}
```

## 5. Theme System Integration

### 5.1. Centralized Theme Management

Create a centralized theme system:

```go
// pkg/ui/theme/theme.go
package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme and styling for the application
type Theme struct {
    Primary   lipgloss.AdaptiveColor
    Secondary lipgloss.AdaptiveColor
    Accent    lipgloss.AdaptiveColor
    Background lipgloss.AdaptiveColor
    Foreground lipgloss.AdaptiveColor
    Success   lipgloss.AdaptiveColor
    Warning   lipgloss.AdaptiveColor
    Error     lipgloss.AdaptiveColor
    Info      lipgloss.AdaptiveColor
}

// Default themes
var (
    LightTheme = Theme{
        Primary:    lipgloss.AdaptiveColor{Light: "#2E8BC0", Dark: "#2E8BC0"},
        Secondary:  lipgloss.AdaptiveColor{Light: "#145DA0", Dark: "#145DA0"},
        Accent:     lipgloss.AdaptiveColor{Light: "#B1D4E0", Dark: "#B1D4E0"},
        Background: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"},
        Foreground: lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"},
        Success:    lipgloss.AdaptiveColor{Light: "#28A745", Dark: "#28A745"},
        Warning:    lipgloss.AdaptiveColor{Light: "#FFC107", Dark: "#FFC107"},
        Error:      lipgloss.AdaptiveColor{Light: "#DC3545", Dark: "#DC3545"},
        Info:       lipgloss.AdaptiveColor{Light: "#17A2B8", Dark: "#17A2B8"},
    }
    
    DarkTheme = Theme{
        Primary:    lipgloss.AdaptiveColor{Light: "#6CA6C1", Dark: "#6CA6C1"},
        Secondary:  lipgloss.AdaptiveColor{Light: "#87CEEB", Dark: "#87CEEB"},
        Accent:     lipgloss.AdaptiveColor{Light: "#B0E0E6", Dark: "#B0E0E6"},
        Background: lipgloss.AdaptiveColor{Light: "#1E1E1E", Dark: "#1E1E1E"},
        Foreground: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"},
        Success:    lipgloss.AdaptiveColor{Light: "#90EE90", Dark: "#90EE90"},
        Warning:    lipgloss.AdaptiveColor{Light: "#FFD700", Dark: "#FFD700"},
        Error:      lipgloss.AdaptiveColor{Light: "#FF6347", Dark: "#FF6347"},
        Info:       lipgloss.AdaptiveColor{Light: "#87CEEB", Dark: "#87CEEB"},
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
            Background(theme.Primary).
            Foreground(theme.Background).
            Padding(0, 1).
            Bold(true),
            
        Footer: lipgloss.NewStyle().
            Background(theme.Secondary).
            Foreground(theme.Background).
            Padding(0, 1),
            
        Sidebar: lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(theme.Primary).
            Padding(1),
            
        MainPanel: lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(theme.Secondary).
            Padding(1),
            
        Title: lipgloss.NewStyle().
            Foreground(theme.Primary).
            Bold(true),
            
        Subtitle: lipgloss.NewStyle().
            Foreground(theme.Secondary).
            Bold(true),
            
        InfoText: lipgloss.NewStyle().
            Foreground(theme.Info),
            
        SuccessText: lipgloss.NewStyle().
            Foreground(theme.Success),
            
        WarningText: lipgloss.NewStyle().
            Foreground(theme.Warning),
            
        ErrorText: lipgloss.NewStyle().
            Foreground(theme.Error),
    }
}
```

This implementation provides a comprehensive layout system for Kafui that follows modern TUI design principles, with responsive layouts, consistent styling, and clear visual hierarchy.