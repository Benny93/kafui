# Bubble Tea Migration Plan

## Overview
This document outlines the step-by-step process to migrate Kafui from tview to Bubble Tea. Each step includes specific commands, code changes, verification steps, and fallback procedures.

## Migration Steps

### Step 1: Add Bubble Tea Dependencies

```powershell
# Add required dependencies
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles

# Tidy modules
go mod tidy
```

**Verification:**
```powershell
go mod verify
go build ./...
```

**Expected Output:**
- No errors in build
- New dependencies in go.mod


### Step 2: Create Base Bubble Tea Models

Create new base models that will replace tview components. Create file `pkg/kafui/model/base.go`:

```go
package model

import (
    tea "github.com/charmbracelet/bubbletea"
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
```

**Verification:**
```powershell
go build ./pkg/kafui/model
```

### Step 3: Create Core UI Components

#### 3.1 Table Component
Create file `pkg/kafui/model/table.go`:

```go
package model

import (
    "strings"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type Table struct {
    BaseModel
    Headers     []string
    Rows        [][]string
    Selected    int
    Scrollable  bool
    scrollOffset int
}

func NewTable(headers []string) *Table {
    return &Table{
        Headers:  headers,
        Selected: 0,
        BaseModel: BaseModel{
            styles: NewStyles(),
        },
    }
}

func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up":
            if t.Selected > 0 {
                t.Selected--
            }
        case "down":
            if t.Selected < len(t.Rows)-1 {
                t.Selected++
            }
        }
    }
    return t, nil
}

func (t *Table) View() string {
    var b strings.Builder
    
    // Build headers
    headerRow := make([]string, len(t.Headers))
    for i, h := range t.Headers {
        headerRow[i] = t.styles.Title.Render(h)
    }
    b.WriteString(strings.Join(headerRow, " | ") + "\n")
    b.WriteString(strings.Repeat("-", t.width) + "\n")
    
    // Build rows
    visibleRows := t.getVisibleRows()
    for i, row := range visibleRows {
        style := t.styles.Normal
        if i+t.scrollOffset == t.Selected {
            style = t.styles.Selected
        }
        renderedRow := make([]string, len(row))
        for j, cell := range row {
            renderedRow[j] = style.Render(cell)
        }
        b.WriteString(strings.Join(renderedRow, " | ") + "\n")
    }
    
    return t.styles.Border.Render(b.String())
}

func (t *Table) getVisibleRows() [][]string {
    if !t.Scrollable {
        return t.Rows
    }
    // Calculate visible rows based on height
    maxVisible := t.height - 3 // Account for headers and border
    start := t.scrollOffset
    end := start + maxVisible
    if end > len(t.Rows) {
        end = len(t.Rows)
    }
    return t.Rows[start:end]
}
```

#### 3.2 Search Bar Component
Create file `pkg/kafui/model/searchbar.go`:

```go
package model

import (
    "strings"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textinput"
)

type SearchBar struct {
    BaseModel
    input textinput.Model
    focused bool
}

func NewSearchBar() *SearchBar {
    ti := textinput.New()
    ti.Placeholder = "Search..."
    return &SearchBar{
        input: ti,
        BaseModel: BaseModel{
            styles: NewStyles(),
        },
    }
}

func (s *SearchBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "esc":
            s.focused = false
            s.input.Blur()
        case "enter":
            if s.focused {
                // Handle search submission
                return s, nil
            }
        }
    }
    if s.focused {
        s.input, cmd = s.input.Update(msg)
    }
    return s, cmd
}

func (s *SearchBar) View() string {
    return s.styles.Border.Render(s.input.View())
}

func (s *SearchBar) Focus() tea.Cmd {
    s.focused = true
    return s.input.Focus()
}
```

**Verification:**
```powershell
go build ./pkg/kafui/model
go test ./pkg/kafui/model
```

### Step 4: Main Application Model

Create file `pkg/kafui/model/app.go`:

```go
package model

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/kafui/pkg/datasource/kafds"
)

type AppModel struct {
    BaseModel
    dataSource  kafds.KafkaDataSource
    table       *Table
    searchBar   *SearchBar
    activeView  string
    error      string
}

func NewAppModel(ds kafds.KafkaDataSource) *AppModel {
    return &AppModel{
        dataSource:  ds,
        table:      NewTable([]string{"Topic", "Partitions", "Replicas"}),
        searchBar:  NewSearchBar(),
        activeView: "main",
        BaseModel: BaseModel{
            styles: NewStyles(),
        },
    }
}

func (m *AppModel) Init() tea.Cmd {
    return tea.Batch(
        m.fetchTopics,
    )
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "/":
            cmds = append(cmds, m.searchBar.Focus())
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }

    var cmd tea.Cmd
    switch m.activeView {
    case "main":
        m.table, cmd = m.table.Update(msg)
        cmds = append(cmds, cmd)
    case "search":
        m.searchBar, cmd = m.searchBar.Update(msg)
        cmds = append(cmds, cmd)
    }

    return m, tea.Batch(cmds...)
}

func (m *AppModel) View() string {
    var b strings.Builder

    if m.error != "" {
        return m.styles.Error.Render(m.error)
    }

    b.WriteString(m.searchBar.View() + "\n\n")
    b.WriteString(m.table.View())

    return b.String()
}

func (m *AppModel) fetchTopics() tea.Msg {
    topics, err := m.dataSource.ListTopics()
    if err != nil {
        return errorMsg{err}
    }
    return topicsMsg{topics}
}

type errorMsg struct{ error }
type topicsMsg struct{ topics []string }
```

### Step 5: Replace Main UI Integration

Update `pkg/kafui/ui.go`:

```go
package kafui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/kafui/pkg/kafui/model"
)

type UI struct {
    program *tea.Program
    model   *model.AppModel
}

func NewUI(ds kafds.KafkaDataSource) *UI {
    m := model.NewAppModel(ds)
    p := tea.NewProgram(m)
    
    return &UI{
        program: p,
        model:   m,
    }
}

func (ui *UI) Start() error {
    _, err := ui.program.Run()
    return err
}
```

### Step 6: Update Tests

Create file `pkg/kafui/model/app_test.go`:

```go
package model

import (
    "testing"
    tea "github.com/charmbracelet/bubbletea"
)

func TestAppModel(t *testing.T) {
    m := NewAppModel(nil)
    
    // Test initialization
    if m.activeView != "main" {
        t.Errorf("Expected activeView to be 'main', got %s", m.activeView)
    }
    
    // Test quit
    model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
    if cmd == nil {
        t.Error("Expected quit command")
    }
    
    // Test search focus
    model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
    if cmd == nil {
        t.Error("Expected search focus command")
    }
}
```

**Verification:**
```powershell
go test ./pkg/kafui/model
```

### Step 7: Remove tview Dependencies

```powershell
# Find all tview imports
go list -f '{{.ImportPath}} {{.Imports}}' ./... | findstr "tview"

# Remove tview
go get -u github.com/rivo/tview@none
go mod tidy
```

**Verification:**
```powershell
# Check for remaining tview imports
go list -f '{{.ImportPath}} {{.Imports}}' ./... | findstr "tview"
# Should return nothing

go build ./...
```

### Step 8: Final Validation

Create file `scripts/validate_migration.sh`:

```bash
#!/bin/bash
set -e

# Build check
echo "Running build check..."
go build ./...

# Test check
echo "Running tests..."
go test ./...

# Import check
echo "Checking for tview imports..."
if go list -f '{{.ImportPath}} {{.Imports}}' ./... | grep -q "tview"; then
    echo "Error: Found remaining tview imports"
    exit 1
fi

# Run application check
echo "Testing application startup..."
timeout 5s ./kafui --version

echo "Migration validation complete!"
```

**Run Validation:**
```powershell
# Make script executable
chmod +x scripts/validate_migration.sh

# Run validation
./scripts/validate_migration.sh
```

### Fallback Plan

If any step fails:

1. Save any working changes:
```powershell
git add .
git commit -m "WIP: Bubble Tea migration progress"
```

2. Restore from backup:
```powershell
git checkout backup/pre-bubbletea-migration
git checkout -b feature/bubbletea-migration-retry
```

3. Cherry-pick working changes if needed:
```powershell
git cherry-pick <commit-hash>
```

## Post-Migration Tasks

1. Update documentation:
```powershell
# Update README.md with new UI library information
git checkout feature/bubbletea-migration
git add README.md
git commit -m "docs: update UI library information"
```

2. Create pull request:
```powershell
git push origin feature/bubbletea-migration
# Create PR through GitHub interface
```

## Completion Checklist

- [ ] All tview dependencies removed
- [ ] All UI components migrated to Bubble Tea
- [ ] All tests passing
- [ ] Application builds successfully
- [ ] Manual testing completed
- [ ] Documentation updated
- [ ] PR created
