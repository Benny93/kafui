# Resource Management System

This directory contains a comprehensive resource management system for the Kafui application, built using the Bubbletea TUI framework.

## Overview

The resource management system provides a modular, extensible interface for managing Kafka resources including topics, consumer groups, schemas, and custom resource types.

## Components

### Core Files

- **`resource_management.go`** - Main resource management model with tree view, details panel, and operations
- **`resource_tree.go`** - Hierarchical tree component for displaying resources
- **`resource_operations.go`** - Forms and handlers for resource CRUD operations
- **`resource_integration.go`** - Integration helpers and utilities
- **`resource_example.go`** - Usage examples and extension patterns
- **`resource_management_test.go`** - Comprehensive test suite

## Features

### 1. Resource Types
- **Topics** - Kafka topic management
- **Consumer Groups** - Consumer group monitoring and management
- **Schemas** - Schema registry integration (placeholder)
- **Custom Resources** - Extensible custom resource types

### 2. Resource Operations
- **Create** - Create new resources with validation
- **Delete** - Delete existing resources with confirmation
- **Update** - Modify resource configurations
- **View** - Display detailed resource information
- **Batch Operations** - Perform operations on multiple resources

### 3. User Interface
- **Resource Tree View** - Hierarchical display of resources by type
- **Details Panel** - Detailed information about selected resources
- **Action Menus** - Context-sensitive operation menus
- **Status Indicators** - Visual status feedback
- **Search Functionality** - Filter resources by name or description

### 4. Integration Features
- **Data Source Connections** - Pluggable data source interface
- **Error Handling** - Comprehensive error reporting and recovery
- **Progress Feedback** - Real-time operation progress
- **State Persistence** - Save and restore UI state

## Usage

### Basic Integration

```go
// Create a resource management page
dataSource := kafds.NewKafkaDataSourceKaf()
resourcePage := NewResourceManagementPage(dataSource)

// Initialize and run
cmd := resourcePage.Init()
// Handle in your main update loop
```

### Extending the Main UI

```go
// Extend the main model to include resource management
type ExtendedModel struct {
    Model // Embed existing model
    resourcePage *ResourceManagementPage
}

// Add resource management as a new page
const resourceManagementPage page = 3

// Handle in Update method
case resourceManagementPage:
    if em.resourcePage != nil {
        model, cmd := em.resourcePage.Update(msg)
        // Handle updates...
    }
```

### Custom Resource Types

```go
// Define custom resource type
const MyCustomResource ResourceType = 100

// Implement loading function
func LoadMyCustomResources(dataSource api.KafkaDataSource) []ResourceItem {
    // Load your custom resources
    return resources
}

// Extend the resource management model
func (m *ResourceManagementModel) loadCustomResources() tea.Msg {
    resources := LoadMyCustomResources(m.dataSource)
    return resourceLoadedMsg{
        resourceType: MyCustomResource,
        resources:    resources,
    }
}
```

## Key Bindings

### Navigation
- `↑/↓` or `j/k` - Navigate resource list
- `→/l` - Expand tree node
- `←/h` - Collapse tree node
- `enter` - View resource details
- `esc` - Go back/cancel

### Operations
- `c` - Create new resource
- `d` - Delete selected resource
- `u` - Update selected resource
- `r` - Refresh resources
- `b` - Batch operations
- `m` - Show action menu

### Search and Filtering
- `/` - Enter search mode
- `esc` - Exit search mode
- `enter` - Apply search filter

### General
- `ctrl+c` - Quit application
- `tab/shift+tab` - Navigate form fields (in operation forms)
- `ctrl+s` - Submit form (in operation forms)

## Architecture

### Model-View-Update Pattern

The system follows the Bubbletea MVU (Model-View-Update) pattern:

1. **Model** - Contains application state
2. **View** - Renders the current state
3. **Update** - Handles messages and updates state

### Component Hierarchy

```
ResourceManagementModel
├── ResourceTreeModel (tree view)
├── Table (details panel)
├── TextInput (search)
├── Spinner (loading indicator)
└── ResourceOperationModel (operation forms)
```

### Message Flow

```
User Input → Key Messages → Update Functions → State Changes → View Updates
```

## Styling

The system uses Lipgloss for styling with consistent color schemes:

- **Primary**: Color 205 (pink/magenta)
- **Success**: Color 46 (green)
- **Error**: Color 196 (red)
- **Warning**: Color 226 (yellow)
- **Secondary**: Color 240 (gray)

## Testing

Run the test suite:

```bash
go test ./pkg/ui/...
```

The test suite includes:
- Unit tests for all components
- Integration tests
- Benchmark tests for performance
- Mock data source testing

## Extension Points

### Adding New Resource Types

1. Define new ResourceType constant
2. Implement loading function
3. Add to resource management model
4. Implement operations (optional)

### Custom Operations

1. Define new ResourceOperation constant
2. Implement operation handler
3. Add to operation model
4. Update UI as needed

### Custom Data Sources

1. Implement api.KafkaDataSource interface
2. Add resource-specific methods
3. Handle in loading functions

## Performance Considerations

- Tree updates are optimized for large resource sets
- Lazy loading of resource details
- Efficient filtering and searching
- Minimal re-renders through proper state management

## Future Enhancements

- Real-time resource monitoring
- Advanced filtering and sorting
- Export/import functionality
- Resource templates
- Audit logging
- Multi-cluster support
- Plugin system for custom resources

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - UI components
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/Benny93/kafui/pkg/api` - Data source interface

## Contributing

When contributing to the resource management system:

1. Follow the existing MVU patterns
2. Add comprehensive tests
3. Update documentation
4. Ensure consistent styling
5. Consider performance implications
6. Maintain backward compatibility