# Kafui Architecture

## System Overview

Kafui is a terminal-based Kafka management tool built on the **Bubble Tea** framework, following the **Model-View-Update (MVU)** architectural pattern. The system is designed with clear separation of concerns across three primary layers: **Data Source**, **API Abstraction**, and **UI Presentation**.

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI Entry Point                         │
│                    (cmd/kafui/root.go)                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    UI Initialization Layer                   │
│                      (pkg/ui/kafui.go)                       │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                           │
        ▼                                           ▼
┌──────────────────┐                    ┌──────────────────────┐
│   Router Layer   │                    │   Core Services      │
│  (pkg/ui/router) │                    │  (pkg/ui/core)       │
│                  │                    │  - Messages          │
│  - Page routing  │                    │  - Focus management  │
│  - Navigation    │                    │  - Key bindings      │
│  - State mgmt    │                    │  - Shared styles     │
└──────────────────┘                    └──────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│                     Page Layer (pkg/ui/pages)                │
│  ┌────────────┐  ┌──────────┐  ┌─────────────┐  ┌─────────┐ │
│  │ Main Page  │  │Topic Page│  │Message Detail│  │Resource │ │
│  │            │  │          │  │             │  │ Detail  │ │
│  │ - Browse   │  │ - Stream │  │ - View      │  │ - Info  │ │
│  │ - Switch   │  │ - Filter │  │ - Formats   │  │ - JSON  │ │
│  │ - Search   │  │ - Pause  │  │ - Metadata  │  │         │ │
│  └────────────┘  └──────────┘  └─────────────┘  └─────────┘ │
└─────────────────────────────────────────────────────────────┘
        │
        ├──────────────────────────────────────────┐
        │                                          │
        ▼                                          ▼
┌──────────────────┐                    ┌──────────────────────┐
│ Template System  │                    │  Reusable Components │
│ (pkg/ui/template)│                    │ (pkg/ui/components)  │
│                  │                    │                      │
│ - Header         │                    │ - Layout             │
│ - Sidebar        │                    │ - SearchBar          │
│ - Content        │                    │ - Footer             │
│ - Footer         │                    │ - Modal              │
│ - Providers      │                    │ - JSONContentView    │
└──────────────────┘                    └──────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│                   API Abstraction Layer                      │
│                     (pkg/api/api.go)                         │
│                                                              │
│  interface KafkaDataSource {                                 │
│    GetTopics()                                               │
│    GetConsumerGroups()                                       │
│    ConsumeMessages()                                         │
│    GetSchemas()                                              │
│  }                                                           │
└─────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│                  Data Source Layer                           │
│  ┌─────────────────────┐      ┌──────────────────────┐      │
│  │   Kafka (kafds)     │      │   Mock (mock)        │      │
│  │                     │      │                      │      │
│  │ - Sarama client     │      │ - Test data          │      │
│  │ - OAuth2/SCRAM auth │      │ - Deterministic      │      │
│  │ - Schema registry   │      │ - No dependencies    │      │
│  │ - Avro/Proto decode │      │                      │      │
│  └─────────────────────┘      └──────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Architectural Layers

### 1. CLI Entry Point (`cmd/kafui/`)

**Responsibility**: Bootstrap the application using Cobra CLI framework.

**Key Design Decisions**:
- Uses **Cobra** for command structure and flag parsing
- Supports `--config` for custom configuration paths
- Supports `--mock` flag for testing without Kafka connection
- Delegates to UI initialization after parsing arguments

**Flow**:
```
main.go → cmd/kafui/root.go → ui.Init() → Bubble Tea runtime
```

### 2. UI Initialization Layer (`pkg/ui/`)

**Responsibility**: Initialize and start the Bubble Tea application.

**Components**:
- **`ui.go`**: Root model implementing Bubble Tea's `Model` interface
- **`kafui.go`**: UI factory and initialization logic

**Initialization Sequence**:
1. Create data source (Kafka or Mock)
2. Initialize router with page registry
3. Create root model with theme and dimensions
4. Start Bubble Tea program with `tea.NewProgram()`

### 3. Core Services (`pkg/ui/core/`)

**Responsibility**: Provide shared infrastructure for the UI layer.

**Modules**:

| Module | Purpose |
|--------|---------|
| `interfaces.go` | Defines `Page` interface, `Dimensions`, `Theme` types |
| `messages.go` | Centralized message types for event communication |
| `keys.go` | Key binding definitions and handlers |
| `focus.go` | Focus management for interactive components |
| `help.go` | Help system with key binding documentation |
| `styles.go` | Shared lipgloss styles and themes |
| `utils.go` | Common utilities (sorting, truncation, formatting) |

**Design Pattern**: Dependency injection via interfaces. Pages receive core services rather than importing them directly, enabling testability.

### 4. Router Layer (`pkg/ui/router/`)

**Responsibility**: Manage page navigation and state transitions.

**Key Features**:
- **Page Registry**: Maps page IDs to page constructors
- **Navigation Stack**: Supports back/forward navigation
- **State Preservation**: Maintains page state during switches
- **Type-Safe Transitions**: Uses typed page identifiers

**Navigation Flow**:
```
User Action → Page Handler → Router.NavigateTo() → Router.Update() → New Page Render
```

### 5. Page Layer (`pkg/ui/pages/`)

**Responsibility**: Implement specific feature screens using the MVU pattern.

**Page Structure** (consistent across all pages):

```
┌─────────────────────────────────────────┐
│  Page Model                             │
│  ├── state (enum)                       │
│  ├── data (page-specific)               │
│  ├── components (search, table, etc.)   │
│  └── router (reference for navigation)  │
├─────────────────────────────────────────┤
│  Init()    → Initialize page state      │
│  Update()  → Handle messages/events     │
│  View()    → Render page UI             │
└─────────────────────────────────────────┘
```

**Page Implementations**:

| Page | Purpose | Key Features |
|------|---------|--------------|
| **Main** | Resource browser | Topic/Consumer Group/Schema listing, resource switching |
| **Topic** | Message consumption | Real-time streaming, pause/resume, filtering |
| **MessageDetail** | Message inspection | Multiple formats (raw/JSON/hex), syntax highlighting |
| **ResourceDetail** | Resource info | JSON view, metadata display |

**Message Handling Pattern**:
```go
// Example: Topic page message handling
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    case MessageBatchMsg:
        return m.handleMessages(msg)
    case ErrorMsg:
        return m.handleError(msg)
    case StatusMsg:
        return m.handleStatus(msg)
    default:
        return m, nil
    }
}
```

### 6. Template System (`pkg/ui/template/`)

**Responsibility**: Provide reusable UI structure and composition.

**Architecture**:
```
ReusableApp
├── Header (title, status)
├── Sidebar (navigation, sections)
├── Content (page-specific)
└── Footer (help, mode indicators)
```

**Provider Pattern**:
- Pages implement `DataProvider` interface
- Template calls providers for dynamic content
- Decouples layout from data logic

**Benefits**:
- Consistent UI across all pages
- Single source of truth for layout
- Easy theme customization
- Reduced code duplication

### 7. Component Library (`pkg/ui/components/`)

**Responsibility**: Reusable UI building blocks.

**Component Catalog**:

| Component | Purpose | Interactive |
|-----------|---------|-------------|
| `Layout` | Responsive container | No |
| `Sidebar` | Navigation panel | Yes (selection) |
| `Footer` | Status and help | No |
| `Header` | Title and branding | No |
| `SearchBar` | Fuzzy search input | Yes (text input) |
| `JSONContentView` | Syntax-highlighted JSON | Yes (scrolling) |
| `Modal` | Dialog overlays | Yes (confirmation) |
| `MainContent` | Content area wrapper | No |

**Component Interface**:
```go
type Component interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

**Design Principles**:
- **Composability**: Components can contain other components
- **Functional Options**: Configuration via option functions
- **Encapsulation**: Internal state management
- **Testability**: Isolated component testing

### 8. API Abstraction (`pkg/api/`)

**Responsibility**: Define contracts between UI and data sources.

**Core Interface**:
```go
type KafkaDataSource interface {
    // Resource enumeration
    GetTopics(ctx context.Context) ([]string, error)
    GetConsumerGroups(ctx context.Context) ([]string, error)
    GetBrokers(ctx context.Context) ([]int32, error)
    
    // Message operations
    ConsumeMessages(ctx context.Context, topic string, cfg ConsumeConfig) (<-chan Message, error)
    
    // Schema operations (extensible)
    GetSchemas(ctx context.Context) ([]string, error)
}
```

**Benefits**:
- UI layer is data-source agnostic
- Easy testing with mock implementations
- Future extensibility (e.g., REST API source)

### 9. Data Source Layer (`pkg/datasource/`)

**Responsibility**: Implement data access logic for Kafka.

#### Kafka Implementation (`kafds/`)

**Architecture**:
```
┌─────────────────────────────────────────┐
│  KafkaDataSource (datasource_kaf.go)    │
│  ├── Sarama client configuration        │
│  ├── Authentication (OAuth2, SCRAM)     │
│  ├── Topic/Consumer Group queries       │
│  └── Message consumption orchestration  │
├─────────────────────────────────────────┤
│  Consumer (consume.go)                  │
│  ├── Context-based cancellation         │
│  ├── Channel-based message streaming    │
│  ├── Error recovery                     │
│  └── Pause/Resume control               │
├─────────────────────────────────────────┤
│  Authentication                         │
│  ├── oauth.go (OAuth2 token mgmt)       │
│  └── scram_client.go (SCRAM auth)       │
└─────────────────────────────────────────┘
```

**Key Design Decisions**:

1. **Channel-Based Streaming**: Messages flow through Go channels for real-time updates
2. **Context Cancellation**: Clean resource cleanup on navigation or errors
3. **Config Compatibility**: Uses kaf config format for familiarity
4. **Schema Registry Integration**: Decoded Avro/Protobuf messages
5. **Error Recovery**: Automatic retry with exponential backoff

#### Mock Implementation (`mock/`)

**Purpose**: Enable testing without Kafka cluster.

**Features**:
- Deterministic test data
- Configurable response delays
- Error simulation
- No external dependencies

### 10. Shared Utilities (`pkg/ui/shared/`)

**Responsibility**: Cross-cutting utilities and common types.

**Modules**:
- **`types.go`**: Common types (`ResourceItem`, `ViewDimensions`, `PageState`)
- **`sorting.go`**: Natural sorting for alphanumeric data
- **`debug.go`**: Debug logging utilities

## Data Flow Patterns

### 1. Resource Listing Flow

```
User opens app
       │
       ▼
┌──────────────┐
│  Main Page   │
│  Init()      │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Router loads │
│ Main Page    │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Main calls   │
│ data.GetTopics()
└──────────────┘
       │
       ▼
┌──────────────┐
│ KafkaDataSource
│ queries broker
└──────────────┘
       │
       ▼
┌──────────────┐
│ Topics loaded│
│ → TopicsLoadedMsg
└──────────────┘
       │
       ▼
┌──────────────┐
│ Main.Update()│
│ updates state
└──────────────┘
       │
       ▼
┌──────────────┐
│ Main.View()  │
│ renders table
└──────────────┘
```

### 2. Message Consumption Flow

```
User selects topic
       │
       ▼
┌──────────────┐
│ Topic Page   │
│ created      │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Consume()    │
│ starts goroutine
└──────────────┘
       │
       ▼
┌──────────────┐
│ Sarama       │
│ consumer loop
└──────────────┘
       │
       ▼
┌──────────────┐
│ Messages     │
│ sent via channel
└──────────────┘
       │
       ▼
┌──────────────┐
│ MessageBatchMsg
│ → Topic Page
└──────────────┘
       │
       ▼
┌──────────────┐
│ Batch added  │
│ to state     │
└──────────────┘
       │
       ▼
┌──────────────┐
│ View renders │
│ new messages │
└──────────────┘
```

### 3. Page Navigation Flow

```
User presses Enter on topic
       │
       ▼
┌──────────────┐
│ Topic Page   │
│ handles key  │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Creates      │
│ TopicPageMsg │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Router       │
│ receives msg │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Router pushes│
│ current page │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Router creates
│ new page     │
└──────────────┘
       │
       ▼
┌──────────────┐
│ New page     │
│ becomes active
└──────────────┘
```

## Concurrency Model

### Goroutine Usage

| Location | Purpose | Synchronization |
|----------|---------|-----------------|
| `kafds.Consume()` | Message consumption loop | Channel → Bubble Tea Cmd |
| `kafds.GetTopics()` | Async topic listing | Callback via message |
| `topic/consumption.go` | Real-time message streaming | Channel batching |
| UI timers | Periodic refresh | `tea.Tick()` |

### Message Passing

Bubble Tea's message system provides safe concurrency:

```go
// Data source sends messages via command
func (d *KafkaDataSource) Consume(topic string) tea.Cmd {
    return func() tea.Msg {
        // Runs in goroutine
        messages := d.consumeLoop()
        return MessageBatchMsg{Messages: messages}
    }
}

// UI receives messages in Update()
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Runs in main thread - no mutex needed
    switch msg := msg.(type) {
    case MessageBatchMsg:
        m.messages = append(m.messages, msg.Messages...)
    }
}
```

**Key Principle**: All state mutations happen in `Update()`, which runs serially in the main thread.

## Error Handling Strategy

### Error Types

```go
// pkg/api/errors.go
type ErrorType int

const (
    ConnectionError ErrorType = iota
    AuthenticationError
    NotFoundError
    TimeoutError
    UnknownError
)

type KafkaError struct {
    Type    ErrorType
    Message string
    Cause   error
}
```

### Error Propagation

```
Data Source Error
       │
       ▼
┌──────────────┐
│ Wrapped as   │
│ KafkaError   │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Sent as      │
│ ErrorMsg     │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Page handles │
│ based on type
└──────────────┘
       │
       ▼
┌──────────────┐
│ User feedback│
│ (modal, status)
└──────────────┘
```

### Recovery Patterns

1. **Connection Errors**: Retry with exponential backoff
2. **Authentication Errors**: Prompt for credentials
3. **Timeout Errors**: Offer retry option
4. **Not Found Errors**: Navigate back automatically

## Testing Architecture

### Test Pyramid

```
         ┌───┐
        │ E2E │ (5 tests - integration/)
       └─────┘
      ┌───────┐
     │Integration│ (datasource tests with Docker)
    └─────────┘
   ┌───────────┐
  │ Unit Tests   │ (comprehensive - all packages)
 └─────────────┘
```

### Test Strategies

**Unit Tests**:
- Mock data source for UI testing
- Table-driven tests for utilities
- Component isolation testing

**Integration Tests**:
- Docker Compose Kafka cluster
- Real Kafka operations
- End-to-end user flows

**Mock Implementation**:
```go
// pkg/datasource/mock/kafka_data_source_mock.go
type MockKafkaDataSource struct {
    TopicsFunc       func(ctx context.Context) ([]string, error)
    ConsumeFunc      func(ctx context.Context, topic string, cfg ConsumeConfig) (<-chan Message, error)
    // ... other methods
}
```

### Test Coverage

- Target: >80% coverage
- Coverage tracking via `.testcoverage.yml`
- Visual reports in `coverage.html`
- Badge in README via `coverage.svg`

## Configuration System

### Config File Format

Uses kaf-compatible YAML format:

```yaml
clusters:
  - name: local
    brokers:
      - localhost:9092
  - name: prod
    brokers:
      - prod-broker-1:9092
      - prod-broker-2:9092
    sasl:
      mechanism: scram-sha-512
      username: user
      password: pass

currentCluster: local
```

### Configuration Loading

```
cmd/kafui/root.go
       │
       ▼
┌──────────────┐
│ --config flag│
│ (optional)   │
└──────────────┘
       │
       ▼
┌──────────────┐
│ Default:     │
│ $HOME/.kaf/config
└──────────────┘
       │
       ▼
┌──────────────┐
│ Parsed by    │
│ kafds layer  │
└──────────────┘
```

## Extension Points

### Adding New Resource Types

1. Implement `KafkaDataSource` methods for new resource
2. Add resource to `Main Page` ResourceManager
3. Create detail page (optional)
4. Register in router

### Adding New Pages

1. Create page directory under `pkg/ui/pages/`
2. Implement `Page` interface (Init, Update, View)
3. Register in router
4. Add navigation trigger

### Custom Authentication

1. Implement Sarama `SASLMechanism`
2. Add to `kafds` authentication factory
3. Configure via YAML

### Theme Customization

1. Modify `pkg/ui/core/styles.go`
2. Or use template theme system
3. Override lipgloss styles

## Performance Considerations

### Memory Management

- **Message Buffering**: Topic page limits in-memory messages
- **Lazy Loading**: Resources loaded on-demand
- **Garbage Collection**: Context cancellation releases resources

### Rendering Optimization

- **Viewport Scrolling**: Only renders visible portion
- **Component Caching**: Expensive computations cached
- **Batch Updates**: Multiple messages batched before render

### Concurrency Limits

- **Consumer Goroutines**: One per topic consumption
- **Connection Pooling**: Shared Sarama client
- **Channel Buffering**: Prevents blocking on slow consumers

## Security Model

### Authentication Methods

| Method | Implementation | Configuration |
|--------|---------------|---------------|
| None | Plain connection | No config needed |
| SASL/SCRAM | `scram_client.go` | `sasl.mechanism` |
| OAuth2 | `oauth.go` | `oauth.*` settings |
| SSL/TLS | Sarama TLS config | `tls.*` settings |

### Credential Handling

- Credentials loaded from config file
- No credential storage in application
- Environment variable support (via kaf config)

### Network Security

- TLS encryption for broker connections
- Certificate validation
- Secure schema registry connections

## Deployment Architecture

### Build Pipeline

```
Source Code
     │
     ▼
┌─────────┐
│ go build │
└─────────┘
     │
     ▼
┌─────────┐
│ Goreleaser│ (cross-platform binaries)
└─────────┘
     │
     ▼
┌─────────┐
│ Docker   │ (container image)
└─────────┘
```

### Distribution

- **Binaries**: GitHub Releases (Linux, macOS, Windows)
- **Package Managers**: Homebrew, AUR (future)
- **Containers**: Docker Hub
- **Source**: Go modules

### Runtime Requirements

- Go 1.21+ (for building)
- Kafka broker 2.0+ (for runtime)
- Terminal with true color support

## Design Principles

### 1. Separation of Concerns

Each layer has a single responsibility:
- CLI: Argument parsing
- UI: Presentation and interaction
- API: Abstraction
- DataSource: Data access

### 2. Interface-Based Design

Dependencies inverted through interfaces:
- UI depends on `KafkaDataSource` interface, not implementation
- Pages depend on `Page` interface for navigation
- Components depend on abstractions, not concretions

### 3. Functional Core, Imperative Shell

- Business logic is pure functions where possible
- Side effects isolated at boundaries
- Easy to test core logic

### 4. Convention Over Configuration

- Consistent page structure
- Standard component interfaces
- Shared styles and themes

### 5. Progressive Enhancement

- Works without Schema Registry
- Works in mock mode
- Graceful degradation on errors

## Comparison: MVU vs MVC

Kafui uses **MVU** (Model-View-Update) instead of traditional MVC:

| Aspect | MVU (Kafui) | MVC |
|--------|-------------|-----|
| State | Immutable updates | Mutable state |
| Flow | Unidirectional | Bidirectional |
| Testing | Pure functions | Mock dependencies |
| Concurrency | Message passing | Shared state + locks |

**Why MVU**:
- Predictable state transitions
- Easy debugging (message trace)
- Natural fit for Bubble Tea
- No race conditions in UI

## Future Architecture Considerations

### Scalability

- Plugin system for custom resources
- WebSocket for real-time multi-cluster updates
- State persistence (save/restore sessions)

### Modularity

- Feature flags for enterprise features
- Modular builds (exclude unused resources)
- Dynamic theme loading

### Extensibility

- REST API for remote control
- Scripting support (Lua/JavaScript)
- Custom view definitions
