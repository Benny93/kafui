# UI Components

This directory contains reusable UI components extracted from `page_main.go` to make the layout system modular and reusable across different pages.

## Components Overview

### 1. Layout Component (`layout.go`)
The main layout manager that handles:
- **Header rendering** with configurable title and resource type indicators
- **Responsive layout calculations** for content and sidebar dimensions
- **Content combination** with proper spacing and margins
- **Complete layout rendering** that combines header, body, and footer

**Key Features:**
- Configurable sidebar width and visibility
- Automatic dimension calculations
- Consistent spacing and margins
- Responsive design support

### 2. Sidebar Component (`sidebar.go`)
A flexible sidebar that can display:
- **Context information** (current Kafka context)
- **Resource navigation buttons** (Topics, Consumer Groups, Schemas, Contexts)
- **Keyboard shortcuts** with customizable lists
- **Custom sections** for page-specific content

**Key Features:**
- Configurable sections (show/hide resources, shortcuts, etc.)
- Current resource highlighting
- Custom content sections
- Consistent styling

### 3. Footer Component (`footer.go`)
A smart footer that adapts to different modes:
- **Normal mode**: Shows selection info, item counts, status, and last update
- **Search mode**: Shows search instructions
- **Responsive text**: Automatically truncates content for narrow screens

**Key Features:**
- Mode-aware rendering (normal vs search)
- Responsive text handling
- Status and timing information
- Spinner integration

### 4. Main Content Component (`main_content.go`)
Manages the main content area:
- **Search bar integration** (optional)
- **List/table display** with proper sizing
- **Responsive dimensions** that adapt to available space

**Key Features:**
- Optional search bar
- Automatic height calculations
- Flexible content rendering

### 5. Styles Component (`styles.go`)
Centralized styling definitions:
- **Color schemes** (adaptive light/dark)
- **Border styles** and layouts
- **Text styles** (headers, subtitles, info text)
- **Component-specific styles**

## Usage Examples

### Basic Page Layout
```go
// Initialize components
layout := components.NewLayout(components.LayoutConfig{
    Width:        120,
    Height:       40,
    SidebarWidth: 35,
    ShowSidebar:  true,
    HeaderTitle:  "My Page Title",
    ResourceType: "TOPICS",
})

sidebar := components.NewSidebar(components.SidebarConfig{
    Context:       "kafka-dev",
    ShowResources: true,
    ShowShortcuts: true,
})

footer := components.NewFooter(components.FooterConfig{
    Width:         120,
    StatusMessage: "Ready",
    LastUpdate:    time.Now(),
})

// Render complete layout
return layout.RenderComplete(
    mainContent,
    sidebar.Render(),
    footer.Render(),
)
```

### Custom Sidebar Sections
```go
sidebar := components.NewSidebar(components.SidebarConfig{
    Context:       "my-context",
    ShowResources: false,
    ShowShortcuts: true,
    CustomSections: []components.SidebarSection{
        {
            Title:   "Custom Info",
            Content: "Additional page-specific content",
        },
    },
})
```

### Responsive Footer
```go
footer := components.NewFooter(components.FooterConfig{
    Width:         width,
    SearchMode:    isSearching,
    SelectedItem:  selectedItemName,
    TotalItems:    itemCount,
    StatusMessage: statusText,
    LastUpdate:    lastUpdateTime,
    Spinner:       spinnerModel,
})
```

## Benefits of This Refactoring

### 1. **Reusability**
- Components can be used across different pages
- Consistent look and feel throughout the application
- Reduced code duplication

### 2. **Maintainability**
- Centralized styling and layout logic
- Easy to update UI elements globally
- Clear separation of concerns

### 3. **Testability**
- Components can be tested in isolation
- Easier to verify rendering behavior
- Reduced complexity in page-level tests

### 4. **Flexibility**
- Configurable components adapt to different use cases
- Optional sections and features
- Easy to extend with new functionality

### 5. **Consistency**
- Standardized spacing, colors, and layouts
- Unified component behavior
- Professional appearance

## Migration Guide

### Before (Original page_main.go)
```go
// Large monolithic View() method with embedded layout logic
func (m MainPageModel) View() string {
    // 60+ lines of layout calculations and rendering
    // Embedded sidebar, header, and footer logic
    // Hard to reuse or test individual components
}
```

### After (Refactored with Components)
```go
// Clean, component-based View() method
func (m MainPageModel) View() string {
    // Update component configurations
    m.layout.UpdateConfig(layoutConfig)
    m.sidebar.UpdateConfig(sidebarConfig)
    m.footer.UpdateConfig(footerConfig)
    
    // Render using components
    return m.layout.RenderComplete(
        m.mainContent.Render(),
        m.sidebar.Render(),
        m.footer.Render(),
    )
}
```

## Testing

The refactored components maintain full test coverage:
- **Layout tests**: Verify proper dimension calculations and rendering
- **Sidebar tests**: Check resource buttons and shortcuts display
- **Footer tests**: Validate different modes and responsive behavior
- **Integration tests**: Ensure components work together correctly

All tests use the `fmt.Println(docStyle.Render(doc.String()))` pattern to display beautifully rendered output during testing.

## Future Enhancements

The component system is designed to be easily extensible:
- Add new sidebar sections
- Create specialized footer modes
- Implement theme switching
- Add animation support
- Create component variants for different page types

This refactoring provides a solid foundation for building consistent, maintainable, and reusable UI components throughout the Kafui application.