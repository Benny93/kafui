package core

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Page navigation messages
type (
	PageChangeMsg struct {
		PageID string
		Data   interface{}
	}

	BackMsg struct{}
	QuitMsg struct{}
)

// Data messages
type (
	DataLoadedMsg struct {
		Type string
		Data interface{}
	}

	DataErrorMsg struct {
		Type  string
		Error error
	}

	RefreshDataMsg struct {
		Type string
	}

	LoadingMsg struct {
		Type    string
		Loading bool
	}
)

// UI messages
type (
	WindowSizeMsg struct {
		Width  int
		Height int
	}

	StatusMsg struct {
		Message string
		Type    StatusType
	}

	TimerTickMsg struct {
		Time time.Time
		ID   string
	}

	DimensionsUpdateMsg struct {
		Width  int
		Height int
	}
)

// Search messages
type (
	SearchMsg struct {
		Query string
		Mode  SearchMode
	}

	ClearSearchMsg struct{}

	FilterAppliedMsg struct {
		Count int
		Query string
	}

	SearchModeChangeMsg struct {
		Mode SearchMode
	}
)

// Resource messages
type (
	ResourceChangeMsg struct {
		ResourceType string
		Data         interface{}
	}

	ResourceSelectedMsg struct {
		ResourceID   string
		ResourceType string
		Item         interface{}
	}

	ResourceLoadMsg struct {
		ResourceType string
	}
)

// Topic-specific messages
type (
	TopicSelectedMsg struct {
		TopicName string
		Topic     interface{}
	}

	MessageConsumedMsg struct {
		Message interface{}
	}

	ConsumptionStartedMsg struct {
		TopicName string
	}

	ConsumptionStoppedMsg struct {
		TopicName string
	}

	ConsumptionErrorMsg struct {
		TopicName string
		Error     error
	}
)

// Detail page messages
type (
	DetailPageOpenMsg struct {
		ResourceID   string
		ResourceType string
		Data         interface{}
	}

	DetailPageCloseMsg struct{}
)

// Common message creation functions
func NewDataLoadedMsg(dataType string, data interface{}) tea.Cmd {
	return func() tea.Msg {
		return DataLoadedMsg{
			Type: dataType,
			Data: data,
		}
	}
}

func NewDataErrorMsg(dataType string, err error) tea.Cmd {
	return func() tea.Msg {
		return DataErrorMsg{
			Type:  dataType,
			Error: err,
		}
	}
}

func NewStatusMsg(message string, statusType StatusType) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{
			Message: message,
			Type:    statusType,
		}
	}
}

func NewPageChangeMsg(pageID string, data interface{}) tea.Cmd {
	return func() tea.Msg {
		return PageChangeMsg{
			PageID: pageID,
			Data:   data,
		}
	}
}

func NewResourceSelectedMsg(resourceID, resourceType string, item interface{}) tea.Cmd {
	return func() tea.Msg {
		return ResourceSelectedMsg{
			ResourceID:   resourceID,
			ResourceType: resourceType,
			Item:         item,
		}
	}
}