// Package messagedetail contains the message detail page components for the Kafui application.
// This package implements the message detail page for viewing individual
// Kafka messages with formatted content and metadata.
//
// The message detail page is responsible for:
// - Displaying individual message content with syntax highlighting
// - Showing message metadata (headers, timestamp, offset, partition)
// - Providing content formatting for different data types (JSON, Avro, etc.)
// - Handling message navigation (previous/next)
// - Supporting content search within the message
//
// Architecture:
// - detail_page.go: Core page model and business logic
// - keys.go: Key binding definitions and key handling logic
// - handlers.go: Event handling for different message types
// - view.go: View rendering and UI layout logic
//
// This modular structure separates concerns and improves maintainability
// while following the established UI patterns in the Kafui application.
package messagedetail