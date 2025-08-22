# Bubble Tea Development Guide - Condensed Context

Bubble Tea is a Go TUI framework using the Elm Architecture pattern. Key development principles:
Core Architecture
Event Loop: Messages → Update() → View() → repeat
Models: Use value receivers by default (functional pattern)
Commands: Offload expensive operations to tea.Cmd to keep event loop fast
Performance Best Practices
Keep Update() and View() fast - offload heavy work to commands
Message ordering: User input is sequential, but commands execute concurrently
Avoid race conditions - make model changes only in Update(), not in goroutines
Development Workflow
Debug: Dump messages to file using spew.Fdump() when DEBUG env var set
Live reload: Use file watchers to rebuild/restart on code changes
Pointer vs value receivers: Value receivers maintain functional purity, pointer receivers useful for helper methods
Key Patterns
// Fast update pattern
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle message, return new state + command
    return m, expensiveOperationCmd()
}

// Debug message dumping
if m.dump != nil {
    spew.Fdump(m.dump, msg)
}
Common Pitfalls
Don't modify model state outside Update()
Don't block the event loop with expensive operations
Messages from commands may arrive out of order