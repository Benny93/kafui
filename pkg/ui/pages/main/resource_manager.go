package mainpage

import (
	"strconv"

	"github.com/Benny93/kafui/pkg/api"
)

// ResourceManager manages different resource types for the main page
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

// Resource implementations

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

// GetData fetches the topic data
func (tr *TopicResource) GetData() ([]ResourceItem, error) {
	topics, err := tr.dataSource.GetTopics()
	if err != nil {
		return nil, err
	}

	items := make([]ResourceItem, 0, len(topics))
	for name, topic := range topics {
		items = append(items, &TopicResourceItem{
			id:                name,
			topic:             topic,
			partitions:        topic.NumPartitions,
			replicationFactor: topic.ReplicationFactor,
			messageCount:      topic.MessageCount,
		})
	}

	return items, nil
}

// TopicResourceItem represents a single Kafka topic
type TopicResourceItem struct {
	id                string
	topic             api.Topic
	partitions        int32
	replicationFactor int16
	messageCount      int64
}

// GetID returns the unique identifier for this topic
func (tri *TopicResourceItem) GetID() string {
	return tri.id
}

// GetValues returns the values for each column
func (tri *TopicResourceItem) GetValues() []string {
	return []string{
		tri.id,
		strconv.FormatInt(int64(tri.partitions), 10),
		strconv.FormatInt(int64(tri.replicationFactor), 10),
		strconv.FormatInt(tri.messageCount, 10),
	}
}

// GetDetails returns detailed information about this topic
func (tri *TopicResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":               tri.id,
		"Partitions":         strconv.FormatInt(int64(tri.partitions), 10),
		"Replication Factor": strconv.FormatInt(int64(tri.replicationFactor), 10),
		"Message Count":      strconv.FormatInt(tri.messageCount, 10),
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

// GetData fetches the consumer group data
func (cgr *ConsumerGroupResource) GetData() ([]ResourceItem, error) {
	groups, err := cgr.dataSource.GetConsumerGroups()
	if err != nil {
		return nil, err
	}

	items := make([]ResourceItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, &ConsumerGroupResourceItem{
			id:        group.Name,
			group:     group,
			state:     group.State,
			consumers: group.Consumers,
		})
	}

	return items, nil
}

// ConsumerGroupResourceItem represents a single Kafka consumer group
type ConsumerGroupResourceItem struct {
	id        string
	group     api.ConsumerGroup
	state     string
	consumers int
}

// GetID returns the unique identifier for this consumer group
func (cgri *ConsumerGroupResourceItem) GetID() string {
	return cgri.id
}

// GetValues returns the values for each column
func (cgri *ConsumerGroupResourceItem) GetValues() []string {
	return []string{
		cgri.id,
		cgri.state,
		strconv.Itoa(cgri.consumers),
	}
}

// GetDetails returns detailed information about this consumer group
func (cgri *ConsumerGroupResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":      cgri.id,
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

// GetData fetches the schema data
func (sr *SchemaResource) GetData() ([]ResourceItem, error) {
	// TODO: Implement schema data fetching
	// This would require implementing schema registry functionality in the data source
	return []ResourceItem{}, nil
}

// SchemaResourceItem represents a single Kafka schema
type SchemaResourceItem struct {
	id         string
	schema     interface{} // Placeholder for schema type
	subject    string
	version    int
	schemaID   int
	schemaType string
}

// GetID returns the unique identifier for this schema
func (sri *SchemaResourceItem) GetID() string {
	return sri.id
}

// GetValues returns the values for each column
func (sri *SchemaResourceItem) GetValues() []string {
	return []string{
		sri.subject,
		strconv.Itoa(sri.version),
		strconv.Itoa(sri.schemaID),
		sri.schemaType,
	}
}

// GetDetails returns detailed information about this schema
func (sri *SchemaResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Subject": sri.subject,
		"Version": strconv.Itoa(sri.version),
		"ID":      strconv.Itoa(sri.schemaID),
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
			id:        context,
			context:   context, // Store the actual context
			name:      context,
			isCurrent: isCurrent,
		})
	}

	return items, nil
}

// ContextResourceItem represents a single Kafka context
type ContextResourceItem struct {
	id        string
	context   string // Store the actual context
	name      string
	isCurrent string
}

// GetID returns the unique identifier for this context
func (cri *ContextResourceItem) GetID() string {
	return cri.id
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
