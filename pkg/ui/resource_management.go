package ui

import (
	"strconv"

	"github.com/Benny93/kafui/pkg/api"
)

// ResourceType represents different types of Kafka resources
type ResourceType int

const (
	TopicResourceType ResourceType = iota
	ConsumerGroupResourceType
	SchemaResourceType
	ContextResourceType
)

// String returns the string representation of a resource type
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

// Resource represents a Kafka resource
type Resource interface {
	// GetType returns the type of the resource
	GetType() ResourceType
	
	// GetName returns the name of the resource
	GetName() string
	
	// GetHeaders returns the column headers for displaying the resource
	GetHeaders() []string
	
	// GetData fetches the data for this resource
	GetData() ([]ResourceItem, error)
	
	// GetActions returns the available actions for this resource
	GetActions() []ResourceAction
}

// ResourceItem represents a single item in a resource
type ResourceItem interface {
	// GetID returns the unique identifier for this item
	GetID() string
	
	// GetValues returns the values for each column
	GetValues() []string
	
	// GetDetails returns detailed information about this item
	GetDetails() map[string]string
}

// ResourceAction represents an action that can be performed on a resource
type ResourceAction struct {
	Name        string
	Description string
	Handler     func(item ResourceItem) error
}

// BaseResource provides a base implementation for resources
type BaseResource struct {
	resourceType ResourceType
	name         string
	dataSource   api.KafkaDataSource
}

// GetType returns the type of the resource
func (br *BaseResource) GetType() ResourceType {
	return br.resourceType
}

// GetName returns the name of the resource
func (br *BaseResource) GetName() string {
	return br.name
}

// TopicResource represents Kafka topics
type TopicResource struct {
	BaseResource
}

// NewTopicResource creates a new topic resource
func NewTopicResource(dataSource api.KafkaDataSource) *TopicResource {
	return &TopicResource{
		BaseResource: BaseResource{
			resourceType: TopicResourceType,
			name:         "Topics",
			dataSource:   dataSource,
		},
	}
}

// GetHeaders returns the column headers for topics
func (tr *TopicResource) GetHeaders() []string {
	return []string{"Name", "Partitions", "Replication Factor", "Message Count"}
}

// GetData fetches the topic data
func (tr *TopicResource) GetData() ([]ResourceItem, error) {
	topics, err := tr.dataSource.GetTopics()
	if err != nil {
		return nil, err
	}
	
	items := make([]ResourceItem, 0, len(topics))
	for name, topic := range topics {
		items = append(items, &TopicResourceItem{
			name:              name,
			partitions:        topic.NumPartitions,
			replicationFactor: topic.ReplicationFactor,
			messageCount:      topic.MessageCount,
		})
	}
	
	return items, nil
}

// GetActions returns the available actions for topics
func (tr *TopicResource) GetActions() []ResourceAction {
	return []ResourceAction{
		{
			Name:        "view",
			Description: "View topic details",
			Handler:     nil, // Implementation would go here
		},
		{
			Name:        "delete",
			Description: "Delete topic",
			Handler:     nil, // Implementation would go here
		},
	}
}

// TopicResourceItem represents a single Kafka topic
type TopicResourceItem struct {
	name              string
	partitions        int32
	replicationFactor int16
	messageCount      int64
}

// GetID returns the unique identifier for this topic
func (tri *TopicResourceItem) GetID() string {
	return tri.name
}

// GetValues returns the values for each column
func (tri *TopicResourceItem) GetValues() []string {
	return []string{
		tri.name,
		strconv.FormatInt(int64(tri.partitions), 10),
		strconv.FormatInt(int64(tri.replicationFactor), 10),
		strconv.FormatInt(tri.messageCount, 10),
	}
}

// GetDetails returns detailed information about this topic
func (tri *TopicResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":              tri.name,
		"Partitions":        strconv.FormatInt(int64(tri.partitions), 10),
		"Replication Factor": strconv.FormatInt(int64(tri.replicationFactor), 10),
		"Message Count":     strconv.FormatInt(tri.messageCount, 10),
	}
}

// ConsumerGroupResource represents Kafka consumer groups
type ConsumerGroupResource struct {
	BaseResource
}

// NewConsumerGroupResource creates a new consumer group resource
func NewConsumerGroupResource(dataSource api.KafkaDataSource) *ConsumerGroupResource {
	return &ConsumerGroupResource{
		BaseResource: BaseResource{
			resourceType: ConsumerGroupResourceType,
			name:         "Consumer Groups",
			dataSource:   dataSource,
		},
	}
}

// GetHeaders returns the column headers for consumer groups
func (cgr *ConsumerGroupResource) GetHeaders() []string {
	return []string{"Name", "State", "Consumers"}
}

// GetData fetches the consumer group data
func (cgr *ConsumerGroupResource) GetData() ([]ResourceItem, error) {
	groups, err := cgr.dataSource.GetConsumerGroups()
	if err != nil {
		return nil, err
	}
	
	items := make([]ResourceItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, &ConsumerGroupResourceItem{
			name:      group.Name,
			state:     group.State,
			consumers: group.Consumers,
		})
	}
	
	return items, nil
}

// GetActions returns the available actions for consumer groups
func (cgr *ConsumerGroupResource) GetActions() []ResourceAction {
	return []ResourceAction{
		{
			Name:        "view",
			Description: "View consumer group details",
			Handler:     nil, // Implementation would go here
		},
		{
			Name:        "delete",
			Description: "Delete consumer group",
			Handler:     nil, // Implementation would go here
		},
	}
}

// ConsumerGroupResourceItem represents a single Kafka consumer group
type ConsumerGroupResourceItem struct {
	name      string
	state     string
	consumers int
}

// GetID returns the unique identifier for this consumer group
func (cgri *ConsumerGroupResourceItem) GetID() string {
	return cgri.name
}

// GetValues returns the values for each column
func (cgri *ConsumerGroupResourceItem) GetValues() []string {
	return []string{
		cgri.name,
		cgri.state,
		strconv.Itoa(cgri.consumers),
	}
}

// GetDetails returns detailed information about this consumer group
func (cgri *ConsumerGroupResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":      cgri.name,
		"State":     cgri.state,
		"Consumers": strconv.Itoa(cgri.consumers),
	}
}

// SchemaResource represents Kafka schemas
type SchemaResource struct {
	BaseResource
}

// NewSchemaResource creates a new schema resource
func NewSchemaResource(dataSource api.KafkaDataSource) *SchemaResource {
	return &SchemaResource{
		BaseResource: BaseResource{
			resourceType: SchemaResourceType,
			name:         "Schemas",
			dataSource:   dataSource,
		},
	}
}

// GetHeaders returns the column headers for schemas
func (sr *SchemaResource) GetHeaders() []string {
	return []string{"Subject", "Version", "ID", "Type"}
}

// GetData fetches the schema data
func (sr *SchemaResource) GetData() ([]ResourceItem, error) {
	// TODO: Implement schema data fetching
	// This would require implementing schema registry functionality in the data source
	return []ResourceItem{}, nil
}

// GetActions returns the available actions for schemas
func (sr *SchemaResource) GetActions() []ResourceAction {
	return []ResourceAction{
		{
			Name:        "view",
			Description: "View schema details",
			Handler:     nil, // Implementation would go here
		},
		{
			Name:        "delete",
			Description: "Delete schema",
			Handler:     nil, // Implementation would go here
		},
	}
}

// SchemaResourceItem represents a single Kafka schema
type SchemaResourceItem struct {
	subject    string
	version    int
	id         int
	schemaType string
}

// GetID returns the unique identifier for this schema
func (sri *SchemaResourceItem) GetID() string {
	return sri.subject
}

// GetValues returns the values for each column
func (sri *SchemaResourceItem) GetValues() []string {
	return []string{
		sri.subject,
		strconv.Itoa(sri.version),
		strconv.Itoa(sri.id),
		sri.schemaType,
	}
}

// GetDetails returns detailed information about this schema
func (sri *SchemaResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Subject": sri.subject,
		"Version": strconv.Itoa(sri.version),
		"ID":      strconv.Itoa(sri.id),
		"Type":    sri.schemaType,
	}
}

// ContextResource represents Kafka contexts
type ContextResource struct {
	BaseResource
}

// NewContextResource creates a new context resource
func NewContextResource(dataSource api.KafkaDataSource) *ContextResource {
	return &ContextResource{
		BaseResource: BaseResource{
			resourceType: ContextResourceType,
			name:         "Contexts",
			dataSource:   dataSource,
		},
	}
}

// GetHeaders returns the column headers for contexts
func (cr *ContextResource) GetHeaders() []string {
	return []string{"Name", "Current"}
}

// GetData fetches the context data
func (cr *ContextResource) GetData() ([]ResourceItem, error) {
	contexts, err := cr.dataSource.GetContexts()
	if err != nil {
		return nil, err
	}
	
	currentContext := cr.dataSource.GetContext()
	
	items := make([]ResourceItem, 0, len(contexts))
	for _, context := range contexts {
		isCurrent := "No"
		if context == currentContext {
			isCurrent = "Yes"
		}
		items = append(items, &ContextResourceItem{
			name:       context,
			isCurrent:  isCurrent,
		})
	}
	
	return items, nil
}

// GetActions returns the available actions for contexts
func (cr *ContextResource) GetActions() []ResourceAction {
	return []ResourceAction{
		{
			Name:        "switch",
			Description: "Switch to context",
			Handler:     nil, // Implementation would go here
		},
	}
}

// ContextResourceItem represents a single Kafka context
type ContextResourceItem struct {
	name      string
	isCurrent string
}

// GetID returns the unique identifier for this context
func (cri *ContextResourceItem) GetID() string {
	return cri.name
}

// GetValues returns the values for each column
func (cri *ContextResourceItem) GetValues() []string {
	return []string{
		cri.name,
		cri.isCurrent,
	}
}

// GetDetails returns detailed information about this context
func (cri *ContextResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":    cri.name,
		"Current": cri.isCurrent,
	}
}

// ResourceManager manages different resource types
type ResourceManager struct {
	dataSource api.KafkaDataSource
	resources  map[ResourceType]Resource
}

// NewResourceManager creates a new resource manager
func NewResourceManager(dataSource api.KafkaDataSource) *ResourceManager {
	rm := &ResourceManager{
		dataSource: dataSource,
		resources:  make(map[ResourceType]Resource),
	}
	
	// Initialize default resources
	rm.resources[TopicResourceType] = NewTopicResource(dataSource)
	rm.resources[ConsumerGroupResourceType] = NewConsumerGroupResource(dataSource)
	rm.resources[SchemaResourceType] = NewSchemaResource(dataSource)
	rm.resources[ContextResourceType] = NewContextResource(dataSource)
	
	return rm
}

// GetResource returns a resource by type
func (rm *ResourceManager) GetResource(resourceType ResourceType) Resource {
	return rm.resources[resourceType]
}

// GetAllResources returns all available resources
func (rm *ResourceManager) GetAllResources() map[ResourceType]Resource {
	return rm.resources
}

// RegisterResource registers a new resource type
func (rm *ResourceManager) RegisterResource(resourceType ResourceType, resource Resource) {
	rm.resources[resourceType] = resource
}

// GetResourceTypes returns all available resource types
func (rm *ResourceManager) GetResourceTypes() []ResourceType {
	types := make([]ResourceType, 0, len(rm.resources))
	for resourceType := range rm.resources {
		types = append(types, resourceType)
	}
	return types
}

// GetResourceNames returns the names of all available resources
func (rm *ResourceManager) GetResourceNames() []string {
	names := make([]string, 0, len(rm.resources))
	for _, resource := range rm.resources {
		names = append(names, resource.GetName())
	}
	return names
}