// Package mainpage contains the main page components for the Kafui application.
// This package implements the main resource listing page with table view,
// search functionality, and resource management.
//
// The main page is responsible for:
// - Displaying a list of Kafka resources (topics, consumer groups, schemas, contexts)
// - Providing search and filtering capabilities
// - Handling resource navigation and selection
// - Managing resource type switching
//
// Architecture:
// - main_page.go: Core page model and business logic
// - keys.go: Key binding definitions and key handling logic
// - handlers.go: Event handling for different message types
// - view.go: View rendering and UI layout logic
// - resource_manager.go: Resource management and data loading
//
// This modular structure separates concerns and improves maintainability
// while following the established UI patterns in the Kafui application.
package mainpage
