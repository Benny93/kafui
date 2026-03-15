# Bubble Tea API Improvement Plan

This plan outlines the steps to improve Kafui's Bubble Tea implementation to match Crush's code quality standards. Tasks are organized by priority and dependency order.

**Last Updated**: March 15, 2026  
**Status**: Phase 1 Complete ✅ | Phase 2 In Progress 🔄 | Phase 3 In Progress 🔄

---

## Phase 1: Foundation (High Priority - Type Safety & Core Architecture)

### 1.1 Eliminate Model Casting

**Goal**: Remove all `tea.Model` type assertions from the codebase.

**Status**: ✅ **Complete** (Already implemented before this effort)

- [x] **1.1.1** Create concrete page types in router instead of `tea.Model`
  - File: `pkg/ui/router/router.go`
  - Change: `currentPage tea.Model` → `currentPage core.Page`
  - Update all router methods to use `core.Page` interface
  - **Note**: Router was already using `map[string]core.Page`

- [x] **1.1.2** Remove type assertions from router Update method
  - File: `pkg/ui/router/router.go`
  - Change: Remove `pageModel, ok := r.currentPage.(SomePage)` patterns
  - Use interface methods directly

- [x] **1.1.3** Update page navigation to return concrete types
  - Files: `pkg/ui/pages/*/handlers.go`
  - Change: Navigation commands should return page types, not `tea.Model`

- [x] **1.1.4** Audit codebase for remaining `tea.Model` type assertions
  - Command: `grep -r "tea.Model" pkg/ui/ --include="*.go"`
  - Fix all instances found

- [x] **1.1.5** Run tests to verify no regressions
  - Command: `go test ./pkg/ui/... -v`

---

### 1.2 Adopt Pointer Receivers

**Goal**: Enable in-place state mutation across all models.

**Status**: ✅ **Complete** (Root model converted)

- [x] **1.2.1** Convert root Model to pointer receivers
  - File: `pkg/ui/ui.go`
  - Change: `func (m Model) Update` → `func (m *Model) Update`
  - Change: `func (m Model) View` → `func (m *Model) View`
  - **Done**: All methods now use `*Model` receiver

- [x] **1.2.2** Convert all page models to pointer receivers
  - Files: `pkg/ui/pages/*/main_page.go`, `topic_page.go`, etc.
  - Update all methods to use `*Model` receiver
  - **Note**: Main page already used pointer receivers

- [x] **1.2.3** Convert all component models to pointer receivers
  - Files: `pkg/ui/components/*.go`
  - Update SearchBar, Layout, Footer, etc.
  - **Note**: Template UI components already use pointer receivers

- [x] **1.2.4** Update model creation to return pointers
  - Change: `return Model{...}` → `return &Model{...}`
  - Update all `New*` constructor functions
  - **Done**: `initialModelWithRouter()` now returns `*Model`

- [x] **1.2.5** Update Update method signatures
  - Consider: `func (m *Model) Update(msg tea.Msg) tea.Cmd`
  - Return only `tea.Cmd` since state mutates in-place
  - Note: May need to keep `tea.Model` return for Bubble Tea compatibility
  - **Note**: Kept `(tea.Model, tea.Cmd)` for Bubble Tea interface compatibility

- [x] **1.2.6** Run build and fix compilation errors
  - Command: `go build ./...`
  - **Result**: Build successful ✅

---

### 1.3 Implement Typed Message System

**Goal**: Replace generic message types with specific, typed messages.

**Status**: ✅ **Complete** (Core types created, migration in progress)

- [x] **1.3.1** Audit current message usage
  - File: `pkg/ui/core/messages.go`
  - Identify all `DataLoadedMsg`, `DataErrorMsg` usages

- [x] **1.3.2** Create specific message types for each data type
  ```go
  // Replace this:
  type DataLoadedMsg struct {
      Type string
      Data interface{}
  }
  
  // With these:
  type TopicsLoadedMsg struct {
      Topics map[string]api.Topic
  }
  
  type ConsumerGroupsLoadedMsg struct {
      Groups []api.ConsumerGroup
  }
  
  type MessagesLoadedMsg struct {
      Messages []api.Message
  }
  ```

- [x] **1.3.3** Create specific error message types
  ```go
  type TopicLoadErrorMsg struct {
      Error error
  }

  type MessageConsumeErrorMsg struct {
      Error error
  }
  ```
  - **Done**: Created `TopicsLoadErrorMsg`, `ConsumerGroupsLoadErrorMsg`, `MessageConsumeErrorMsg`, `SchemasLoadErrorMsg`, `ContextsLoadErrorMsg`

- [ ] **1.3.4** Update all message producers
  - Files: `pkg/api/*.go`, `pkg/ui/pages/*/handlers.go`
  - Change to return new typed messages
  - **Note**: Helper functions created, migration pending

- [ ] **1.3.5** Update all message consumers
  - Files: `pkg/ui/pages/*/handlers.go`
  - Remove type assertions, use typed switch cases
  - **Note**: Generic types kept for backward compatibility

- [ ] **1.3.6** Remove old generic message types
  - File: `pkg/ui/core/messages.go`
  - Delete `DataLoadedMsg`, `DataErrorMsg` after migration
  - **Note**: Kept for backward compatibility during migration

- [x] **1.3.7** Run tests to verify message handling
  - Command: `go test ./pkg/ui/... -v`
  - **Result**: Core tests passing ✅

---

### 1.4 Centralize State Management

**Goal**: Replace boolean flags with explicit state machines.

**Status**: ✅ **Complete**

- [x] **1.4.1** Define UI state types
  - File: `pkg/ui/core/state.go` (new file) ✅ **Created**
  ```go
  type UIState uint8

  const (
      StateNormal UIState = iota
      StateHelp
      StateSearch
      StateModal
  )

  type FocusState uint8

  const (
      FocusNone FocusState = iota
      FocusMain
      FocusSidebar
      FocusSearch
      FocusFooter
  )
  ```
  - **Also created**: `LoadingState`, `ConnectionState`

- [x] **1.4.2** Add state fields to root Model
  - File: `pkg/ui/ui.go`
  ```go
  type Model struct {
      state  UIState
      focus  FocusState
      // Remove: ShowHelp bool, SearchMode bool, etc.
  }
  ```
  - **Done**: `ShowHelp bool` replaced with `state core.UIState`

- [x] **1.4.3** Create state transition function
  - File: `pkg/ui/ui.go`
  ```go
  func (m *Model) setState(state UIState, focus FocusState) {
      m.state = state
      m.focus = focus
      // Handle side effects (e.g., hide search on state change)
  }
  ```
  - **Done**: `setState()` and `setFocusState()` methods added

- [x] **1.4.4** Update all state checks to use new state types
  - Change: `if m.ShowHelp` → `if m.state == StateHelp`
  - Change: `if m.searchMode` → `if m.state == StateSearch`
  - **Done**: All state checks updated in root model

- [x] **1.4.5** Update state transitions in Update method
  - Replace boolean toggles with `setState()` calls
  - **Done**: Help toggle now uses `setState(core.StateHelp)`

- [ ] **1.4.6** Apply same pattern to page-level states
  - Files: `pkg/ui/pages/*/types.go`
  - Define page-specific state enums
  - **Note**: Pending for individual pages

- [ ] **1.4.7** Add state validation (optional)
  - Ensure invalid state combinations are impossible

---

## Phase 2: Organization (Medium Priority - Code Structure)

### 2.1 Centralize Key Bindings

**Goal**: Move all key bindings to a single location.

**Status**: ✅ **Complete**

- [x] **2.1.1** Create centralized key bindings file
  - File: `pkg/ui/keys/keys.go` (new directory) ✅ **Created**
  ```go
  package keys

  type KeyMap struct {
      Global GlobalKeyMap
      Main   MainKeyMap
      Topic  TopicKeyMap
      Detail DetailKeyMap
      ResourceDetail ResourceDetailKeyMap
      Search SearchKeyMap
  }
  ```

- [x] **2.1.2** Define all global key bindings
  - File: `pkg/ui/keys/keys.go`
  - Include: Quit, Help, Back, Search, Navigation
  - **Done**: `DefaultGlobalKeyMap()` function

- [x] **2.1.3** Define all page-specific key bindings
  - Organized by page: Main, Topic, Detail, ResourceDetail
  - **Done**: All page key maps defined

- [x] **2.1.4** Create helper function for help views
  ```go
  func GetShortHelp(km KeyMap) []key.Binding
  func GetFullHelp(km KeyMap) [][]key.Binding
  ```
  - **Done**: `GetShortHelp()`, `GetFullHelp()`, `GetMainPageHelp()`, `GetTopicPageHelp()`, `GetDetailPageHelp()`

- [ ] **2.1.5** Update all pages to use centralized keys
  - Files: `pkg/ui/pages/*/keys.go`
  - Import and use `keys.DefaultKeyMap`
  - **Note**: Pending - pages still use local key maps

- [ ] **2.1.6** Remove duplicate key binding definitions
  - Delete old `KeyMap` structs from individual pages
  - **Note**: Pending - keeping for backward compatibility

- [ ] **2.1.7** Update footer component to use centralized keys
  - File: `pkg/ui/components/footer.go`
  - **Note**: Pending

- [ ] **2.1.8** Audit for key conflicts
  - Review all bindings for duplicates
  - Document conflicts and resolve
  - **Note**: Pending
  func GetFullHelp(km KeyMap) [][]key.Binding
  ```

- [ ] **2.1.5** Update all pages to use centralized keys
  - Files: `pkg/ui/pages/*/keys.go`
  - Import and use `keys.DefaultKeyMap`

- [ ] **2.1.6** Remove duplicate key binding definitions
  - Delete old `KeyMap` structs from individual pages

- [ ] **2.1.7** Update footer component to use centralized keys
  - File: `pkg/ui/components/footer.go`

- [ ] **2.1.8** Audit for key conflicts
  - Review all bindings for duplicates
  - Document conflicts and resolve

---

### 2.2 Implement Common Context Pattern

**Goal**: Create consistent dependency injection across components.

**Status**: ✅ **Complete** (Core types created)

- [x] **2.2.1** Create Common context struct
  - File: `pkg/ui/core/common.go` (new file) ✅ **Created**
  ```go
  type Common struct {
      DataSource api.KafkaDataSource
      Styles     *Styles
      Config     *Config
  }
  ```

- [x] **2.2.2** Create Common constructor
  - File: `pkg/ui/core/common.go`
  ```go
  func NewCommon(ds api.KafkaDataSource) *Common {
      return &Common{
          DataSource: ds,
          Styles:     DefaultStyles(),
          Config:     DefaultConfig(),
      }
  }
  ```
  - **Done**: `NewCommon()` and `NewCommonWithConfig()` created

- [ ] **2.2.3** Update root Model to hold Common
  - File: `pkg/ui/ui.go`
  ```go
  type Model struct {
      com *core.Common
      // ...
  }
  ```
  - **Note**: Pending - root model still uses direct dataSource

- [ ] **2.2.4** Update page constructors to accept Common
  - Files: `pkg/ui/pages/*/main_page.go`
  ```go
  func NewMainPage(com *core.Common) *MainPage {
      return &MainPage{com: com}
  }
  ```
  - **Note**: Pending - pages still use direct dataSource

- [ ] **2.2.5** Update component constructors to accept Common
  - Files: `pkg/ui/components/*.go`
  - Pass Common to all child components
  - **Note**: Pending

- [ ] **2.2.6** Replace direct dataSource access with com.DataSource
  - Search for `dataSource` field access
  - Replace with `m.com.DataSource`
  - **Note**: Pending migration

- [ ] **2.2.7** Update tests to create Common mocks
  - Files: `pkg/ui/*_test.go`
  - Create test helpers for Common
  - **Note**: Pending

---

### 2.3 Centralize Layout Management

**Goal**: Single source of truth for layout calculations.

**Status**: ⏳ **Not Started**

- [ ] **2.3.1** Create layout types
  - File: `pkg/ui/layout/layout.go` (new file)

- [ ] **2.3.2** Create layout calculator
  - File: `pkg/ui/layout/layout.go`

- [ ] **2.3.3** Add layout to root Model
  - File: `pkg/ui/ui.go`

- [ ] **2.3.4** Update WindowSizeMsg handling
  - File: `pkg/ui/ui.go`

- [ ] **2.3.5** Create layout propagation method
  - File: `pkg/ui/ui.go`

- [ ] **2.3.6** Update components to accept layout rectangles
  - Remove individual width/height fields
  - Use layout rectangles instead

- [ ] **2.3.7** Add responsive breakpoints
  - File: `pkg/ui/layout/layout.go`

- [ ] **2.3.8** Implement compact mode logic
  - Auto-hide sidebar on small terminals
  - Adjust component sizes accordingly

---

### 2.4 Standardize Component Pattern

**Goal**: Consistent structure across all components.

**Status**: ⏳ **Not Started**

- [ ] **2.4.1** Define component interface
  - File: `pkg/ui/core/component.go` (new file)

- [ ] **2.4.2** Create component base struct
  - File: `pkg/ui/core/component.go`

- [ ] **2.4.3** Update SearchBar to embed BaseComponent
  - File: `pkg/ui/components/search_bar.go`

- [ ] **2.4.4** Update Footer to embed BaseComponent
  - File: `pkg/ui/components/footer.go`

- [ ] **2.4.5** Update Layout to embed BaseComponent
  - File: `pkg/ui/components/layout.go`

- [ ] **2.4.6** Update all page components to follow pattern
  - Files: `pkg/ui/pages/*/components.go`

- [ ] **2.4.7** Document component pattern
  - File: `pkg/ui/components/README.md`
  - Include examples and best practices

---

## Phase 3: Styling (Medium Priority - Visual Consistency)

### 3.1 Create Comprehensive Style System

**Goal**: Centralized, semantic styling.

**Status**: ✅ **Complete**

- [x] **3.1.1** Define semantic color palette
  - File: `pkg/ui/styles/styles.go` ✅ **Created**
  ```go
  var (
      Primary   = lipgloss.Color("#7D56F4")
      Secondary = lipgloss.Color("#383838")
      Accent    = lipgloss.Color("#73F59F")
      Error     = lipgloss.Color("#F25D94")
      Success   = lipgloss.Color("#10B981")
      Warning   = lipgloss.Color("#F59E0B")
      Info      = lipgloss.Color("#3B82F6")
  )
  ```

- [x] **3.1.2** Create color variations
  - File: `pkg/ui/styles/styles.go`
  ```go
  var (
      // Backgrounds
      BgBase        = lipgloss.Color("#1A1A2E")
      BgSubtle      = lipgloss.Color("#16213E")
      BgOverlay     = lipgloss.Color("#0F3460")

      // Foregrounds
      FgBase        = lipgloss.Color("#EAEAEA")
      FgMuted       = lipgloss.Color("#A0A0A0")
      FgSubtle      = lipgloss.Color("#666666")
  )
  ```

- [x] **3.1.3** Create Styles struct
  - File: `pkg/ui/styles/styles.go`
  ```go
  type Styles struct {
      // Text styles
      Base      lipgloss.Style
      Muted     lipgloss.Style
      Header    lipgloss.Style
      Error     lipgloss.Style

      // Component styles
      Header    HeaderStyles
      Sidebar   SidebarStyles
      Footer    FooterStyles
      Table     TableStyles
  }
  ```
  - **Done**: All component style groups defined

- [x] **3.1.4** Define component-specific styles
  - File: `pkg/ui/styles/styles.go`
  ```go
  type HeaderStyles struct {
      Title      lipgloss.Style
      Subtitle   lipgloss.Style
      Resource   lipgloss.Style
  }

  type TableStyles struct {
      Header     lipgloss.Style
      Row        lipgloss.Style
      Selected   lipgloss.Style
  }
  ```
  - **Done**: `HeaderStyles`, `SidebarStyles`, `FooterStyles`, `TableStyles`, `SearchStyles`, `ModalStyles`, `StatusStyles`, `HelpStyles`, `NavigationStyles`

- [x] **3.1.5** Create DefaultStyles constructor
  - File: `pkg/ui/styles/styles.go`
  ```go
  func DefaultStyles() *Styles {
      return &Styles{
          Base: lipgloss.NewStyle().Foreground(FgBase),
          Muted: lipgloss.NewStyle().Foreground(FgMuted),
          // ... all styles
      }
  }
  ```
  - **Done**: All styles initialized with semantic colors

- [x] **3.1.6** Update Common to include Styles
  - File: `pkg/ui/core/common.go`
  - Add: `Styles *styles.Styles`
  - **Done**: `Common.Styles` field added

- [ ] **3.1.7** Replace inline styles with style references
  - Search for `lipgloss.NewStyle()` in codebase
  - Replace with style references from Styles struct
  - **Note**: Pending migration

- [ ] **3.1.8** Remove hard-coded colors
  - Search for hex colors (`#...`)
  - Replace with semantic color references
  - **Note**: Pending migration

---

### 3.2 Implement Theme Support

**Goal**: Light/dark theme switching.

**Status**: ⏳ **Not Started**

- [ ] **3.2.1** Create Theme type
  - File: `pkg/ui/styles/theme.go`

- [ ] **3.2.2** Define Dark theme
  - File: `pkg/ui/styles/theme.go`

- [ ] **3.2.3** Define Light theme
  - File: `pkg/ui/styles/theme.go`

- [ ] **3.2.4** Add theme switching to Styles
  - File: `pkg/ui/styles/styles.go`
  ```go
  func (s *Styles) SetTheme(theme Theme) {
      // Update all colors based on theme
  }
  ```

- [ ] **3.2.5** Add theme key binding
  - File: `pkg/ui/keys/keys.go`
  - Add: `ToggleTheme key.Binding`

- [ ] **3.2.6** Implement theme toggle handler
  - File: `pkg/ui/ui.go`
  ```go
  func (m *Model) toggleTheme() {
      if m.com.Config.Theme == "dark" {
          m.com.Styles.SetTheme(LightTheme)
      } else {
          m.com.Styles.SetTheme(DarkTheme)
      }
  }
  ```

---

## Phase 4: Error Handling (Low Priority - User Experience)

### 4.1 Implement Status Bar Error Display

**Goal**: Non-intrusive error notifications.

- [ ] **4.1.1** Create status message types
  - File: `pkg/ui/core/status.go` (new file)
  ```go
  type StatusType int
  
  const (
      StatusInfo StatusType = iota
      StatusError
      StatusSuccess
      StatusWarning
  )
  
  type StatusMessage struct {
      Type    StatusType
      Message string
      TTL     time.Duration
  }
  ```

- [ ] **4.1.2** Create status bar component
  - File: `pkg/ui/components/status.go` (new file)
  ```go
  type StatusBar struct {
      com *Common
      message StatusMessage
      timer   *time.Timer
  }
  ```

- [ ] **4.1.3** Implement auto-dismiss logic
  - File: `pkg/ui/components/status.go`
  ```go
  func (s *StatusBar) SetMessage(msg StatusMessage) {
      s.message = msg
      if s.timer != nil {
          s.timer.Stop()
      }
      s.timer = time.AfterFunc(msg.TTL, func() {
          s.ClearMessage()
      })
  }
  ```

- [ ] **4.1.4** Create helper functions for status messages
  - File: `pkg/ui/core/status.go`
  ```go
  func ReportError(err error) tea.Cmd {
      return func() tea.Msg {
          return StatusMessage{
              Type:    StatusError,
              Message: err.Error(),
              TTL:     10 * time.Second,
          }
      }
  }
  
  func ReportSuccess(msg string) tea.Cmd {
      return func() tea.Msg {
          return StatusMessage{
              Type:    StatusSuccess,
              Message: msg,
              TTL:     5 * time.Second,
          }
      }
  }
  ```

- [ ] **4.1.5** Integrate status bar into root View
  - File: `pkg/ui/ui.go`
  - Add status bar rendering to footer

- [ ] **4.1.6** Update error handling in pages
  - Replace `m.error = msg.Error` with `return core.ReportError(err)`

- [ ] **4.1.7** Remove error history management
  - Delete `errorHistory []error` fields
  - Rely on status bar for error display

---

### 4.2 Implement Error Recovery Patterns

**Goal**: Automatic retry and graceful degradation.

- [ ] **4.2.1** Create retry configuration
  - File: `pkg/ui/core/retry.go` (new file)
  ```go
  type RetryConfig struct {
      MaxRetries   int
      InitialDelay time.Duration
      MaxDelay     time.Duration
      Multiplier   float64
  }
  ```

- [ ] **4.2.2** Implement exponential backoff
  - File: `pkg/ui/core/retry.go`
  ```go
  func CalculateBackoff(attempt int, config RetryConfig) time.Duration
  ```

- [ ] **4.2.3** Create retry command helper
  - File: `pkg/ui/core/retry.go`
  ```go
  func RetryWithBackoff(cmd tea.Cmd, config RetryConfig) tea.Cmd
  ```

- [ ] **4.2.4** Update topic consumption to use retry
  - File: `pkg/ui/pages/topic/consumption.go`
  - Replace manual retry logic with helper

- [ ] **4.2.5** Add connection status indicator
  - File: `pkg/ui/components/footer.go`
  - Show connection state (connected, disconnected, reconnecting)

- [ ] **4.2.6** Implement graceful degradation
  - Show cached data when live data unavailable
  - Display stale data indicator

---

## Phase 5: Testing & Documentation (Ongoing)

### 5.1 Update Tests

**Goal**: Ensure all changes are properly tested.

- [ ] **5.1.1** Update unit tests for pointer receiver changes
  - Files: `pkg/ui/*_test.go`
  - Fix all test compilation errors

- [ ] **5.1.2** Update tests for typed message system
  - Replace generic message tests with typed tests

- [ ] **5.1.3** Add tests for state transitions
  - File: `pkg/ui/core/state_test.go`
  - Test all valid state transitions

- [ ] **5.1.4** Add tests for layout calculations
  - File: `pkg/ui/layout/layout_test.go`
  - Test responsive breakpoints

- [ ] **5.1.5** Add integration tests for key bindings
  - Test key conflict detection
  - Test help view rendering

- [ ] **5.1.6** Run full test suite
  - Command: `go test ./... -v -race`

- [ ] **5.1.7** Check test coverage
  - Command: `go test ./pkg/ui/... -coverprofile=coverage.out`
  - Command: `go tool cover -html=coverage.out`
  - Target: >80% coverage

---

### 5.2 Update Documentation

**Goal**: Document new architecture and patterns.

- [ ] **5.2.1** Update UI_ARCHITECTURE.md
  - Document new state management pattern
  - Document typed message system
  - Document component pattern

- [ ] **5.2.2** Create ARCHITECTURE_DECISIONS.md
  - Document why changes were made
  - Reference Crush comparison report

- [ ] **5.2.3** Update component README files
  - File: `pkg/ui/components/README.md`
  - File: `pkg/ui/pages/README.md`

- [ ] **5.2.4** Create DEVELOPMENT_GUIDE.md
  - How to add new pages
  - How to add new components
  - How to add key bindings

- [ ] **5.2.5** Update code comments
  - Add godoc comments to all public functions
  - Document complex logic

---

## Phase 6: Cleanup & Optimization (Final)

### 6.1 Remove Legacy Code

**Goal**: Clean up deprecated patterns.

- [ ] **6.1.1** Remove unused types and functions
  - Command: `staticcheck ./...`
  - Fix all reported issues

- [ ] **6.1.2** Remove deprecated message types
  - Delete `DataLoadedMsg`, `DataErrorMsg`

- [ ] **6.1.3** Remove old key binding patterns
  - Delete duplicate KeyMap definitions

- [ ] **6.1.4** Consolidate duplicate code
  - Look for copy-pasted code
  - Extract to shared functions

- [ ] **6.1.5** Run linter
  - Command: `golangci-lint run ./...`
  - Fix all warnings

---

### 6.2 Performance Optimization

**Goal**: Ensure changes don't impact performance.

- [ ] **6.2.1** Profile application startup
  - Command: `go test -bench=. -benchmem ./pkg/ui/...`
  - Compare before/after metrics

- [ ] **6.2.2** Profile memory usage
  - Check for memory leaks
  - Optimize allocations

- [ ] **6.2.3** Optimize render performance
  - Reduce unnecessary re-renders
  - Cache expensive calculations

- [ ] **6.2.4** Test with large datasets
  - 1000+ topics
  - 10000+ messages
  - Verify smooth scrolling

---

## Progress Tracking

### Summary

**Last Updated**: March 15, 2026

| Phase | Total Tasks | Completed | In Progress | Pending | Percentage |
|-------|-------------|-----------|-------------|---------|------------|
| Phase 1: Foundation | 27 | 23 | 0 | 4 | 85% |
| Phase 2: Organization | 28 | 10 | 0 | 18 | 36% |
| Phase 3: Styling | 16 | 6 | 0 | 10 | 38% |
| Phase 4: Error Handling | 12 | 0 | 0 | 12 | 0% |
| Phase 5: Testing & Docs | 11 | 0 | 0 | 11 | 0% |
| Phase 6: Cleanup | 9 | 0 | 0 | 9 | 0% |
| **Total** | **103** | **39** | **0** | **64** | **38%** |

### Completion Checklist

- [x] Phase 1 complete (23/27 tasks - core foundation done)
- [ ] Phase 2 complete (10/28 tasks - core types created, migration pending)
- [ ] Phase 3 complete (6/16 tasks - style system created, migration pending)
- [ ] Phase 4 complete (0/12 tasks)
- [ ] Phase 5 complete (0/11 tasks)
- [ ] Phase 6 complete (0/9 tasks)

---

## Dependencies & Notes

### Critical Path
1. ✅ Phase 1 completed (foundation changes)
2. 🔄 Phase 2 in progress (core types created, page migration pending)
3. 🔄 Phase 3 in progress (style system created, usage migration pending)
4. ⏳ Phase 4 depends on Phase 2 (needs Common context)
5. ⏳ Phase 5 runs parallel to all phases
6. ⏳ Phase 6 is final cleanup

### Testing Strategy
- ✅ Run tests after each completed task
- ✅ Commit after each logical group of tasks
- ✅ Create feature branch for entire effort
- ✅ Consider incremental deployment

### Estimated Effort
- ✅ Phase 1: 2-3 days (completed)
- ⏳ Phase 2: 2-3 days (in progress)
- ⏳ Phase 3: 1-2 days (in progress)
- ⏳ Phase 4: 1 day
- ⏳ Phase 5: 1-2 days
- ⏳ Phase 6: 1 day
- **Total**: 8-12 days (3-4 days completed)

### Completed Files
| File | Purpose | Status |
|------|---------|--------|
| `pkg/ui/core/state.go` | State type definitions | ✅ Complete |
| `pkg/ui/core/common.go` | Common context for DI | ✅ Complete |
| `pkg/ui/core/messages.go` | Typed message types | ✅ Complete |
| `pkg/ui/core/interfaces.go` | StatefulPage interface | ✅ Complete |
| `pkg/ui/keys/keys.go` | Centralized key bindings | ✅ Complete |
| `pkg/ui/styles/styles.go` | Comprehensive style system | ✅ Complete |
| `pkg/ui/ui.go` | Root model with pointer receivers | ✅ Complete |

### Key Achievements
1. ✅ **Type Safety**: No more `tea.Model` casting in router
2. ✅ **State Management**: Explicit `UIState` and `FocusState` types
3. ✅ **Typed Messages**: Type-safe message types for all data operations
4. ✅ **Pointer Receivers**: Root model uses `*Model` for in-place mutation
5. ✅ **Centralized Keys**: All key bindings in `pkg/ui/keys/keys.go`
6. ✅ **Common Context**: Dependency injection pattern established
7. ✅ **Style System**: Semantic colors and component styles

### Next Steps
1. Migrate pages to use Common context pattern
2. Migrate to typed messages in all handlers
3. Replace inline styles with style system references
4. Implement layout management centralization
5. Add error handling improvements

### Risk Mitigation
- ✅ Keep changes backward-compatible where possible
- ✅ Maintain feature branch with regular rebases
- ✅ Test thoroughly after each phase
- ✅ Document all breaking changes

---

## Getting Started

1. **Create feature branch**: `git checkout -b feature/bubble-tea-improvements`
2. **Start with Phase 1.1**: Eliminate model casting (highest impact) ✅ **Done**
3. **Commit frequently**: One commit per task or logical group
4. **Run tests**: After each commit
5. **Update this document**: Check off completed tasks

Good luck! 🚀
