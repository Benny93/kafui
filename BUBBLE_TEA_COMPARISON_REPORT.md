# Bubble Tea API Usage Comparison: Kafui vs Crush

## Executive Summary

This report compares the Bubble Tea TUI implementation between **Kafui** (our Kafka TUI) and **Crush** (a modern AI-powered terminal assistant). The analysis reveals significant architectural and API usage differences that impact code quality, maintainability, and user experience.

**Key Finding**: Crush demonstrates superior Bubble Tea API usage through centralized state management, consistent component architecture, and proper separation of concerns. Kafui's implementation shows several anti-patterns that should be addressed.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Critical API Usage Differences](#critical-api-usage-differences)
3. [Component Architecture](#component-architecture)
4. [State Management](#state-management)
5. [Message Handling](#message-handling)
6. [Key Bindings](#key-bindings)
7. [Styling System](#styling-system)
8. [Error Handling](#error-handling)
9. [Recommendations](#recommendations)

---

## Architecture Overview

### Crush Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      main.go                                 │
│                   tea.NewProgram()                           │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    UI Model (ui.go)                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Central State: session, focus, dialogs, status      │   │
│  │  Components: chat, editor, attachments, completions  │   │
│  │  Layout: Dynamic layout calculation (uiLayout)       │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   ┌────────┐    ┌──────────┐   ┌──────────┐
   │  Chat  │    │  Dialog  │   │  Status  │
   │ Model  │    │ Overlay  │   │  Model   │
   └────────┘    └──────────┘   └──────────┘
```

**Key Characteristics:**
- **Single Root Model**: One `UI` struct manages all application state
- **Component Composition**: Child components (Chat, Dialog, Status) are fields within the main model
- **Centralized Update**: All `Update()` logic stays in the root model
- **No Model Casting**: Child components return their own type, no `tea.Model` casting needed

### Kafui Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      kafui.go                                │
│                   tea.NewProgram()                           │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Root Model (ui.go)                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Router, HelpSystem, FocusManager                    │   │
│  │  ShowHelp flag                                       │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                   Router (router.go)                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  currentPage tea.Model                               │   │
│  │  Pages: main, topic, detail, resource_detail         │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   ┌────────┐    ┌──────────┐   ┌──────────┐
   │  Main  │    │  Topic   │   │  Detail  │
   │  Page  │    │   Page   │   │   Page   │
   └────────┘    └──────────┘   └──────────┘
```

**Key Characteristics:**
- **Router Pattern**: Additional routing layer between root and pages
- **Interface-based Pages**: Pages implement `core.Page` interface
- **Model Casting Required**: Router stores pages as `tea.Model` requiring type assertions
- **Distributed Update Logic**: Update logic spread across router and pages

---

## Critical API Usage Differences

### 1. Model Type Safety

#### ❌ Kafui Anti-Pattern: Model Casting

```go
// pkg/ui/router/router.go
type Router struct {
    currentPage tea.Model  // Loss of type safety!
    // ...
}

func (r *Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Every update requires type assertion
    pageModel, cmd := r.currentPage.Update(msg)
    r.currentPage = pageModel  // Must reassign as tea.Model
}
```

**Problems:**
- Type assertions required on every access
- Compile-time type safety lost
- Easy to introduce runtime panics
- IDE autocomplete doesn't work

#### ✅ Crush Pattern: Concrete Types

```go
// crush/internal/ui/model/ui.go
type UI struct {
    chat *Chat        // Concrete type!
    dialog *dialog.Overlay
    status *Status
    // ...
}

func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Direct method calls, no casting
    m.chat.Update(msg)
    m.dialog.Update(msg)
}
```

**Benefits:**
- Full type safety at compile time
- No runtime type assertions
- Better IDE support
- Clearer code intent

---

### 2. Update Method Return Values

#### ❌ Kafui: Unnecessary Model Returning

```go
// pkg/ui/pages/main/main_page.go
func (m *MainPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    updatedApp, cmd := m.reusableApp.Update(msg)
    if updatedReusableApp, ok := updatedApp.(*templateui.ReusableApp); ok {
        m.reusableApp = updatedReusableApp  // Type assertion needed
    }
    return m, cmd  // Returns self as tea.Model
}
```

#### ✅ Crush: Pointer Receiver Pattern

```go
// crush/internal/ui/model/chat.go
func (m *Chat) HandleKeyMsg(key tea.KeyMsg) (bool, tea.Cmd) {
    // Updates state in-place via pointer
    // Returns whether handled + optional command
}

// crush/internal/ui/model/ui.go
func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Child components updated directly, no model return needed
    m.attachments.Update(msg)
    m.chat.Update(msg)
    return m, tea.Batch(cmds...)
}
```

**Key Insight**: Crush uses pointer receivers extensively, allowing in-place state mutation without needing to return updated models.

---

### 3. Component Initialization

#### ❌ Kafui: Fragmented Initialization

```go
// pkg/ui/ui.go
func initialModelWithRouter(dataSource api.KafkaDataSource) Model {
    r := router.NewRouter(dataSource)
    helpSystem := core.NewHelpSystem()
    focusManager := core.NewFocusManager()
    
    return Model{
        dataSource:   dataSource,
        Router:       r,
        ShowHelp:     false,
        HelpSystem:   helpSystem,
        FocusManager: focusManager,
    }
}
```

#### ✅ Crush: Centralized Constructor

```go
// crush/internal/ui/model/ui.go
func New(com *common.Common) *UI {
    // All components initialized in one place
    ta := textarea.New()
    ta.SetStyles(com.Styles.TextArea)
    // ... configure textarea
    
    ch := NewChat(com)
    keyMap := DefaultKeyMap()
    comp := completions.New(...)
    attachments := attachments.New(...)
    
    ui := &UI{
        com:         com,
        dialog:      dialog.NewOverlay(),
        keyMap:      keyMap,
        textarea:    ta,
        chat:        ch,
        // ... all fields initialized
    }
    
    // Post-initialization setup
    ui.setEditorPrompt(false)
    ui.randomizePlaceholders()
    ui.setState(desiredState, desiredFocus)
    
    return ui
}
```

**Benefits of Crush approach:**
- Single source of truth for initialization
- Clear dependency flow
- Easier to understand component relationships
- Better testability

---

## Component Architecture

### 1. Component Structure

#### ✅ Crush: Consistent Component Pattern

Every component follows the same structure:

```go
// crush/internal/ui/model/chat.go
type Chat struct {
    com  *common.Common  // Shared context
    list *list.List      // Composition
    // ... fields
}

func NewChat(com *common.Common) *Chat {
    // Constructor with clear dependencies
}

func (m *Chat) Draw(scr uv.Screen, area uv.Rectangle) {
    // Rendering
}

func (m *Chat) SetSize(width, height int) {
    // Size management
}

func (m *Chat) HandleKeyMsg(key tea.KeyMsg) (bool, tea.Cmd) {
    // Input handling
}
```

**Pattern:**
1. Struct with `com *common.Common` for shared context
2. Constructor function `NewXxx(com *common.Common)`
3. Explicit `Draw()` or `Render()` method
4. `SetSize()` for dimensions
5. `HandleXxx()` methods for specific events

#### ❌ Kafui: Inconsistent Component Patterns

```go
// pkg/ui/components/search_bar.go
type SearchBarModel struct {
    textInput         textinput.Model
    searchHistory     []string
    // ... many fields
    onSearch          func(query string) tea.Msg  // Callback functions
    onResourceSwitch  func(resource string) tea.Msg
}

// pkg/ui/pages/main/main_page.go
type MainPageModel struct {
    dataSource      api.KafkaDataSource
    reusableApp     *templateui.ReusableApp  // Wrapper pattern
    contentProvider *KafuiContentProvider
}
```

**Issues:**
- No consistent component interface
- Callback functions create tight coupling
- Some components use pointer receivers, others don't
- Mixed responsibility (data loading + UI rendering)

---

### 2. Layout Management

#### ✅ Crush: Declarative Layout System

```go
// crush/internal/ui/model/ui.go
type uiLayout struct {
    sidebar uv.Rectangle  // Using ultraviolet layout system
    main    uv.Rectangle
    header  uv.Rectangle
    footer  uv.Rectangle
    editor  uv.Rectangle
    pills   uv.Rectangle
}

func (m *UI) updateLayoutAndSize() {
    // Calculate layout based on window size and state
    m.layout = calculateLayout(m.width, m.height, m.state, m.isCompact)
    
    // Propagate to child components
    m.chat.SetSize(m.layout.main.Dx(), m.layout.main.Dy())
    m.status.SetWidth(m.width)
}
```

**Benefits:**
- Layout calculated once, propagated to children
- Clear separation between layout and rendering
- Responsive design built-in
- Compact mode support

#### ❌ Kafui: Distributed Layout Logic

```go
// pkg/ui/ui.go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.Router.SetDimensions(msg.Width, msg.Height)
        m.HelpSystem.SetDimensions(msg.Width, msg.Height)
}

// pkg/ui/components/layout.go
type LayoutConfig struct {
    Width         int
    Height        int
    SidebarWidth  int
    ShowSidebar   bool
    // ...
}

func (l Layout) CalculateDimensions() {
    // Each component calculates its own dimensions
}
```

**Issues:**
- Layout logic scattered across components
- No centralized layout calculation
- Components must be updated individually
- Potential for inconsistent dimensions

---

## State Management

### 1. State Representation

#### ✅ Crush: Explicit State Machine

```go
// crush/internal/ui/model/ui.go
type uiState uint8

const (
    uiOnboarding uiState = iota
    uiInitialize
    uiLanding
    uiChat
)

type uiFocusState uint8

const (
    uiFocusNone uiFocusState = iota
    uiFocusEditor
    uiFocusMain
)

type UI struct {
    state uiState
    focus uiFocusState
    // ...
}

func (m *UI) setState(state uiState, focus uiFocusState) {
    if state == uiLanding {
        m.isCompact = false  // Side effects handled centrally
    }
    m.state = state
    m.focus = focus
    m.updateLayoutAndSize()  // Layout updated on state change
}
```

**Benefits:**
- Explicit state enumeration
- Type-safe state transitions
- Centralized state change logic
- Clear state diagram

#### ❌ Kafui: Boolean Flags

```go
// pkg/ui/ui.go
type Model struct {
    ShowHelp     bool  // Boolean flag for state
    // ...
}

// pkg/ui/pages/main/main_page.go
type MainPageModel struct {
    searchMode      bool
    isFiltered      bool
    loading         bool
    // Many boolean flags scattered across models
}
```

**Issues:**
- State represented by multiple booleans
- No clear state machine
- Easy to have inconsistent states
- Hard to track valid state combinations

---

### 2. Shared Context

#### ✅ Crush: Common Context Object

```go
// crush/internal/ui/common/common.go
type Common struct {
    App    *app.App
    Styles *styles.Styles
}

func DefaultCommon(app *app.App) *Common {
    s := styles.DefaultStyles()
    return &Common{
        App:    app,
        Styles: &s,
    }
}

// Usage in components:
type Chat struct {
    com *common.Common  // Injected everywhere
}
```

**Benefits:**
- Single source for dependencies
- Easy to pass to child components
- Consistent access to app/services/styles
- Testable via dependency injection

#### ❌ Kafui: Mixed Dependency Injection

```go
// pkg/ui/pages/main/main_page.go
type MainPageModel struct {
    dataSource api.KafkaDataSource  // Direct dependency
    reusableApp *templateui.ReusableApp
    contentProvider *KafuiContentProvider
}

// pkg/ui/pages/topic/topic_page.go
type Model struct {
    dataSource api.KafkaDataSource  // Same pattern, but...
    handlers *Handlers
    keys *Keys
    view *View
    // Different structure from main page
}
```

**Issues:**
- Inconsistent dependency patterns
- Some pages use wrapper pattern, others don't
- Hard to share common services
- Testing requires different mocks per page

---

## Message Handling

### 1. Message Type Organization

#### ✅ Crush: Typed Message Structs

```go
// crush/internal/ui/model/ui.go
type (
    cancelTimerExpiredMsg struct{}
    userCommandsLoadedMsg struct {
        Commands []commands.CustomCommand
    }
    mcpStateChangedMsg struct {
        states map[string]mcp.ClientInfo
    }
    sendMessageMsg struct {
        Content     string
        Attachments []message.Attachment
    }
)

// Usage in Update:
func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case userCommandsLoadedMsg:
        m.customCommands = msg.Commands
        // Handle message
    case sendMessageMsg:
        cmds = append(cmds, m.sendMessage(msg.Content, msg.Attachments...))
    }
}
```

**Benefits:**
- Clear message contract
- Type-safe message data
- Easy to trace message flow
- Self-documenting code

#### ❌ Kafui: Generic Message Types

```go
// pkg/ui/core/messages.go
type DataLoadedMsg struct {
    Type string  // String-based type discrimination
    Data interface{}  // Interface{} loses type safety
}

type DataErrorMsg struct {
    Type  string
    Error error
}

// Usage requires type assertions:
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case DataLoadedMsg:
        data := msg.Data.(SomeSpecificType)  // Runtime type assertion
}
```

**Issues:**
- Runtime type assertions required
- No compile-time type checking
- Easy to panic with wrong type
- String-based type discrimination error-prone

---

### 2. Command Creation

#### ✅ Crush: Helper Functions

```go
// crush/internal/ui/model/ui.go
func clearInfoMsgCmd(ttl time.Duration) tea.Cmd {
    return tea.Tick(ttl, func(time.Time) tea.Msg {
        return util.ClearStatusMsg{}
    })
}

// Usage:
cmds = append(cmds, clearInfoMsgCmd(ttl))
```

#### ❌ Kafui: Inline Command Creation

```go
// pkg/ui/pages/topic/topic_page.go
func (m *Model) startConsumption() tea.Cmd {
    return func() tea.Msg {
        // Inline command logic
        msgChan, errChan, cancel, err := m.dataSource.ConsumeMessages(...)
        // ...
    }
}
```

**Note**: Both approaches are valid, but Crush's approach provides better reusability and testability.

---

## Key Bindings

### 1. Key Map Structure

#### ✅ Crush: Centralized Key Maps

```go
// crush/internal/ui/model/keys.go
type KeyMap struct {
    Editor struct {
        AddFile     key.Binding
        SendMessage key.Binding
        // ...
    }
    Chat struct {
        NewSession     key.Binding
        AddAttachment  key.Binding
        // ...
    }
    Quit     key.Binding
    Help     key.Binding
    // ...
}

func DefaultKeyMap() KeyMap {
    km := KeyMap{
        Quit: key.NewBinding(
            key.WithKeys("ctrl+c"),
            key.WithHelp("ctrl+c", "quit"),
        ),
        // ... all bindings defined in one place
    }
    return km
}
```

**Benefits:**
- All key bindings in one file
- Clear organization by feature
- Easy to audit and modify
- Consistent help text

#### ❌ Kafui: Distributed Key Bindings

```go
// pkg/ui/pages/main/main_page.go
type KeyMap struct {
    Search         key.Binding
    SwitchResource key.Binding
    // ...
}

var DefaultKeyMap = KeyMap{
    Search: key.NewBinding(
        key.WithKeys("/"),
        key.WithHelp("/", "search"),
    ),
    // ...
}

// pkg/ui/pages/topic/topic_page.go
type Keys struct {
    model *Model
}

func (k *Keys) ShortHelp() []key.Binding {
    // Different pattern from main page
}
```

**Issues:**
- Key bindings defined per-page
- No centralized overview
- Inconsistent patterns (struct vs variable)
- Hard to detect conflicts

---

### 2. Help System Integration

#### ✅ Crush: Integrated Help

```go
// crush/internal/ui/model/status.go
type Status struct {
    help help.Model
    helpKm help.KeyMap
}

func (s *Status) Draw(scr uv.Screen, area uv.Rectangle) {
    if !s.hideHelp {
        helpView := s.com.Styles.Status.Help.Render(
            s.help.View(s.helpKm)
        )
        uv.NewStyledString(helpView).Draw(scr, area)
    }
}
```

**Benefits:**
- Help rendered as part of status bar
- Consistent help display
- Toggle support built-in

#### ❌ Kafui: Separate Help System

```go
// pkg/ui/ui.go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, core.DefaultGlobalKeys.Help):
            m.ShowHelp = !m.ShowHelp
            m.HelpSystem.Toggle()
        }
}

func (m Model) View() string {
    if m.ShowHelp {
        return m.HelpSystem.Render()
    }
    return m.Router.View()
}
```

**Issues:**
- Help is a separate overlay mode
- Not integrated into normal view
- Requires state flag management

---

## Styling System

### 1. Style Organization

#### ✅ Crush: Comprehensive Style System

```go
// crush/internal/ui/styles/styles.go
type Styles struct {
    // Reusable text styles
    Base      lipgloss.Style
    Muted     lipgloss.Style
    HalfMuted lipgloss.Style
    
    // Semantic colors
    Primary   color.Color
    Secondary color.Color
    Error     color.Color
    // ...
    
    // Component-specific styles
    Chat struct {
        Message struct {
            UserBlurred      lipgloss.Style
            UserFocused      lipgloss.Style
            // ...
        }
    }
    
    // Dialog styles
    Dialog struct {
        Title       lipgloss.Style
        View        lipgloss.Style
        // ...
    }
}

func DefaultStyles() Styles {
    // All styles defined with semantic colors
    // Using charmtone color palette
}
```

**Benefits:**
- Centralized style definitions
- Semantic color names
- Component-specific style groups
- Consistent theming

#### ❌ Kafui: Basic Style System

```go
// pkg/ui/core/styles.go
type GlobalStyles struct {
    Theme Theme
}

func (gs *GlobalStyles) HeaderText() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(lipgloss.Color(gs.Theme.Primary)).
        Bold(true)
}

// pkg/ui/pages/main/main_page.go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#7D56F4"))  // Hard-coded color
)
```

**Issues:**
- Styles created ad-hoc in components
- Hard-coded colors in some places
- No centralized style management
- Theme support limited

---

### 2. Color Palette

#### ✅ Crush: Professional Color System

```go
// Using charmtone library
var (
    primary   = charmtone.Charple
    secondary = charmtone.Dolly
    tertiary  = charmtone.Bok
    
    bgBase        = charmtone.Pepper
    bgBaseLighter = charmtone.BBQ
    bgSubtle      = charmtone.Charcoal
    
    fgBase      = charmtone.Ash
    fgMuted     = charmtone.Squid
    fgHalfMuted = charmtone.Smoke
    
    error   = charmtone.Sriracha
    warning = charmtone.Zest
    info    = charmtone.Malibu
)
```

**Benefits:**
- Curated color palette
- Consistent naming
- Professional appearance
- Light/dark mode support

#### ❌ Kafui: Basic Color Definitions

```go
// pkg/ui/core/styles.go
func DefaultTheme() Theme {
    return Theme{
        Primary:   "#7D56F4",
        Secondary: "#383838",
        Accent:    "#73F59F",
        Error:     "#F25D94",
        // ...
    }
}
```

**Issues:**
- Hex colors without semantic meaning
- No curated palette
- Limited color variations

---

## Error Handling

### 1. Error Display

#### ✅ Crush: Multi-level Error Handling

```go
// crush/internal/ui/model/ui.go
case util.InfoMsg:
    m.status.SetInfoMsg(msg)
    ttl := msg.TTL
    if ttl <= 0 {
        ttl = DefaultStatusTTL
    }
    cmds = append(cmds, clearInfoMsgCmd(ttl))

// crush/internal/ui/util/errors.go
func ReportError(err error) tea.Cmd {
    return func() tea.Msg {
        return InfoMsg{
            Type: InfoTypeError,
            Msg:  err.Error(),
            TTL:  10 * time.Second,
        }
    }
}
```

**Benefits:**
- Errors shown in status bar
- Auto-dismiss with TTL
- Different error types (error, warn, info, success)
- Non-intrusive error display

#### ❌ Kafui: Basic Error Handling

```go
// pkg/ui/pages/topic/topic_page.go
type Model struct {
    error        error
    lastError    error
    errorHistory []error
    retryCount   int
    // ...
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case DataErrorMsg:
        m.error = msg.Error
        m.statusMessage = fmt.Sprintf("Error: %v", msg.Error)
}
```

**Issues:**
- Errors stored in model state
- Manual error history management
- No standardized error display
- Error messages may block UI

---

## Recommendations

### High Priority

#### 1. **Eliminate Model Casting**

**Current:**
```go
type Router struct {
    currentPage tea.Model
}
```

**Recommended:**
```go
type Router struct {
    currentPage Page  // Use interface or concrete type
}
```

**Benefit**: Type safety, no runtime assertions, better IDE support.

---

#### 2. **Adopt Pointer Receivers**

**Current:**
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

**Recommended:**
```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

**Benefit**: In-place mutation, no need to return updated model, clearer intent.

---

#### 3. **Centralize State Management**

**Current:**
```go
type Model struct {
    ShowHelp     bool
    SearchMode   bool
    Loading      bool
    // Many boolean flags
}
```

**Recommended:**
```go
type uiState uint8

const (
    stateNormal uiState = iota
    stateHelp
    stateSearch
)

type Model struct {
    state uiState
    focus uiFocusState
}
```

**Benefit**: Explicit state machine, type-safe transitions, easier debugging.

---

#### 4. **Use Typed Message Structs**

**Current:**
```go
type DataLoadedMsg struct {
    Type string
    Data interface{}
}
```

**Recommended:**
```go
type TopicsLoadedMsg struct {
    Topics map[string]api.Topic
}

type ConsumerGroupsLoadedMsg struct {
    Groups []api.ConsumerGroup
}
```

**Benefit**: Type safety, no assertions, self-documenting code.

---

#### 5. **Centralize Key Bindings**

**Current:** Key bindings defined per-page

**Recommended:** Single `keys.go` file with all bindings:
```go
type KeyMap struct {
    Main MainKeyMap
    Topic TopicKeyMap
    Global GlobalKeyMap
}
```

**Benefit**: Easy to audit, detect conflicts, maintain consistency.

---

### Medium Priority

#### 6. **Adopt Common Context Pattern**

**Recommended:**
```go
type Common struct {
    App    *app.App
    Styles *styles.Styles
}

func NewMainPage(common *common.Common) *MainPage {
    return &MainPage{com: common}
}
```

**Benefit**: Consistent dependency injection, easier testing.

---

#### 7. **Improve Style System**

**Recommended:**
- Use semantic color names
- Create component-specific style groups
- Centralize all styles in one package

**Benefit**: Consistent theming, easier maintenance.

---

#### 8. **Implement Declarative Layout**

**Recommended:**
```go
type Layout struct {
    Sidebar, Main, Header, Footer Rectangle
}

func CalculateLayout(width, height int, state uiState) Layout
```

**Benefit**: Centralized layout logic, consistent dimensions.

---

### Low Priority

#### 9. **Enhance Error Display**

**Recommended:**
- Use status bar for errors
- Auto-dismiss with TTL
- Different severity levels

**Benefit**: Less intrusive error handling.

---

#### 10. **Standardize Component Pattern**

**Recommended:**
```go
type Component struct {
    com *common.Common
}

func NewComponent(com *common.Common) *Component
func (c *Component) SetSize(width, height int)
func (c *Component) HandleKeyMsg(msg tea.KeyMsg) (bool, tea.Cmd)
```

**Benefit**: Consistent component API, easier to understand.

---

## Conclusion

Crush demonstrates several best practices for Bubble Tea applications that Kafui should adopt:

1. **Type Safety**: Avoid `tea.Model` casting; use concrete types or interfaces
2. **Pointer Receivers**: Enable in-place mutation, simplify Update methods
3. **Centralization**: State, styles, key bindings, and layout in one place
4. **Typed Messages**: Use specific message types instead of generic containers
5. **Component Pattern**: Consistent structure across all components
6. **Professional Styling**: Semantic colors, curated palettes

Implementing these changes will significantly improve Kafui's code quality, maintainability, and developer experience.

---

## Appendix: File Structure Comparison

### Crush Structure
```
internal/ui/
├── model/
│   ├── ui.go           # Main UI model
│   ├── chat.go         # Chat component
│   ├── keys.go         # All key bindings
│   ├── status.go       # Status bar
│   └── ...
├── common/
│   └── common.go       # Shared context
├── styles/
│   └── styles.go       # All styles
└── components/
    ├── attachments/
    ├── completions/
    ├── dialog/
    └── ...
```

### Kafui Structure
```
pkg/ui/
├── ui.go               # Root model
├── kafui.go            # Entry point
├── core/
│   ├── interfaces.go   # Page interface
│   ├── messages.go     # Generic messages
│   └── styles.go       # Basic styles
├── components/         # Legacy components
├── pages/
│   ├── main/
│   ├── topic/
│   └── ...
└── router/
    └── router.go       # Routing logic
```

**Recommendation**: Consider restructuring to match Crush's flatter, more organized hierarchy.
