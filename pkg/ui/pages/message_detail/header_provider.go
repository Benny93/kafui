package messagedetail

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// MessageDetailHeaderDataProvider implements the HeaderDataProvider interface for message detail
type MessageDetailHeaderDataProvider struct {
	model *Model
}

// NewMessageDetailHeaderDataProvider creates a new header data provider for message detail
func NewMessageDetailHeaderDataProvider(model *Model) *MessageDetailHeaderDataProvider {
	return &MessageDetailHeaderDataProvider{
		model: model,
	}
}

// GetBrandName returns the brand name
func (m *MessageDetailHeaderDataProvider) GetBrandName() string {
	return "Kafui™"
}

// GetAppName returns the application name
func (m *MessageDetailHeaderDataProvider) GetAppName() string {
	return "Message Detail"
}

// GetStatusData returns status information for the header
func (m *MessageDetailHeaderDataProvider) GetStatusData() map[string]interface{} {
	status := make(map[string]interface{})

	if m.model != nil {
		status["topic"] = m.model.topicName
		status["partition"] = fmt.Sprintf("P%d", m.model.message.Partition)
		status["offset"] = fmt.Sprintf("O%d", m.model.message.Offset)
		status["format"] = m.model.displayFormat.ValueFormat

		// Add current time as a fallback since message timestamp may not be available
		status["time"] = time.Now().Format("15:04:05")
	}

	return status
}

// HandleHeaderUpdate handles header updates
func (m *MessageDetailHeaderDataProvider) HandleHeaderUpdate(msg tea.Msg) tea.Cmd {
	return nil
}

// InitHeader initializes the header provider
func (m *MessageDetailHeaderDataProvider) InitHeader() tea.Cmd {
	return nil
}
