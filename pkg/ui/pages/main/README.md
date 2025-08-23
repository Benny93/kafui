# Main Page Package

This package contains the modular implementation of the main page for the Kafui application. The main page is responsible for displaying and managing Kafka resources (topics, consumer groups, schemas, contexts) in a table view with search and filtering capabilities.

## Architecture

The main page follows a modular architecture with separated concerns:

### Core Components

- **main_page.go**: Contains the main page model (`Model`) and core business logic
- **handlers.go**: Event handling logic for different message types
- **keys.go**: Key binding definitions and key handling logic  
- **view.go**: View rendering and UI layout logic
- **resource_manager.go**: Resource management and data loading abstraction
- **types.go**: Type definitions and message types
- **main_page_test.go**: Unit tests for the main page components

### Design Patterns

1. **Separation of Concerns**: Each file has a specific responsibility
   - Model: State management and business logic
   - Handlers: Event processing
   - Keys: Input handling
   - View: Rendering
   - ResourceManager: Data access abstraction

2. **Dependency Injection**: Components receive their dependencies through constructors

3. **Interface-Based Design**: Implements the `core.Page` interface for consistency

4. **Event-Driven Architecture**: Uses Bubble Tea's message-passing system

## Key Features

### Resource Management
- Supports multiple resource types: Topics, Consumer Groups, Schemas, Contexts
- Dynamic resource switching via `:` command
- Automatic data refresh with configurable intervals

### Search and Filtering
- Real-time search with `/` command
- Natural sorting for alphanumeric data
- Search result highlighting
- Persistent filter state during data updates

### Navigation
- Vim-style key bindings (j/k, g/G)
- Table-based navigation
- Enter to select and navigate to detail views
- ESC for back navigation and search cancellation

### UI Components
- Integrated layout with sidebar, main content, and footer
- Responsive design that adapts to terminal dimensions
- Status messages and loading indicators
- Spinner animations during data loading

## Usage

### Creating a Main Page

```go
import "github.com/Benny93/kafui/pkg/ui/pages/main"

// Create a new main page with a data source
model := main.NewModel(dataSource)

// Initialize the page
cmd := model.Init()

// Use in Bubble Tea application
app := tea.NewProgram(model)
```

### Implementing the Page Interface

The main page implements the `core.Page` interface:

```go
type Page interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
    SetDimensions(width, height int)
    GetID() string
}
```

## Message Types

### Input Messages
- `SearchTopicsMsg`: Triggered when user performs a search
- `ClearSearchMsg`: Triggered when user clears the search
- `SwitchResourceMsg`: Triggered when switching between resource types
- `SwitchResourceByNameMsg`: Triggered when switching by resource name

### Data Messages
- `TopicListMsg`: Contains loaded topic data
- `CurrentResourceListMsg`: Contains loaded resource data
- `ErrorMsg`: Contains error information
- `TimerTickMsg`: Periodic refresh trigger

## Key Bindings

| Key | Action |
|-----|--------|
| `/` | Enter search mode |
| `:` | Enter resource switching mode |
| `j` or `↓` | Move down |
| `k` or `↑` | Move up |
| `g` or `Home` | Go to top |
| `G` or `End` | Go to bottom |
| `Enter` | Select item |
| `Esc` | Back/Cancel |
| `q` or `Ctrl+C` | Quit |

## Resource Types

### Topics
- Display topic name, partitions, replication factor, message count
- Navigate to topic detail page for message consumption

### Consumer Groups  
- Display group name, state, consumer count
- Navigate to consumer group detail page

### Schemas
- Display schema subject, version, ID, type
- Navigate to schema detail page

### Contexts
- Display context name and current status
- Support context switching

## Testing

The package includes comprehensive unit tests covering:

- Model initialization and state management
- Resource manager functionality
- Event handling
- Key binding logic
- View rendering
- Interface compliance

Run tests with:
```bash
go test ./pkg/ui/pages/main/
```

## Dependencies

- `github.com/Benny93/kafui/pkg/api`: Kafka data source interface
- `github.com/Benny93/kafui/pkg/ui/components`: Reusable UI components
- `github.com/Benny93/kafui/pkg/ui/core`: Core UI interfaces and utilities
- `github.com/Benny93/kafui/pkg/ui/shared`: Shared utilities and types
- `github.com/charmbracelet/bubbletea`: Terminal UI framework
- `github.com/charmbracelet/bubbles`: UI components (table, spinner)
- `github.com/charmbracelet/lipgloss`: Styling and layout

## Future Enhancements

- Support for custom resource types via plugin system
- Advanced filtering with regex and fuzzy matching
- Keyboard shortcuts customization
- Theme support
- Export functionality for resource data
- Bulk operations on selected resources