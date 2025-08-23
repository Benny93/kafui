package mainpage

import (
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// Custom message types for the main page
type (
	SearchTopicsMsg         string
	ClearSearchMsg          struct{}
	SwitchResourceMsg       ResourceType
	SwitchResourceByNameMsg string
	CurrentResourceListMsg  struct {
		ResourceType ResourceType
		Items        []interface{} // Changed from []list.Item to []interface{}
	}
	TopicListMsg []TopicItem
	ErrorMsg     error
	TimerTickMsg time.Time
)

// TopicItem represents a topic item for display
type TopicItem struct {
	name  string
	topic api.Topic
}

func (t TopicItem) FilterValue() string {
	return t.name
}

func (t TopicItem) GetID() string {
	return t.name
}

func (t TopicItem) GetTopic() api.Topic {
	return t.topic
}

// ResourceType represents different types of Kafka resources
type ResourceType int

const (
	TopicResourceType ResourceType = iota
	ConsumerGroupResourceType
	SchemaResourceType
	ContextResourceType
)

func (rt ResourceType) String() string {
	switch rt {
	case TopicResourceType:
		return "topics"
	case ConsumerGroupResourceType:
		return "consumer-groups"
	case SchemaResourceType:
		return "schemas"
	case ContextResourceType:
		return "contexts"
	default:
		return "unknown"
	}
}

// Resource represents a Kafka resource that can be displayed and managed
type Resource interface {
	GetType() ResourceType
	GetName() string
	GetData() ([]ResourceItem, error)
}

// ResourceItem represents a displayable resource item
type ResourceItem interface {
	GetID() string
	GetValues() []string
	GetDetails() map[string]string
}
