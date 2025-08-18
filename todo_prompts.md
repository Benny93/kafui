# Kafui Migration Todo Prompts

## 1. Topic Page Migration

### Current Status
The main page has been migrated to use Bubbletea, with a working topic list and search functionality. The next major component to migrate is the Topic Page.

### Prompt for Topic Page Migration
```
Continue the Kafui migration from tview to Bubbletea by implementing the Topic Page component. The Topic Page should be implemented in pkg/ui/page_topic.go.

Required Features:
1. Topic Details Section
   - Display topic name, partition count, and replication factor
   - Show configuration entries
   - Support for copying topic name

2. Message List Section
   - Display messages in a scrollable list
   - Show message key, offset, and timestamp
   - Support filtering/searching messages
   - Enable message navigation (up/down)

3. Controls/Actions
   - Implement message consumption controls (start/stop)
   - Add message format selection (JSON/Avro)
   - Support for tail/follow mode
   - Enable partition selection

4. Key Bindings
   - ESC to return to main page
   - / for message search
   - Space to pause/resume consumption
   - Enter to view message details

Follow the established Bubbletea patterns from the main page implementation, particularly:
- Use pointer receivers for model methods
- Implement proper state management
- Handle window resizing
- Provide clear status feedback

The original implementation can be found in pkg/kafui/page_topic.go for reference.
```

## 2. Message Detail Page Migration

### Prompt for Message Detail Page
```
Continue the Kafui migration from tview to Bubbletea by implementing the Message Detail Page component in pkg/ui/page_detail.go. This page shows the full content of a selected Kafka message.

Required Features:
1. Message Content Display
   - Show formatted message content (JSON/Avro)
   - Support for syntax highlighting
   - Enable content wrapping
   - Implement scrolling for long messages

2. Message Metadata Section
   - Display key, offset, partition
   - Show timestamp and headers
   - Present schema information if available

3. Navigation Features
   - Previous/Next message navigation
   - Quick return to message list
   - Copy message content

4. Key Bindings
   - ESC to return to topic page
   - j/k for scrolling
   - n/p for next/previous message
   - c to copy content
   - q to quit

Use Bubbletea's viewport component for scrolling long content and implement proper styling using lipgloss.
```

## 3. Search Bar Component Migration

### Prompt for Search Bar
```
Create a reusable search bar component in pkg/ui/components/search_bar.go that can be used across different pages and is used in the main page to filter resouces in the main table.

Required Features:
1. Core Functionality
   - Text input with placeholder
   - Search history
   - Advanced search options
   - Real-time filtering

2. Visual Elements
   - Search icon/indicator
   - Input field highlighting
   - Results count display
   - Error state handling

3. Integration Points
   - Support different search modes
   - Callback system for search results
   - Clear integration with list components

4. Key Bindings
   - / to activate search
   - ESC to clear/exit
   - Up/Down for history
   - Tab for completion

Follow Bubbletea's component patterns and ensure the component is reusable across different contexts.
```

## 4. Resource Group Management Migration

### Prompt for Resource Management
```
Implement the resource group management functionality in pkg/ui/resource_management.go.

Required Features:
1. Resource Types
   - Topic resources
   - Consumer group resources
   - Schema resources
   - Custom resource types


4. Integration
   - Data source connections
   - Error handling
   - Progress feedback
   - State persistence

The main page should allow the user to use the search bar to switch between resouces and the table of the mainpage should display the current resouce (for example list of topics for the topic resource)
Similar to the legacy implementation in pkg\kafui\resource_topic.go

Implement this as a modular system that can be integrated into different pages while maintaining the Bubbletea architecture.
```

## 5. Modal System Migration

### Prompt for Modal System
```
Create a reusable modal system in pkg/ui/components/modal.go for displaying dialogs, confirmations, and error messages.

Required Features:
1. Modal Types
   - Alert modals
   - Confirmation dialogs
   - Input prompts
   - Error messages

2. Visual Features
   - Centered overlay
   - Customizable styling
   - Animation support
   - Focus management

3. Integration
   - Easy creation API
   - Result handling
   - Stack management
   - Keyboard navigation

4. Accessibility
   - Keyboard controls
   - Clear focus indicators
   - Screen reader support

Follow Bubbletea's component patterns and ensure the modal system can be used consistently across the application.
```

## Implementation Order

1. Topic Page
   - This is the most critical component as it's needed for basic functionality
   - Builds on the existing main page implementation
   - Enables core Kafka interaction features

2. Message Detail Page
   - Natural extension of the Topic Page
   - Required for message inspection
   - Completes the basic message viewing workflow

3. Search Bar Component
   - Needed across multiple pages
   - Improves usability of both topic and message views
   - Can be implemented incrementally

4. Modal System
   - Required for user feedback and confirmations
   - Needed by other components for error handling
   - Improves overall UX

5. Resource Management
   - Can be implemented last as it's a more advanced feature
   - Builds on other components
   - Completes the full feature set

## Testing Strategy

For each component:
1. Write unit tests using Go's testing package
2. Add integration tests for component interactions
3. Test with mock data source
4. Verify keyboard navigation
5. Check error handling
6. Validate style rendering
7. Test window resizing

## Additional Notes

- Maintain consistent styling across components
- Follow the established patterns for state management
- Keep components loosely coupled
- Document key bindings and features
- Add proper error handling
- Consider accessibility throughout
- Maintain performance with large datasets
