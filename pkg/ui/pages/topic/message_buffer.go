package topic

import (
	"fmt"
	"sort"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// sortMessages sorts messages by offset ascending (required for pagination logic)
func (m *Model) sortMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sort messages by offset ascending
	sort.Slice(m.messages, func(i, j int) bool {
		if m.messages[i].Offset != m.messages[j].Offset {
			return m.messages[i].Offset < m.messages[j].Offset
		}
		return m.messages[i].Partition < m.messages[j].Partition
	})

	// Also sort filtered messages
	if len(m.filteredMessages) > 0 && len(m.filteredMessages) != len(m.messages) {
		sort.Slice(m.filteredMessages, func(i, j int) bool {
			if m.filteredMessages[i].Offset != m.filteredMessages[j].Offset {
				return m.filteredMessages[i].Offset < m.filteredMessages[j].Offset
			}
			return m.filteredMessages[i].Partition < m.filteredMessages[j].Partition
		})
	} else {
		// Make sure filteredMessages is updated to the sorted messages
		m.filteredMessages = m.messages
	}
}

// Business logic methods for the original Model

// FilterMessages recomputes the filtered view from the current search input or
// active smart filter (MSG-23/24), then refreshes pagination and the table.
func (m *Model) FilterMessages() {
	m.applyFilter()
	m.pagination.SetTotalMessages(len(m.filteredMessages))
	m.pendingReset = true
	m.updateMessageTable()
	m.markRenderDirty()
}

// addMessageInternal adds a message without triggering view update (for background consumption)
func (m *Model) addMessageInternal(msg api.Message) {
	key := fmt.Sprintf("%d-%d", msg.Partition, msg.Offset)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Only add if not already consumed
	if _, exists := m.consumedMessages[key]; !exists {
		m.consumedMessages[key] = msg

		// Enforce message buffer limit (FIFO)
		if len(m.messages) >= m.maxMessages {
			oldMsg := m.messages[0]
			oldKey := fmt.Sprintf("%d-%d", oldMsg.Partition, oldMsg.Offset)
			delete(m.consumedMessages, oldKey)
			m.messages = m.messages[1:]
		}

		// Add new message
		m.messages = append(m.messages, msg)

		// Update filtered messages but don't trigger render
		m.filteredMessages = m.messages
		m.pagination.SetTotalMessages(len(m.filteredMessages))
	}
}

// AddMessage adds a new message and triggers view update (for manual refresh)
func (m *Model) AddMessage(msg api.Message) {
	m.addMessageInternal(msg)
	m.mu.Lock()
	m.statusMessage = fmt.Sprintf("Consumed %d messages", len(m.messages))
	m.renderVersion++
	m.mu.Unlock()
}

// shouldUpdate checks if enough time has passed since last update (throttling)
func (m *Model) shouldUpdate() bool {
	if m.updateThrottle <= 0 {
		return true
	}
	return time.Since(m.lastUpdateTime) >= m.updateThrottle
}

// maxPendingWhilePaused caps the messages buffered while paused.
const maxPendingWhilePaused = MaxMessageBuffer

// TogglePause freezes or unfreezes the message table. While paused, live
// messages keep being drained from the stream but are buffered instead of
// shown; resuming merges them in.
func (m *Model) TogglePause() {
	m.paused = !m.paused
	if m.paused {
		m.statusMessage = "Consumption paused"
		m.markRenderDirty()
		return
	}
	pending := len(m.pendingWhilePaused)
	for _, msg := range m.pendingWhilePaused {
		m.addMessageInternal(msg)
	}
	m.pendingWhilePaused = nil
	if pending > 0 {
		m.sortMessages()
		m.updateMessageTable()
		m.pagination.SetTotalMessages(len(m.filteredMessages))
	}
	m.markRenderDirty()
	m.statusMessage = fmt.Sprintf("Consumption resumed (%d new messages)", pending)
}

// bufferWhilePaused holds a live message until the table is resumed.
func (m *Model) bufferWhilePaused(msg api.Message) {
	if len(m.pendingWhilePaused) >= maxPendingWhilePaused {
		m.pendingWhilePaused = m.pendingWhilePaused[1:]
	}
	m.pendingWhilePaused = append(m.pendingWhilePaused, msg)
	m.statusMessage = fmt.Sprintf("Consumption paused (%d new messages)", len(m.pendingWhilePaused))
	m.markRenderDirty() // the table stays frozen, but the paused indicator and count update
}

// SetError records err as the current and last error.
func (m *Model) SetError(err error) {
	m.error = err
	m.lastError = err

	m.connectionStatus = "failed"
	m.statusMessage = fmt.Sprintf("Error: %v", err)
}

// SetConnectionStatus updates the connection status
func (m *Model) SetConnectionStatus(status string) {
	m.connectionStatus = status
	switch status {
	case "connected":
		m.statusMessage = "Connected and consuming messages"
	case "connecting":
		m.statusMessage = "Connecting to topic..."
	case "disconnected":
		m.statusMessage = "Disconnected"
	case "failed":
		m.statusMessage = "Connection failed"
	}
}

// Utility functions
