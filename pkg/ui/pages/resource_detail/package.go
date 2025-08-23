// Package resource_detail contains the resource detail page components for the Kafui application.
// This package implements the resource detail page for viewing detailed
// information about Kafka resources (topics, consumer groups, schemas, contexts).
//
// The resource detail page is responsible for:
// - Displaying detailed resource information and configuration
// - Showing resource-specific metadata and properties
// - Providing resource configuration editing capabilities (if supported)
// - Handling resource operations (refresh, export, etc.)
// - Supporting resource-specific navigation and actions
//
// Architecture:
// - resource_detail_page.go: Core page model and business logic
// - keys.go: Key binding definitions and key handling logic
// - handlers.go: Event handling for different message types
// - view.go: View rendering and UI layout logic
//
// This modular structure separates concerns and improves maintainability
// while following the established UI patterns in the Kafui application.
package resource_detail