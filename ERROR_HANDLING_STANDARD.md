# Error Handling Standard

**Date:** 2026-02-24  
**Status:** Approved  
**Version:** 1.0

---

## Executive Summary

This document defines the standard error handling patterns for the Kafui application. Consistent error handling improves:

- **Debuggability** - Clear error messages with context
- **User Experience** - Appropriate error display and recovery options
- **Maintainability** - Predictable error handling patterns
- **Testability** - Errors can be mocked and tested consistently

---

## Error Handling Architecture

### Error Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Error Occurs                              │
│  (Data Source / Business Logic / UI Operation)               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Wrap in Appropriate Error Type                  │
│  - UIError (typed, with context)                            │
│  - ErrorMsg (for Bubble Tea commands)                       │
│  - DataErrorMsg (for data layer errors)                     │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Send via tea.Cmd                                │
│  return func() tea.Msg { return ErrorMsg(err) }             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Handle in Update()                              │
│  - Set error state                                          │
│  - Display to user                                          │
│  - Attempt recovery (if applicable)                         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Display to User                                 │
│  - Status message (non-blocking)                            │
│  - Modal (requires acknowledgment)                          │
│  - Inline error (context-specific)                          │
└─────────────────────────────────────────────────────────────┘
```

---

## Standard Error Types

### 1. UIError (Base Error Type)

**Location:** `pkg/ui/shared/types.go`

**Purpose:** Typed errors with context for all UI-related errors.

```go
type UIError struct {
    Type    string  // Error category
    Message string  // User-friendly message
    Cause   error   // Original error (for debugging)
}

func (e UIError) Error() string {
    if e.Cause != nil {
        return e.Message + ": " + e.Cause.Error()
    }
    return e.Message
}

func NewUIError(errorType, message string, cause error) UIError {
    return UIError{
        Type:    errorType,
        Message: message,
        Cause:   cause,
    }
}
```

**Error Type Constants:**
```go
const (
    ErrorTypeDataLoad       = "data_load"        // Failed to load data
    ErrorTypeValidation     = "validation"       // Invalid input/state
    ErrorTypeNavigation     = "navigation"       // Navigation failed
    ErrorTypeRender         = "render"           // Rendering failed
    ErrorTypeConfiguration  = "configuration"    // Config error
    ErrorTypeConnection     = "connection"       // Connection lost
    ErrorTypeAuthentication = "authentication"   // Auth failed
    ErrorTypeTimeout        = "timeout"          // Operation timed out
)
```

**Usage Example:**
```go
// In data provider
func (p *ContentProvider) LoadData() error {
    data, err := p.dataSource.GetTopics()
    if err != nil {
        return shared.NewUIError(
            shared.ErrorTypeDataLoad,
            "Failed to load topics",
            err,
        )
    }
    // ...
}
```

### 2. ErrorMsg (Bubble Tea Message)

**Location:** Page-specific `types.go` files

**Purpose:** Wrap errors as Bubble Tea messages for async handling.

```go
// Standard pattern (defined in each page's types.go)
type ErrorMsg error

// Usage in command
func loadData() tea.Cmd {
    return func() tea.Msg {
        err := someOperation()
        if err != nil {
            return ErrorMsg(err)
        }
        return DataLoadedMsg{Data: data}
    }
}
```

**Handling in Update:**
```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ErrorMsg:
        m.error = error(msg)
        m.statusMessage = fmt.Sprintf("Error: %v", msg)
        return m, nil
    }
}
```

### 3. DataErrorMsg (Core Message)

**Location:** `pkg/ui/core/messages.go`

**Purpose:** Standardized error message for data layer errors.

```go
type DataErrorMsg struct {
    Type  string  // Data type that failed
    Error error   // The error
}

// Helper function
func NewDataErrorMsg(dataType string, err error) tea.Cmd {
    return func() tea.Msg {
        return DataErrorMsg{
            Type:  dataType,
            Error: err,
        }
    }
}
```

**Usage:**
```go
// In router or page
case core.DataErrorMsg:
    // Handle data error with type information
    m.errorMessage = fmt.Sprintf("Failed to load %s: %v", msg.Type, msg.Error)
```

---

## Error Display Patterns

### Pattern 1: Status Message (Non-blocking)

**Use Case:** Minor errors that don't block operation.

**Implementation:**
```go
// In Update method
case ErrorMsg:
    m.error = error(msg)
    m.statusMessage = fmt.Sprintf("Error: %v", msg)
    m.statusTime = time.Now()
    return m, nil

// In View method (footer or status area)
func (m *Model) View() string {
    if m.error != nil {
        return fmt.Sprintf("⚠️ %v", m.error)
    }
    // Normal view
}
```

**Example:**
```
┌─────────────────────────────────────┐
│  Topics                    [120]    │
│  ┌─────────────────────────────┐   │
│  │ topic-1                     │   │
│  │ topic-2                     │   │
│  └─────────────────────────────┘   │
│                                     │
│ ⚠️ Failed to refresh: timeout       │  ← Status message
└─────────────────────────────────────┘
```

### Pattern 2: Error Modal (Blocking)

**Use Case:** Critical errors requiring user acknowledgment.

**Implementation:**
```go
// Create error modal
func showErrorModal(err error) tea.Cmd {
    return func() tea.Msg {
        return ShowModalMsg{
            Title:   "Error",
            Message: fmt.Sprintf("An error occurred:\n%v", err),
            Type:    ErrorModal,
            Buttons: []string{"OK", "Retry"},
        }
    }
}

// Handle in Update
case ShowModalMsg:
    m.modal = NewModal(msg.Title, msg.Message, msg.Type)
    return m, m.modal.Init()
```

**Example:**
```
┌─────────────────────────────────────┐
│           ⚠️  Error                  │
│                                     │
│  Failed to connect to Kafka:        │
│  connection refused                 │
│                                     │
│        [  Retry  ]  [  OK  ]        │
└─────────────────────────────────────┘
```

### Pattern 3: Inline Error (Context-specific)

**Use Case:** Errors in specific UI components.

**Implementation:**
```go
// In content provider
func (p *ContentProvider) RenderContent(width, height int) string {
    if p.model.error != nil {
        return p.renderError()
    }
    // Normal content
}

func (p *ContentProvider) renderError() string {
    style := lipgloss.NewStyle().
        Foreground(lipgloss.Color("196")).
        Bold(true).
        Padding(1)
    
    return style.Render(fmt.Sprintf("⚠️ Error: %v", p.model.error))
}
```

**Example:**
```
┌─────────────────────────────────────┐
│  Topic: my-topic                    │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  ⚠️ Error: No messages      │   │  ← Inline error
│  │  available                  │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Pattern 4: Error with Recovery Options

**Use Case:** Errors where retry/recovery is possible.

**Implementation:**
```go
// Error state with recovery
type Model struct {
    error         error
    retryCount    int
    maxRetries    int
    canRetry      bool
}

// In Update
case ErrorMsg:
    m.error = error(msg)
    m.canRetry = m.retryCount < m.maxRetries
    return m, nil

// Retry command
func (m *Model) Retry() tea.Cmd {
    m.retryCount++
    return m.loadOperation()
}
```

**Example:**
```
┌─────────────────────────────────────┐
│  ⚠️ Connection failed               │
│                                     │
│  Retry attempt 2/3                  │
│  [R] Retry  [Q] Quit                │
└─────────────────────────────────────┘
```

---

## Error Handling by Layer

### Data Source Layer

**Pattern:** Return typed errors with context.

```go
// In datasource
func (ds *KafkaDataSource) GetTopics() (map[string]api.Topic, error) {
    topics, err := ds.client.ListTopics()
    if err != nil {
        return nil, shared.NewUIError(
            shared.ErrorTypeConnection,
            "Failed to connect to Kafka broker",
            err,
        )
    }
    return topics, nil
}

// Handle authentication errors
func (ds *KafkaDataSource) Init(cfgOption string) error {
    err := ds.authenticate()
    if err != nil {
        if isAuthError(err) {
            return shared.NewUIError(
                shared.ErrorTypeAuthentication,
                "Authentication failed. Check credentials.",
                err,
            )
        }
        return err
    }
    return nil
}
```

### Business Logic Layer

**Pattern:** Wrap data source errors, add business context.

```go
// In business model
func (m *TopicModel) LoadTopicDetails() error {
    topics, err := m.dataSource.GetTopics()
    if err != nil {
        // Wrap with business context
        return shared.NewUIError(
            shared.ErrorTypeDataLoad,
            fmt.Sprintf("Failed to load details for topic '%s'", m.topicName),
            err,
        )
    }
    
    topic, exists := topics[m.topicName]
    if !exists {
        return shared.NewUIError(
            shared.ErrorTypeValidation,
            fmt.Sprintf("Topic '%s' not found", m.topicName),
            nil,
        )
    }
    
    m.topicDetails = topic
    return nil
}
```

### Provider Layer

**Pattern:** Convert errors to commands for async handling.

```go
// In content provider
func (p *ContentProvider) HandleContentUpdate(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "r" {
            return p.loadWithRetry()
        }
    }
    return nil
}

func (p *ContentProvider) loadWithRetry() tea.Cmd {
    return func() tea.Msg {
        err := p.model.LoadData()
        if err != nil {
            // Return as ErrorMsg for Update to handle
            return ErrorMsg(err)
        }
        return DataLoadedMsg{Data: p.model.GetData()}
    }
}
```

### Page Model Layer

**Pattern:** Handle errors, update state, display to user.

```go
// In page model Update
func (m *PageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ErrorMsg:
        return m.handleError(error(msg))
        
    case core.DataErrorMsg:
        return m.handleDataError(msg.Type, msg.Error)
    }
    // ...
}

func (m *PageModel) handleError(err error) (tea.Model, tea.Cmd) {
    // Store error
    m.error = err
    
    // Determine error severity
    if uiErr, ok := err.(shared.UIError); ok {
        switch uiErr.Type {
        case shared.ErrorTypeAuthentication:
            // Critical - show modal
            return m, showErrorModal(err)
        case shared.ErrorTypeTimeout:
            // Recoverable - show status, allow retry
            m.statusMessage = "Request timed out. Press 'r' to retry."
            m.canRetry = true
        default:
            // Standard - show status
            m.statusMessage = fmt.Sprintf("Error: %v", err)
        }
    }
    
    return m, nil
}
```

---

## Retry and Recovery Patterns

### Pattern 1: Simple Retry

```go
type RetryPolicy struct {
    MaxRetries    int
    RetryDelay    time.Duration
}

func (m *Model) RetryWithPolicy(operation func() error) tea.Cmd {
    if m.retryCount >= m.policy.MaxRetries {
        return func() tea.Msg {
            return ErrorMsg(fmt.Errorf("max retries exceeded"))
        }
    }
    
    m.retryCount++
    
    return tea.Tick(m.policy.RetryDelay, func(t time.Time) tea.Msg {
        err := operation()
        if err != nil {
            return ErrorMsg(err)
        }
        return nil
    })
}
```

### Pattern 2: Exponential Backoff

```go
type RetryPolicy struct {
    MaxRetries        int
    InitialDelay      time.Duration
    MaxDelay          time.Duration
    BackoffFactor     float64
}

func (p *RetryPolicy) CalculateDelay(attempt int) time.Duration {
    delay := p.InitialDelay
    for i := 1; i < attempt; i++ {
        delay = time.Duration(float64(delay) * p.BackoffFactor)
    }
    
    if delay > p.MaxDelay {
        delay = p.MaxDelay
    }
    
    return delay
}
```

### Pattern 3: Circuit Breaker

```go
type CircuitBreaker struct {
    state          string  // "closed", "open", "half-open"
    failureCount   int
    failureThreshold int
    resetTimeout   time.Duration
    lastFailure    time.Time
}

func (cb *CircuitBreaker) CanExecute() bool {
    switch cb.state {
    case "closed":
        return true
    case "open":
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = "half-open"
            return true
        }
        return false
    case "half-open":
        return true
    }
    return false
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.failureCount = 0
    cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.failureCount++
    cb.lastFailure = time.Now()
    
    if cb.failureCount >= cb.failureThreshold {
        cb.state = "open"
    }
}
```

---

## Error Context and Logging

### Error Context

**Purpose:** Provide debugging context with errors.

```go
type ErrorContext struct {
    Operation string                 // What was being done
    Resource  string                 // Which resource
    Params    map[string]interface{} // Operation parameters
    Timestamp time.Time              // When it occurred
    UserID    string                 // Who triggered it (if applicable)
}

func (c *ErrorContext) String() string {
    return fmt.Sprintf(
        "Operation: %s, Resource: %s, Time: %s",
        c.Operation, c.Resource, c.Timestamp.Format(time.RFC3339),
    )
}

// Wrap error with context
func wrapWithContext(err error, ctx *ErrorContext) error {
    return fmt.Errorf("%w\nContext: %s", err, ctx)
}
```

### Debug Logging (Development Only)

**Purpose:** Log errors for debugging (not in production).

```go
// In shared/debug.go (development only)
func LogError(operation string, err error, context map[string]interface{}) {
    if !isDebugMode() {
        return
    }
    
    log.Printf(
        "[ERROR] %s: %v\nContext: %+v",
        operation, err, context,
    )
}

// Usage
func (p *Provider) LoadData() error {
    err := p.load()
    if err != nil {
        shared.LogError(
            "LoadData",
            err,
            map[string]interface{}{
                "resourceType": p.resourceType,
                "dataSource":   p.dataSourceName,
            },
        )
        return err
    }
    return nil
}
```

---

## Testing Error Handling

### Unit Tests for Error Cases

```go
func TestContentProvider_HandleError(t *testing.T) {
    model := &Model{}
    provider := NewContentProvider(model)
    
    // Create error
    testErr := shared.NewUIError(
        shared.ErrorTypeDataLoad,
        "test error",
        errors.New("cause"),
    )
    
    // Handle error
    cmd := provider.HandleError(testErr)
    
    // Verify command returns ErrorMsg
    msg := cmd()
    assert.IsType(t, ErrorMsg(""), msg)
    
    // Verify error message
    assert.Equal(t, "test error: cause", string(msg.(ErrorMsg)))
}
```

### Integration Tests for Error Flow

```go
func TestErrorFlow_FromDataToUI(t *testing.T) {
    // Setup
    mockDS := &MockDataSource{
        GetTopicsFunc: func() (map[string]api.Topic, error) {
            return nil, errors.New("connection failed")
        },
    }
    
    model := NewMainPageModel(mockDS)
    
    // Trigger load
    cmd := model.Init()
    
    // Process commands until error
    for cmd != nil {
        msg := cmd()
        if err, ok := msg.(ErrorMsg); ok {
            // Verify error reached UI
            assert.Contains(t, err.Error(), "connection failed")
            break
        }
        // Handle other messages...
    }
}
```

### Mock Error Scenarios

```go
// Mock data source with configurable errors
type MockDataSource struct {
    GetTopicsFunc    func() (map[string]api.Topic, error)
    GetMessagesFunc  func() ([]api.Message, error)
    ShouldFail       bool
    FailureType      string  // "auth", "timeout", "connection"
}

func (m *MockDataSource) GetTopics() (map[string]api.Topic, error) {
    if m.ShouldFail {
        switch m.FailureType {
        case "auth":
            return nil, shared.NewUIError(
                shared.ErrorTypeAuthentication,
                "Authentication failed",
                errors.New("invalid credentials"),
            )
        case "timeout":
            return nil, shared.NewUIError(
                shared.ErrorTypeTimeout,
                "Request timed out",
                errors.New("context deadline exceeded"),
            )
        default:
            return nil, errors.New("unknown error")
        }
    }
    return m.GetTopicsFunc()
}
```

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Silent Errors

```go
// BAD: Error ignored
func loadData() {
    data, err := ds.GetTopics()
    _ = data  // Error silently ignored
}

// GOOD: Error handled
func loadData() tea.Cmd {
    return func() tea.Msg {
        data, err := ds.GetTopics()
        if err != nil {
            return ErrorMsg(err)
        }
        return DataLoadedMsg{Data: data}
    }
}
```

### ❌ Anti-Pattern 2: Generic Error Messages

```go
// BAD: Unhelpful error message
if err != nil {
    return ErrorMsg(errors.New("error occurred"))
}

// GOOD: Specific error message
if err != nil {
    return ErrorMsg(shared.NewUIError(
        shared.ErrorTypeDataLoad,
        fmt.Sprintf("Failed to load topics from %s", brokerURL),
        err,
    ))
}
```

### ❌ Anti-Pattern 3: Error Spam

```go
// BAD: Error shown repeatedly on every render
func (m *Model) View() string {
    if m.error != nil {
        log.Println(m.error)  // Logs on every render!
        return fmt.Sprintf("Error: %v", m.error)
    }
}

// GOOD: Error logged once, shown appropriately
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case ErrorMsg:
        shared.LogError("Update", error(msg), nil)  // Log once
        m.error = error(msg)
        m.errorShown = false  // Track if shown
}
```

### ❌ Anti-Pattern 4: Panic on Error

```go
// BAD: Panic crashes the application
if err != nil {
    panic(err)
}

// GOOD: Graceful error handling
if err != nil {
    return ErrorMsg(err)
}
```

### ❌ Anti-Pattern 5: Mixing Error Types

```go
// BAD: Inconsistent error handling
func operation1() error {
    return errors.New("plain error")
}

func operation2() error {
    return shared.UIError{...}  // Different type!
}

// GOOD: Consistent error types
func operation1() error {
    return shared.NewUIError(
        shared.ErrorTypeOperation,
        "operation failed",
        errors.New("cause"),
    )
}
```

---

## Error Handling Checklist

When implementing error handling:

- [ ] **Error is typed** - Use `UIError` with appropriate type
- [ ] **Error has context** - Include operation, resource, parameters
- [ ] **Error is wrapped** - Preserve original error as `Cause`
- [ ] **Error is sent via command** - Use `ErrorMsg` for async handling
- [ ] **Error is displayed** - User sees appropriate message
- [ ] **Recovery is offered** - Retry option if applicable
- [ ] **Error is logged** - Debug logging for development
- [ ] **Error is tested** - Unit test for error case
- [ ] **Error message is clear** - User-friendly, actionable
- [ ] **Error doesn't crash** - Graceful degradation

---

## Quick Reference

### Error Type Selection

| Scenario | Error Type | Display |
|----------|-----------|---------|
| Data load failure | `ErrorTypeDataLoad` | Status + Retry |
| Invalid input | `ErrorTypeValidation` | Inline error |
| Connection lost | `ErrorTypeConnection` | Modal + Retry |
| Auth failed | `ErrorTypeAuthentication` | Modal + Config |
| Timeout | `ErrorTypeTimeout` | Status + Retry |
| Config error | `ErrorTypeConfiguration` | Modal + Quit |

### Error Handling Template

```go
// 1. Define error in types.go
type ErrorMsg error

// 2. Create error in command
func loadData() tea.Cmd {
    return func() tea.Msg {
        data, err := ds.GetData()
        if err != nil {
            return ErrorMsg(shared.NewUIError(
                shared.ErrorTypeDataLoad,
                "Failed to load data",
                err,
            ))
        }
        return DataLoadedMsg{Data: data}
    }
}

// 3. Handle error in Update
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case ErrorMsg:
        m.error = error(msg)
        m.statusMessage = fmt.Sprintf("Error: %v", msg)
        return m, nil
}

// 4. Display error in View
func (m *Model) View() string {
    if m.error != nil {
        return fmt.Sprintf("⚠️ %v", m.error)
    }
    // Normal view
}
```

---

## Implementation Priority

### Immediate (Week 1)
- [ ] Document current error handling patterns ✅
- [ ] Standardize on `UIError` type
- [ ] Add error type constants
- [ ] Update topic page error handling

### Short Term (Week 2-3)
- [ ] Implement retry patterns consistently
- [ ] Add error context to all data operations
- [ ] Standardize error display components
- [ ] Add error handling tests

### Medium Term (Month 1)
- [ ] Implement circuit breaker for connections
- [ ] Add error analytics (optional)
- [ ] Create error handling utilities
- [ ] Document error scenarios

---

**Related Documents:**
- `PAGE_ARCHITECTURE_STANDARD.md` - Page architecture patterns
- `UI_ARCHITECTURE_REVIEW.md` - Architecture review
- `MISSING_FEATURES_NEW_UI.md` - Feature roadmap
