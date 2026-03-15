package core

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
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

	BreadcrumbUpdateMsg struct {
		Items []string
	}
)

// Typed data messages - replacing generic DataLoadedMsg and DataErrorMsg
type (
	// Topics loaded messages
	TopicsLoadedMsg struct {
		Topics map[string]api.Topic
	}

	TopicsLoadErrorMsg struct {
		Error error
	}

	// Consumer groups loaded messages
	ConsumerGroupsLoadedMsg struct {
		Groups []api.ConsumerGroup
	}

	ConsumerGroupsLoadErrorMsg struct {
		Error error
	}

	// Messages consumed
	MessagesConsumedMsg struct {
		Messages []api.Message
	}

	MessageConsumeErrorMsg struct {
		Error error
	}

	// Schema loaded messages
	SchemasLoadedMsg struct {
		Schemas []api.SchemaInfo
	}

	SchemasLoadErrorMsg struct {
		Error error
	}

	// Contexts loaded messages
	ContextsLoadedMsg struct {
		Contexts []string
	}

	ContextsLoadErrorMsg struct {
		Error error
	}

	// Generic loading messages (for backward compatibility during migration)
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

// Typed message helper functions

// NewTopicsLoadedMsg creates a command that sends a TopicsLoadedMsg
func NewTopicsLoadedMsg(topics map[string]api.Topic) tea.Cmd {
	return func() tea.Msg {
		return TopicsLoadedMsg{Topics: topics}
	}
}

// NewTopicsLoadErrorMsg creates a command that sends a TopicsLoadErrorMsg
func NewTopicsLoadErrorMsg(err error) tea.Cmd {
	return func() tea.Msg {
		return TopicsLoadErrorMsg{Error: err}
	}
}

// NewConsumerGroupsLoadedMsg creates a command that sends a ConsumerGroupsLoadedMsg
func NewConsumerGroupsLoadedMsg(groups []api.ConsumerGroup) tea.Cmd {
	return func() tea.Msg {
		return ConsumerGroupsLoadedMsg{Groups: groups}
	}
}

// NewConsumerGroupsLoadErrorMsg creates a command that sends a ConsumerGroupsLoadErrorMsg
func NewConsumerGroupsLoadErrorMsg(err error) tea.Cmd {
	return func() tea.Msg {
		return ConsumerGroupsLoadErrorMsg{Error: err}
	}
}

// NewMessagesConsumedMsg creates a command that sends a MessagesConsumedMsg
func NewMessagesConsumedMsg(messages []api.Message) tea.Cmd {
	return func() tea.Msg {
		return MessagesConsumedMsg{Messages: messages}
	}
}

// NewMessageConsumeErrorMsg creates a command that sends a MessageConsumeErrorMsg
func NewMessageConsumeErrorMsg(err error) tea.Cmd {
	return func() tea.Msg {
		return MessageConsumeErrorMsg{Error: err}
	}
}

// NewSchemasLoadedMsg creates a command that sends a SchemasLoadedMsg
func NewSchemasLoadedMsg(schemas []api.SchemaInfo) tea.Cmd {
	return func() tea.Msg {
		return SchemasLoadedMsg{Schemas: schemas}
	}
}

// NewSchemasLoadErrorMsg creates a command that sends a SchemasLoadErrorMsg
func NewSchemasLoadErrorMsg(err error) tea.Cmd {
	return func() tea.Msg {
		return SchemasLoadErrorMsg{Error: err}
	}
}

// NewContextsLoadedMsg creates a command that sends a ContextsLoadedMsg
func NewContextsLoadedMsg(contexts []string) tea.Cmd {
	return func() tea.Msg {
		return ContextsLoadedMsg{Contexts: contexts}
	}
}

// NewContextsLoadErrorMsg creates a command that sends a ContextsLoadErrorMsg
func NewContextsLoadErrorMsg(err error) tea.Cmd {
	return func() tea.Msg {
		return ContextsLoadErrorMsg{Error: err}
	}
}
