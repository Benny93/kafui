# Page Architecture Standard

**Date:** 2026-02-24  
**Status:** Approved

---

## Standard Page Architecture Pattern

All pages in Kafui should follow the **Hybrid Pattern** which separates business logic from UI presentation while using the template system for consistent layout.

### Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                   Page Model (Public)                    │
│  - Implements core.Page interface                        │
│  - Wraps business logic model                            │
│  - Uses template system for layout                       │
│  - Handles navigation and lifecycle                      │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│              Business Logic Model (Internal)             │
│  - Contains domain logic                                 │
│  - Manages data state                                    │
│  - Handles business operations                           │
│  - Independent of UI framework                           │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                Template System (UI Layer)                │
│  - Header, Sidebar, Content, Footer                      │
│  - Responsive layout                                     │
│  - Consistent styling                                    │
│  - Provider pattern for data injection                   │
└─────────────────────────────────────────────────────────┘
```

---

## File Structure

Each page module should have the following structure:

```
pkg/ui/pages/<page_name>/
├── <page_name>_page.go       # Public Page Model (template wrapper)
├── <page_name>_model.go      # Business Logic Model (internal)
├── <page_name>_providers.go  # Template providers (content, header, sidebar)
├── keys.go                    # Key bindings
├── types.go                   # Message types and constants
├── package.go                 # Package documentation
└── *_test.go                  # Tests
```

**Optional files (for complex pages):**
- `handlers.go` - Event handlers (if business logic is complex)
- `components.go` - Page-specific UI components
- `view.go` - Custom view rendering (if template is insufficient)

---

## Implementation Guidelines

### 1. Business Logic Model (`<page_name>_model.go`)

**Purpose:** Contains domain-specific logic independent of UI framework.

**Characteristics:**
- Plain Go struct (no Bubble Tea dependencies where possible)
- Contains data fields and business state
- Implements business operations as methods
- Can be tested independently
- No direct UI component references

**Example:**
```go
// TopicModel contains the business logic for topic page
type TopicModel struct {
    dataSource      api.KafkaDataSource
    topicName       string
    topicDetails    api.Topic
    messages        []api.Message
    selectedMessage *api.Message
    consuming       bool
    paused          bool
    // ... business state
}

// Business operations
func (m *TopicModel) AddMessage(msg api.Message) {
    m.messages = append(m.messages, msg)
}

func (m *TopicModel) FilterMessages(query string) []api.Message {
    // Filter logic
}

func (m *TopicModel) GetSelectedMessage() *api.Message {
    return m.selectedMessage
}
```

### 2. Page Model (`<page_name>_page.go`)

**Purpose:** Implements `core.Page` interface and wraps business logic with template system.

**Characteristics:**
- Implements `core.Page` interface (Init, Update, View, SetDimensions, GetID, etc.)
- Contains `reusableApp *templateui.ReusableApp`
- Contains `businessModel *BusinessModel`
- Contains providers (content, header, sidebar)
- Handles navigation messages
- Manages lifecycle (OnFocus, OnBlur)

**Example:**
```go
// TopicPageModel implements core.Page interface
type TopicPageModel struct {
    dataSource      api.KafkaDataSource
    topicName       string
    reusableApp     *templateui.ReusableApp
    contentProvider *TopicContentProvider
    businessModel   *TopicModel
}

// Implement core.Page interface
func (m *TopicPageModel) Init() tea.Cmd {
    return m.reusableApp.Init()
}

func (m *TopicPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    updatedApp, cmd := m.reusableApp.Update(msg)
    if app, ok := updatedApp.(*templateui.ReusableApp); ok {
        m.reusableApp = app
    }
    return m, cmd
}

func (m *TopicPageModel) View() string {
    return m.reusableApp.View()
}

func (m *TopicPageModel) GetID() string {
    return "topic"
}
```

### 3. Template Providers (`<page_name>_providers.go`)

**Purpose:** Bridge between business logic and template system.

**Providers to implement:**
- **ContentProvider** - Renders main content area
- **HeaderDataProvider** - Provides header title and status
- **SidebarSections** - Provides sidebar content (optional)

**Example:**
```go
// TopicContentProvider implements providers.ContentProvider
type TopicContentProvider struct {
    model *TopicPageModel
}

func (p *TopicContentProvider) RenderContent(width, height int) string {
    // Use business model data to render content
    return p.model.businessModel.RenderTable(width, height)
}

func (p *TopicContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
    // Handle content-specific updates
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return p.handleKeyPress(msg)
    }
    return nil
}
```

### 4. Key Bindings (`keys.go`)

**Purpose:** Define page-specific key bindings.

**Characteristics:**
- Implements `key.Map` interface (ShortHelp, FullHelp)
- Contains `HandleKey()` method for processing keys
- Returns `tea.Cmd` for side effects

**Example:**
```go
type KeyMap struct {
    Search     key.Binding
    Pause      key.Binding
    Copy       key.Binding
    Back       key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Search, k.Pause, k.Back}
}

func (k KeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Search, k.Pause},
        {k.Copy, k.Back},
    }
}
```

---

## Page Complexity Levels

### Level 1: Simple Pages (Read-only display)

**Example:** Resource Detail Page

**Structure:**
- Page Model only (no separate business model)
- Simple content provider
- Minimal key bindings

**When to use:**
- Display-only pages
- No complex state management
- No real-time updates

### Level 2: Medium Pages (Some interaction)

**Example:** Main Page

**Structure:**
- Page Model with embedded logic
- Content provider with state
- Moderate key bindings

**When to use:**
- List/table display
- Search/filter functionality
- Navigation to other pages

### Level 3: Complex Pages (Real-time, stateful)

**Example:** Topic Page

**Structure:**
- Separate Business Model
- Page Model wrapper
- Complex content provider
- Event handlers
- Advanced key bindings

**When to use:**
- Real-time data streaming
- Complex state management
- Multiple interactive components
- Error recovery logic

---

## Migration Checklist

When migrating a page to the standard architecture:

- [ ] Create/identify business logic model
- [ ] Create page model with template system
- [ ] Implement content provider
- [ ] Implement header data provider
- [ ] Create sidebar sections (if needed)
- [ ] Define key bindings
- [ ] Implement navigation handling
- [ ] Add tests for business logic
- [ ] Add tests for page model
- [ ] Add tests for providers
- [ ] Update router to use new page
- [ ] Remove old implementation

---

## Common Patterns

### Pattern 1: Navigation to Detail Page

```go
// In content provider
func (p *ContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            selectedItem := p.model.GetSelectedItem()
            return core.NewPageChangeMsg("detail", &core.NavigationData{
                ResourceItem: selectedItem,
                ResourceType: p.model.GetResourceType(),
            })
        }
    }
    return nil
}
```

### Pattern 2: Real-time Updates

```go
// In business model
func (m *Model) StartStreaming() tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithCancel(context.Background())
        msgChan := make(chan Message, 100)
        
        go func() {
            defer close(msgChan)
            // Stream data
        }()
        
        return StreamStartedMsg{
            Chan:   msgChan,
            Cancel: cancel,
        }
    }
}
```

### Pattern 3: Error Recovery

```go
// In page model
func (m *PageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ErrorMsg:
        m.businessModel.SetError(msg)
        if m.businessModel.CanRecover() {
            return m, m.businessModel.Retry()
        }
        // Show error to user
        m.reusableApp.ShowError(msg.Error())
    }
    return m.reusableApp.Update(msg)
}
```

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Mixing Business Logic with UI

```go
// BAD: Business logic mixed with UI components
type Model struct {
    table        table.Model      // UI component
    messages     []api.Message    // Business data
    spinner      spinner.Model    // UI component
    consuming    bool             // Business state
}

// GOOD: Separated
type BusinessModel struct {
    messages  []api.Message
    consuming bool
}

type PageModel struct {
    businessModel *BusinessModel
    reusableApp   *templateui.ReusableApp
}
```

### ❌ Anti-Pattern 2: Direct UI Manipulation

```go
// BAD: Directly manipulating UI components
func (m *Model) AddMessage(msg api.Message) {
    row := table.Row{msg.Key, msg.Value}
    m.table.AppendRow(row)  // Direct UI manipulation
}

// GOOD: Update state, let UI re-render
func (m *BusinessModel) AddMessage(msg api.Message) {
    m.messages = append(m.messages, msg)
}

func (p *ContentProvider) RenderContent() string {
    // Render from state
    return renderTable(p.model.businessModel.messages)
}
```

### ❌ Anti-Pattern 3: Skipping Template System

```go
// BAD: Custom layout instead of template
func (m *Model) View() string {
    header := renderHeader()
    sidebar := renderSidebar()
    content := renderContent()
    footer := renderFooter()
    return lipgloss.JoinVertical(header, content, sidebar, footer)
}

// GOOD: Use template system
func (m *PageModel) View() string {
    return m.reusableApp.View()
}
```

---

## Testing Strategy

### Business Logic Tests

```go
func TestTopicModel_AddMessage(t *testing.T) {
    model := NewTopicModel(dataSource, "test-topic", topicDetails)
    
    msg := api.Message{Key: "key1", Value: "value1"}
    model.AddMessage(msg)
    
    assert.Len(t, model.messages, 1)
    assert.Equal(t, "key1", model.messages[0].Key)
}
```

### Page Model Tests

```go
func TestTopicPageModel_ImplementsPage(t *testing.T) {
    var _ core.Page = (*TopicPageModel)(nil)
}

func TestTopicPageModel_Init(t *testing.T) {
    model := NewTopicPageModel(dataSource, "test", topic)
    cmd := model.Init()
    assert.NotNil(t, cmd)
}
```

### Provider Tests

```go
func TestTopicContentProvider_RenderContent(t *testing.T) {
    provider := NewTopicContentProvider(pageModel)
    content := provider.RenderContent(80, 24)
    assert.NotEmpty(t, content)
}
```

---

## Benefits of Standard Architecture

1. **Consistency** - All pages follow the same pattern
2. **Testability** - Business logic can be tested independently
3. **Maintainability** - Clear separation of concerns
4. **Reusability** - Template system provides consistent layout
5. **Extensibility** - Easy to add new pages
6. **Type Safety** - Clear interfaces between layers

---

## Migration Priority

| Page | Current Pattern | Target Pattern | Effort | Priority |
|------|----------------|----------------|--------|----------|
| Main | Template-only | Hybrid (Level 2) | Medium | Low |
| Topic | Standalone MVU | Hybrid (Level 3) | High | **High** |
| Message Detail | Hybrid | Hybrid (Level 2) | Low | Done |
| Resource Detail | Template-only | Hybrid (Level 1) | Low | Done |

---

**Next Steps:**
1. Migrate Topic page to Hybrid pattern (Level 3)
2. Refactor Main page to separate business logic (optional)
3. Document any deviations from standard pattern
