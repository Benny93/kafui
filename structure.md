# Project Structure Analysis

## Overview
Kafui is a terminal-based UI application for Kafka management and monitoring written in Go. It uses a hybrid UI approach with **Bubble Tea** (Charm framework) as the primary TUI framework, while maintaining some **tview** components during migration. The application follows a modular, component-based architecture with clear separation of concerns.

## Project Structure

```
kafui/
├── main.go                          # Application entry point
├── cmd/kafui/                       # CLI command structure
│   ├── root.go                      # Root command implementation
│   └── root_test.go
├── pkg/
│   ├── api/                         # Core API interfaces
│   │   ├── api.go                   # KafkaDataSource interface definition
│   │   ├── api_test.go
│   │   └── errors.go                # Error types and handling
│   │
│   ├── datasource/                  # Data source implementations
│   │   ├── kafds/                   # Kafka data source implementation
│   │   │   ├── datasource_kaf.go    # Main Kafka datasource using kaf config
│   │   │   ├── datasource_kaf_test.go
│   │   │   ├── consume.go           # Message consumption logic
│   │   │   ├── consume_test.go
│   │   │   ├── consume_enhanced_test.go
│   │   │   ├── consume_interfaces.go # Consumer interfaces
│   │   │   ├── oauth.go             # OAuth2 authentication
│   │   │   ├── oauth_test.go
│   │   │   ├── scram_client.go      # SCRAM authentication client
│   │   │   ├── scram_client_test.go
│   │   │   ├── interfaces.go        # Internal interfaces
│   │   │   └── mocks.go             # Mock implementations for testing
│   │   │
│   │   └── mock/                    # Mock data source for testing
│   │       ├── kafka_data_source_mock.go
│   │       └── kafka_data_source_mock_test.go
│   │
│   ├── ui/                          # Bubble Tea UI layer (Modern)
│   │   ├── ui.go                    # Root UI controller
│   │   ├── ui_test.go
│   │   ├── kafui.go                 # UI initialization
│   │   ├── README.md                # Resource management documentation
│   │   ├── UI_ARCHITECTURE.md       # Comprehensive UI architecture docs
│   │   │
│   │   ├── core/                    # Core UI interfaces and utilities
│   │   │   ├── interfaces.go        # Page interface, Dimensions, Theme
│   │   │   ├── interfaces_test.go
│   │   │   ├── messages.go          # Centralized message types
│   │   │   ├── keys.go              # Key binding utilities
│   │   │   ├── keys_test.go
│   │   │   ├── help.go              # Help system
│   │   │   ├── help_test.go
│   │   │   ├── focus.go             # Focus management
│   │   │   ├── focus_test.go
│   │   │   ├── styles.go            # Shared styles
│   │   │   ├── utils.go             # Utility functions
│   │   │   └── utils_test.go
│   │   │
│   │   ├── components/              # Reusable UI components
│   │   │   ├── layout.go            # Responsive layout system
│   │   │   ├── sidebar.go           # Sidebar component
│   │   │   ├── footer.go            # Smart footer component
│   │   │   ├── header.go            # Header component
│   │   │   ├── main_content.go      # Main content area
│   │   │   ├── search_bar.go        # Advanced search bar
│   │   │   ├── json_content_view.go # JSON viewer with syntax highlighting
│   │   │   ├── json_content_view_test.go
│   │   │   ├── modal.go             # Modal dialogs
│   │   │   ├── styles.go            # Component styles
│   │   │   ├── fuzzy.go             # Fuzzy matching engine
│   │   │   ├── fuzzy_matching_test.go
│   │   │   ├── example_usage.go     # Usage examples
│   │   │   └── README.md
│   │   │
│   │   ├── pages/                   # Modular page implementations
│   │   │   │
│   │   │   ├── main/                # Main resource browser page
│   │   │   │   ├── main_page.go     # Core page logic
│   │   │   │   ├── main_page_test.go
│   │   │   │   ├── handlers.go      # Event handling
│   │   │   │   ├── keys.go          # Key bindings
│   │   │   │   ├── view.go          # View rendering
│   │   │   │   ├── types.go         # Data structures
│   │   │   │   ├── resource_manager.go # Resource management
│   │   │   │   ├── sidebar_sections.go # Sidebar configuration
│   │   │   │   ├── providers.go     # Data providers
│   │   │   │   ├── package.go       # Package documentation
│   │   │   │   └── README.md
│   │   │   │
│   │   │   ├── topic/               # Topic message consumption page
│   │   │   │   ├── topic_page.go    # Core topic page logic
│   │   │   │   ├── topic_page_test.go
│   │   │   │   ├── handlers.go      # Event and message handling
│   │   │   │   ├── keys.go          # Topic-specific key bindings
│   │   │   │   ├── view.go          # Topic view rendering
│   │   │   │   ├── types.go         # Topic-specific types
│   │   │   │   ├── consumption.go   # Real-time consumption controller
│   │   │   │   ├── search.go        # Message search
│   │   │   │   ├── search_test.go
│   │   │   │   ├── enter_key_test.go
│   │   │   │   └── package.go
│   │   │   │
│   │   │   ├── message_detail/      # Message detail view page
│   │   │   │   ├── message_detail_page.go
│   │   │   │   ├── message_detail_page_test.go
│   │   │   │   ├── message_detail_providers.go
│   │   │   │   ├── package.go
│   │   │   │   ├── json_test.go
│   │   │   │   └── viewport_test.go
│   │   │   │
│   │   │   └── resource_detail/     # Resource detail view page
│   │   │       ├── resource_detail_page.go
│   │   │       ├── resource_detail_page_test.go
│   │   │       ├── components.go
│   │   │       ├── package.go
│   │   │       └── types.go
│   │   │
│   │   ├── router/                  # Page routing and navigation
│   │   │   ├── router.go
│   │   │   └── router_test.go
│   │   │
│   │   ├── shared/                  # Shared utilities and types
│   │   │   ├── types.go             # Common types (ResourceItem, etc.)
│   │   │   ├── sorting.go           # Natural sorting utilities
│   │   │   ├── sorting_test.go
│   │   │   ├── highlight_test.go
│   │   │   └── debug.go             # Debug utilities
│   │   │
│   │   ├── template/                # UI templates and examples
│   │   │   └── ui/
│   │   │       ├── reusable_app.go  # Reusable app template
│   │   │       ├── components/      # Template components
│   │   │       │   ├── interfaces.go
│   │   │       │   ├── header.go
│   │   │       │   ├── sidebar.go
│   │   │       │   ├── content.go
│   │   │       │   └── footer.go
│   │   │       │   └── footer_test.go
│   │   │       ├── providers/       # Data providers
│   │   │       │   ├── interfaces.go
│   │   │       │   ├── default_providers.go
│   │   │       │   └── default_sections.go
│   │   │       └── styles/          # Theme and styling
│   │   │           ├── theme.go
│   │   │           └── utils.go
│   │   │
│   │   └── docs/                    # UI documentation
│   │       ├── UI_ARCHITECTURE.md
│   │       ├── advanced_bubble_tea_layouts.md
│   │       ├── bubble_tea_layout_guide.md
│   │       ├── bubbletea_readme.md
│   │       ├── IMPLEMENTATION_SUMMARY.md
│   │       ├── kafui_layout_patterns.md
│   │       ├── layout_design_summary.md
│   │       ├── layout_improvements.md
│   │       ├── navigation_and_key_input_improvement_plan.md
│   │       ├── navigation_example.md
│   │       ├── NAVIGATION_FIX_SUMMARY.md
│   │       ├── PHASE3_IMPLEMENTATION_SUMMARY.md
│   │       ├── RESOURCE_SWITCHING_IMPLEMENTATION.md
│   │       ├── technical_implementation_plan.md
│   │       ├── ui_improvements_plan.md
│   │       └── ui_modernization_summary.md
│   │
│   └── kafui/                       # Legacy tview UI layer (being phased out)
│       ├── kafui.go                 # Core application logic
│       ├── kafui_test.go
│       ├── ui.go                    # UI orchestration
│       ├── ui_test.go
│       ├── page_main.go             # Main page (tview)
│       ├── page_main_test.go
│       ├── page_topic.go            # Topic page (tview)
│       ├── page_topic_test.go
│       ├── page_detail.go           # Detail views (tview)
│       ├── page_detail_test.go
│       ├── search_bar.go            # Search functionality
│       ├── search_bar_test.go
│       ├── search_bar_advanced_test.go
│       ├── table_input.go           # Table input components
│       ├── table_input_test.go
│       ├── resource.go              # Resource abstraction
│       ├── resource_test.go
│       ├── resource_topic.go        # Topic resource
│       ├── resource_topic_test.go
│       ├── resource_group.go        # Consumer group resource
│       ├── resource_group_test.go
│       ├── resource_context.go      # Context resource
│       ├── resource_context_test.go
│       ├── helper.go                # Helper functions
│       ├── helper_test.go
│       ├── constants.go             # Constants
│       ├── constants_test.go
│       ├── integration_test.go
│       ├── benchmark_test.go
│       └── ui_workflow_test.go
│
└── test/                            # Integration and end-to-end tests
    ├── docker/                      # Docker test environment
    │   └── docker-compose.test.yml
    └── integration/                 # Integration tests
        └── e2e_test.go
```

## Core Components

### Application Layer
- **`main.go`** - Entry point that executes the root command
- **`cmd/kafui/root.go`** - Cobra command structure with flags for config and mock mode

### API Layer (`pkg/api/`)
- **`api.go`** - Defines `KafkaDataSource` interface
- **`errors.go`** - Custom error types for the application
- Provides abstraction between UI and data sources

### Data Source Layer (`pkg/datasource/`)
#### Kafka Implementation (`kafds/`)
- **`datasource_kaf.go`** - Main Kafka datasource using kaf configuration
- **`consume.go`** - Real-time message consumption with context management
- **`oauth.go`** - OAuth2 authentication support
- **`scram_client.go`** - SCRAM authentication client
- **`consume_interfaces.go`** - Consumer interfaces for extensibility

#### Mock Implementation (`mock/`)
- **`kafka_data_source_mock.go`** - Mock data source for testing without Kafka

### UI Layer (`pkg/ui/`) - Modern Bubble Tea Implementation

#### Core (`core/`)
- **`interfaces.go`** - Standardized `Page` interface that all pages implement
- **`messages.go`** - Centralized message types for event-driven architecture
- **`keys.go`** - Key binding utilities and help system
- **`focus.go`** - Focus management for interactive components
- **`styles.go`** - Shared lipgloss styles
- **`utils.go`** - Common utilities (sorting, truncation, formatting)

#### Components (`components/`)
Reusable UI building blocks:
- **`layout.go`** - Responsive layout system with header, sidebar, main content
- **`sidebar.go`** - Configurable sidebar with resource navigation
- **`footer.go`** - Smart footer with mode-aware rendering
- **`search_bar.go`** - Advanced search with fuzzy matching and multiple modes
- **`json_content_view.go`** - JSON viewer with syntax highlighting
- **`modal.go`** - Dialog system for confirmations
- **`fuzzy.go`** - Fuzzy matching engine
- **`styles.go`** - Component-specific styles

#### Pages (`pages/`) - Modular Page Architecture
Each page follows a consistent structure with separated concerns:

**Main Page (`main/`)**:
- Resource browser for Topics, Consumer Groups, Schemas, Contexts
- Resource switching with `:` command
- Search functionality with `/` command
- ResourceManager for data loading and caching

**Topic Page (`topic/`)**:
- Real-time message consumption from Kafka topics
- Pause/resume functionality
- Message filtering and search
- Connection status monitoring
- Error recovery with exponential backoff

**Message Detail Page (`message_detail/`)**:
- Detailed message view with multiple format options (raw, pretty, JSON, hex)
- Syntax-highlighted JSON display
- Scrollable viewport for large content
- Metadata display (headers, offset, partition)

**Resource Detail Page (`resource_detail/`)**:
- Detailed view of Kafka resources
- JSON formatting and display
- Resource-specific information panels

#### Router (`router/`)
- **`router.go`** - Page routing and navigation management
- Handles page transitions and state management

#### Template (`template/ui/`)
- **`reusable_app.go`** - Reusable application template
- **`components/`** - Template components for custom UIs
- **`providers/`** - Data providers and section configurations
- **`styles/`** - Theme definitions and utilities

### Legacy UI Layer (`pkg/kafui/`) - tview (Being Phased Out)
The legacy tview implementation is maintained during the migration to Bubble Tea:
- **`kafui.go`** - Core application logic
- **`ui.go`** - UI orchestration
- **`page_*.go`** - Various page implementations
- **`resource_*.go`** - Resource management implementations
- **`search_bar.go`** - Search functionality
- **`table_input.go`** - Table input components

## Testing Strategy

### Unit Tests
- Comprehensive unit tests alongside implementations (`*_test.go`)
- Component-level testing in `pkg/ui/components/`
- Page-level testing in `pkg/ui/pages/*/`

### Integration Tests
- **`test/integration/e2e_test.go`** - End-to-end tests
- **`test/docker/docker-compose.test.yml`** - Docker-based test environment
- **`scripts/run_integration_tests.sh`** - Integration test runner

### Test Coverage
- **`.testcoverage.yml`** - Coverage configuration
- **`coverage.svg`** - Coverage badge
- **`coverage.html`** - HTML coverage report

## Build and Development

### Build Configuration
- **`Makefile`** - Build, test, and development tasks
- **`Dockerfile`** - Container build definition
- **`.goreleaser.yaml`** - Release automation configuration
- **`go.mod`** / **`go.sum`** - Go module dependencies

### Scripts
- **`create_tag.sh`** - Git tag creation
- **`next_tag.sh`** - Next version tag calculation
- **`godownloader.sh`** - Installation script generator
- **`scripts/run_integration_tests.sh`** - Integration test execution

### CI/CD
- **`.github/`** - GitHub Actions workflows
- Automated testing and release processes

## Examples and Documentation

### Examples
- **`example-config.yaml`** - Sample configuration file
- **`example/dockercompose/`** - Docker Compose setup with Kafka
  - `docker-compose.yaml` - Kafka cluster configuration
  - `data/` - Sample data and configurations
  - `scripts/` - Helper scripts
- **`example/producer/`** - Example producer scripts
  - `create_topics.sh` - Topic creation script
  - `produce_data.sh` - Data production script

### Documentation
- **`README.md`** - Project overview and usage
- **`CRUSH.md`** - Development guidelines (if exists)
- **`AUTO_COMPLETION_IMPLEMENTATION.md`** - Auto-completion feature docs
- **`pkg/ui/docs/`** - Comprehensive UI documentation
  - Architecture documents
  - Implementation guides
  - Layout patterns
  - Migration summaries

## Dependencies

### Core Dependencies
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) - TUI framework
- **Bubbles** (`github.com/charmbracelet/bubbles`) - TUI components
- **Lipgloss** (`github.com/charmbracelet/lipgloss`) - Styling
- **tview** (`github.com/rivo/tview`) - Legacy TUI framework (during migration)
- **tcell** (`github.com/gdamore/tcell/v2`) - Terminal cell handling (for tview)
- **Sarama** (`github.com/IBM/sarama`) - Kafka client library
- **kaf** (`github.com/birdayz/kaf`) - Kafka CLI tool (config compatibility)
- **Cobra** (`github.com/spf13/cobra`) - CLI framework
- **colorjson** (`github.com/TylerBrock/colorjson`) - JSON coloring
- **prettyjson** (`github.com/hokaccha/go-prettyjson`) - JSON pretty printing
- **schema-registry** (`github.com/Landoop/schema-registry`) - Schema registry client
- **goavro** (`github.com/linkedin/goavro/v2`) - Avro encoding/decoding
- **protocompile** (`github.com/bufbuild/protocompile`) - Protocol Buffers compilation
- **protoreflect** (`github.com/jhump/protoreflect`) - Protocol Buffers reflection

### Testing Dependencies
- **testify** (`github.com/stretchr/testify`) - Testing toolkit
- **mockery** - Mock generation (via go:generate)

## Architecture Patterns

### 1. Model-View-Update (MVU)
The Bubble Tea UI follows the MVU pattern:
- **Model**: Application state
- **View**: Pure rendering functions
- **Update**: Event handling and state transitions

### 2. Component-Based Architecture
- Reusable UI components with functional options pattern
- Clear separation between layout, logic, and rendering
- Configurable components for different use cases

### 3. Interface-Based Design
- `KafkaDataSource` interface for data abstraction
- `Page` interface for consistent page implementations
- `Resource` and `ResourceItem` interfaces for resource management

### 4. Event-Driven Architecture
- Message passing via Bubble Tea's message system
- Command batching for efficient updates
- Channel-based communication for real-time data

### 5. Modular Page Structure
Each page module follows a consistent pattern:
- **Core logic** (`*_page.go`) - Main business logic
- **Handlers** (`handlers.go`) - Event processing
- **Keys** (`keys.go`) - Input handling
- **View** (`view.go`) - Rendering
- **Types** (`types.go`) - Data structures
- **Components** (`components.go`) - UI components (optional)

## Key Features

### Resource Management
- **Topics**: Browse, view details, consume messages
- **Consumer Groups**: Monitor state, view consumers
- **Schemas**: Schema Registry integration (extensible)
- **Contexts**: Multi-cluster context switching

### Message Consumption
- Real-time streaming from Kafka topics
- Pause/resume functionality
- Message filtering and search
- Multiple display formats (raw, JSON, hex)
- Error recovery with retry logic

### Search and Navigation
- Fuzzy matching for resource search
- Natural sorting for alphanumeric data
- Vim-style key bindings (j/k, g/G)
- Resource switching with `:` command
- Search mode with `/` command

### Authentication
- SASL/SCRAM authentication
- OAuth2 authentication
- SSL/TLS support
- Configuration via kaf config file

## Development Workflow

### Running Tests
```bash
# Unit tests
go test ./...

# Integration tests
./scripts/run_integration_tests.sh

# With coverage
go test -coverprofile=coverage.out ./...
```

### Building
```bash
# Development build
go build -o kafui

# Release build
goreleaser build

# Docker build
docker build -t kafui .
```

### Running
```bash
# Run with default config
go run main.go

# Run with custom config
go run main.go --config /path/to/config.yaml

# Run with mock data
go run main.go --mock
```

## Migration Notes

### tview to Bubble Tea Migration
The project is undergoing a migration from tview to Bubble Tea:
- **Legacy**: `pkg/kafui/` contains tview implementation
- **Modern**: `pkg/ui/` contains Bubble Tea implementation
- **Status**: Core pages implemented in Bubble Tea, legacy maintained for compatibility

### Migration Benefits
- Better composability with Bubble Tea's MVU pattern
- More maintainable modular architecture
- Improved testability with isolated components
- Modern styling with lipgloss
- Active community and ecosystem (Charm)

## Future Enhancements

### Planned Features
- Plugin system for custom resource types
- Advanced filtering with regex support
- Theme customization
- Export functionality
- Bulk operations
- Real-time resource monitoring
- Multi-cluster operations
- Audit logging

### Architecture Improvements
- Complete tview migration
- State persistence layer
- Performance optimizations for large datasets
- Enhanced error recovery mechanisms
- Improved accessibility
