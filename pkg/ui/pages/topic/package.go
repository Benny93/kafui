// Package topic contains the topic page components for the Kafui application.
// This package implements the topic detail page with message consumption,
// search functionality, and topic-specific operations.
//
// The topic page is responsible for:
// - Displaying topic details and configuration
// - Consuming and displaying Kafka messages in real-time
// - Providing message search and filtering capabilities
// - Handling message consumption control (start/stop/pause)
// - Managing consumption error handling and retry logic
//
// Architecture:
// - topic_page.go: Core page model and business logic
// - keys.go: Key binding definitions and key handling logic
// - handlers.go: Event handling for different message types
// - view.go: View rendering and UI layout logic
// - consumption.go: Message consumption logic and control
//
// This modular structure separates concerns and improves maintainability
// while following the established UI patterns in the Kafui application.
package topic