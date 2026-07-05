package mainpage

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
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
	rm.resources[ACLResourceType] = NewACLResource(dataSource)
	rm.resources[BrokerResourceType] = NewBrokerResource(dataSource)
	rm.resources[QuotaResourceType] = NewQuotaResource(dataSource)
	// Connect resources are always registered; visibility (sidebar + resource
	// cycle + :connectors) is gated on api.CapKafkaConnect, mirroring how the
	// schema/ACL resources are registered here and gated in the sidebar.
	rm.resources[ConnectClusterResourceType] = NewConnectClusterResource(dataSource)
	rm.resources[ConnectorResourceType] = NewConnectorResource(dataSource)

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
			outOfSync:         -1, // filled by loadTopicDetailsExt
			size:              -1, // filled by loadTopicDetailsExt
			isInternal:        isInternalTopicName(name),
		})
	}

	return items, nil
}

// TopicResourceItem represents a single Kafka topic.
// outOfSync/size start at -1 (not loaded); detailsExtLoaded flips true once the
// extended per-page fetch (GetTopicDetails/GetTopicSizes) has completed so the
// row can distinguish a still-loading "…" from a failed "N/A".
type TopicResourceItem struct {
	id                string
	topic             api.Topic
	partitions        int32
	replicationFactor int16
	messageCount      int64
	outOfSync         int   // -1 = not loaded (UnderReplicatedPartitions once loaded)
	size              int64 // -1 = not loaded
	detailsExtLoaded  bool
	isInternal        bool
	selected          bool // multi-select marker (mirrors provider.selected)
}

// GetID returns the unique identifier for this topic
func (tri *TopicResourceItem) GetID() string {
	return tri.id
}

// GetValues returns the values for each column
func (tri *TopicResourceItem) GetValues() []string {
	parts := strconv.FormatInt(int64(tri.partitions), 10)
	repl := strconv.FormatInt(int64(tri.replicationFactor), 10)
	if tri.partitions < 0 {
		parts = "…"
	}
	if tri.replicationFactor < 0 {
		repl = "…"
	}
	return []string{
		tri.id,
		parts,
		repl,
		strconv.FormatInt(tri.messageCount, 10),
	}
}

// GetDetails returns detailed information about this topic
func (tri *TopicResourceItem) GetDetails() map[string]string {
	msgCount := strconv.FormatInt(tri.messageCount, 10)
	if tri.messageCount < 0 {
		msgCount = "…" // async count not yet loaded
	}
	parts := strconv.FormatInt(int64(tri.partitions), 10)
	if tri.partitions < 0 {
		parts = "…" // async detail not yet loaded
	}
	repl := strconv.FormatInt(int64(tri.replicationFactor), 10)
	if tri.replicationFactor < 0 {
		repl = "…"
	}
	return map[string]string{
		"Name":               tri.id,
		"Partitions":         parts,
		"Replication Factor": repl,
		"Message Count":      msgCount,
	}
}

// GetTopic returns the underlying api.Topic data
func (tri *TopicResourceItem) GetTopic() api.Topic {
	return tri.topic
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

// GetData fetches the consumer group data. This is the fast, names-only phase
// (GetConsumerGroups); real state/members/topics/lag/coordinator are filled in
// lazily for the visible page via GetConsumerGroupDetails (see loadGroupDetails).
func (cgr *ConsumerGroupResource) GetData() ([]ResourceItem, error) {
	groups, err := cgr.dataSource.GetConsumerGroups()
	if err != nil {
		return nil, err
	}

	items := make([]ResourceItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, &ConsumerGroupResourceItem{
			id:    group.Name,
			group: group,
			// detailsLoaded stays false — real state/lag are enriched lazily.
		})
	}

	return items, nil
}

// ConsumerGroupResourceItem represents a single Kafka consumer group row.
// group holds the enriched api.ConsumerGroup once detailsLoaded is true; until
// then only the name is trustworthy and columns render "…"/"—" placeholders.
type ConsumerGroupResourceItem struct {
	id            string
	group         api.ConsumerGroup
	detailsLoaded bool
}

// GetID returns the unique identifier for this consumer group
func (cgri *ConsumerGroupResourceItem) GetID() string {
	return cgri.id
}

// Group returns the underlying (possibly enriched) api.ConsumerGroup.
func (cgri *ConsumerGroupResourceItem) Group() api.ConsumerGroup { return cgri.group }

// DetailsLoaded reports whether lazy enrichment has completed for this row.
func (cgri *ConsumerGroupResourceItem) DetailsLoaded() bool { return cgri.detailsLoaded }

// State returns the enriched canonical state, or GroupStateUnknown before load.
func (cgri *ConsumerGroupResourceItem) State() string {
	if !cgri.detailsLoaded {
		return api.GroupStateUnknown
	}
	if cgri.group.State == "" {
		return api.GroupStateUnknown
	}
	return cgri.group.State
}

// SetDetail attaches enriched group data to the item.
func (cgri *ConsumerGroupResourceItem) SetDetail(g api.ConsumerGroup) {
	cgri.group = g
	cgri.detailsLoaded = true
}

// GetValues returns unstyled column values: Name, State, Members, Topics, Lag,
// Coordinator. Unenriched rows render "…"; undefined lag renders "—" (never 0).
func (cgri *ConsumerGroupResourceItem) GetValues() []string {
	if !cgri.detailsLoaded {
		return []string{cgri.id, "…", "…", "…", "…", "…"}
	}
	return []string{
		cgri.id,
		cgri.State(),
		strconv.Itoa(cgri.group.MemberCount),
		strconv.Itoa(cgri.group.TopicCount),
		formatGroupLag(cgri.group.Lag),
		formatCoordinator(cgri.group.CoordinatorID),
	}
}

// GetDetails returns detailed information about this consumer group.
func (cgri *ConsumerGroupResourceItem) GetDetails() map[string]string {
	if !cgri.detailsLoaded {
		return map[string]string{
			"Name":        cgri.id,
			"State":       "…",
			"Members":     "…",
			"Topics":      "…",
			"Lag":         "…",
			"Coordinator": "…",
		}
	}
	return map[string]string{
		"Name":        cgri.id,
		"State":       cgri.State(),
		"Members":     strconv.Itoa(cgri.group.MemberCount),
		"Topics":      strconv.Itoa(cgri.group.TopicCount),
		"Lag":         formatGroupLag(cgri.group.Lag),
		"Coordinator": formatCoordinator(cgri.group.CoordinatorID),
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

// GetData fetches the schema data from the schema registry.
func (sr *SchemaResource) GetData() ([]ResourceItem, error) {
	schemas, err := sr.dataSource.GetSchemas()
	if err != nil {
		return nil, err
	}
	items := make([]ResourceItem, 0, len(schemas))
	for _, s := range schemas {
		items = append(items, &SchemaResourceItem{
			id:      s.Subject,
			subject: s.Subject,
			// detailsLoaded stays false — version/ID/type are fetched lazily
		})
	}
	return items, nil
}

// SchemaResourceItem represents a single registered schema subject.
// version/schemaID/schemaType start as zero values; detailsLoaded is false
// until GetSchemaDetails has been called for this subject.
type SchemaResourceItem struct {
	id            string
	subject       string
	version       int
	schemaID      int
	schemaType    string
	compatibility string
	detailsLoaded bool
}

// GetID returns the subject name as the unique identifier.
func (sri *SchemaResourceItem) GetID() string {
	return sri.id
}

// GetValues returns display values for each table column.
func (sri *SchemaResourceItem) GetValues() []string {
	if !sri.detailsLoaded {
		return []string{sri.subject, "…", "…", "…", "…"}
	}
	return []string{
		sri.subject,
		strconv.Itoa(sri.version),
		strconv.Itoa(sri.schemaID),
		sri.schemaType,
		sri.compatibility,
	}
}

// GetDetails returns sidebar detail fields for this schema subject.
func (sri *SchemaResourceItem) GetDetails() map[string]string {
	if !sri.detailsLoaded {
		return map[string]string{
			"Subject":       sri.subject,
			"Version":       "…",
			"ID":            "…",
			"Type":          "…",
			"Compatibility": "…",
		}
	}
	return map[string]string{
		"Subject":       sri.subject,
		"Version":       strconv.Itoa(sri.version),
		"ID":            strconv.Itoa(sri.schemaID),
		"Type":          sri.schemaType,
		"Compatibility": sri.compatibility,
	}
}

// Exported accessor methods — used by the schema detail page.
func (sri *SchemaResourceItem) Subject() string       { return sri.subject }
func (sri *SchemaResourceItem) Version() int          { return sri.version }
func (sri *SchemaResourceItem) SchemaID() int         { return sri.schemaID }
func (sri *SchemaResourceItem) SchemaType() string    { return sri.schemaType }
func (sri *SchemaResourceItem) Compatibility() string { return sri.compatibility }

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

	items := make([]ResourceItem, 0, len(contexts))
	for _, name := range contexts {
		info, err := cr.dataSource.GetClusterDetails(name)
		if err != nil {
			// Fall back to minimal info when details are unavailable.
			info = api.ClusterInfo{Name: name}
		}
		items = append(items, &ContextResourceItem{
			id:                name,
			name:              name,
			isCurrent:         info.IsCurrent,
			brokers:           info.Brokers,
			schemaRegistryURL: info.SchemaRegistryURL,
		})
	}

	return items, nil
}

// ContextResourceItem represents a single Kafka context
type ContextResourceItem struct {
	id                string
	name              string
	isCurrent         bool
	brokers           []string
	schemaRegistryURL string
}

// GetID returns the unique identifier for this context
func (cri *ContextResourceItem) GetID() string {
	return cri.id
}

// GetValues returns the values for each column
func (cri *ContextResourceItem) GetValues() []string {
	current := "No"
	if cri.isCurrent {
		current = "Yes"
	}
	return []string{cri.name, current}
}

// GetDetails returns detailed information about this context for display in the table.
// Keys are chosen to match the slots used by convertItemsToRows:
//   - "State"    → shown in the Replication column ("active" / "inactive")
//   - "Brokers"  → shown in the Partitions column
//   - "Schema"   → shown in the Details column
func (cri *ContextResourceItem) GetDetails() map[string]string {
	state := "inactive"
	if cri.isCurrent {
		state = "★ active"
	}

	brokerStr := "-"
	if len(cri.brokers) > 0 {
		brokerStr = strings.Join(cri.brokers, ", ")
	}

	schemaStr := "-"
	if cri.schemaRegistryURL != "" {
		schemaStr = cri.schemaRegistryURL
	}

	return map[string]string{
		"State":   state,
		"Brokers": brokerStr,
		"Schema":  schemaStr,
	}
}

// ACLResource represents Kafka ACL bindings. It carries a live server-side
// filter (AQ-14) applied via GetACLsFiltered; the zero filter matches all.
type ACLResource struct {
	BaseResource
	filter api.ACLFilter
}

// NewACLResource creates a new ACL resource
func NewACLResource(dataSource api.KafkaDataSource) *ACLResource {
	return &ACLResource{
		BaseResource: BaseResource{
			resourceType: ACLResourceType,
			name:         "ACLs",
			dataSource:   dataSource,
		},
	}
}

// SetFilter installs the server-side resource-dimension filter used by the next
// GetData call (wired from the ACL filter cycle keys).
func (ar *ACLResource) SetFilter(f api.ACLFilter) { ar.filter = f }

// Filter returns the currently active server-side filter.
func (ar *ACLResource) Filter() api.ACLFilter { return ar.filter }

// GetData fetches ACL bindings from the cluster, honoring the active filter.
func (ar *ACLResource) GetData() ([]ResourceItem, error) {
	acls, err := ar.dataSource.GetACLsFiltered(ar.filter)
	if err != nil {
		return nil, err
	}
	items := make([]ResourceItem, 0, len(acls))
	for _, a := range acls {
		items = append(items, &ACLResourceItem{
			principal:    a.Principal,
			host:         a.Host,
			resourceType: a.ResourceType,
			resourceName: a.ResourceName,
			patternType:  a.PatternType,
			operation:    a.Operation,
			permission:   a.Permission,
		})
	}
	return items, nil
}

// ACLResourceItem represents a single Kafka ACL binding.
type ACLResourceItem struct {
	principal    string
	host         string
	resourceType string
	resourceName string
	patternType  string
	operation    string
	permission   string
}

// GetID returns a unique string for this ACL entry. Pattern type is part of the
// identity so a Literal and a Prefixed binding on the same name are distinct.
func (a *ACLResourceItem) GetID() string {
	return a.principal + "|" + a.resourceType + ":" + a.resourceName + "|" + a.patternType + "|" + a.operation
}

// GetValues returns the column values: principal, resource, pattern, host,
// operation, permission (AQ-13).
func (a *ACLResourceItem) GetValues() []string {
	pattern := a.patternType
	if pattern == "" {
		pattern = "Literal"
	}
	host := a.host
	if host == "" {
		host = "*"
	}
	return []string{a.principal, a.resourceType + ":" + a.resourceName, pattern, host, a.operation, a.permission}
}

// GetDetails returns the sidebar detail fields for this ACL binding.
func (a *ACLResourceItem) GetDetails() map[string]string {
	resource := a.resourceType + ":" + a.resourceName
	pattern := a.patternType
	if pattern == "" {
		pattern = "Literal"
	}
	host := a.host
	if host == "" {
		host = "*"
	}
	return map[string]string{
		"Name":        a.principal,
		"Resource":    resource,
		"PatternType": pattern,
		"Host":        host,
		"Operation":   a.operation,
		"Permission":  a.permission,
	}
}

// Entry reconstructs the full api.ACLEntry for this row (delete identity, CSV).
func (a *ACLResourceItem) Entry() api.ACLEntry {
	return api.ACLEntry{
		Principal:    a.principal,
		Host:         a.host,
		ResourceType: a.resourceType,
		ResourceName: a.resourceName,
		PatternType:  a.patternType,
		Operation:    a.operation,
		Permission:   a.permission,
	}
}

// BrokerResource represents Kafka brokers. It loads fast (ID/host/port/controller)
// via GetBrokers; per-broker statistics are enriched asynchronously (two-phase).
type BrokerResource struct {
	BaseResource
}

// NewBrokerResource creates a new broker resource.
func NewBrokerResource(dataSource api.KafkaDataSource) *BrokerResource {
	return &BrokerResource{
		BaseResource: BaseResource{
			resourceType: BrokerResourceType,
			name:         "Brokers",
			dataSource:   dataSource,
		},
	}
}

// GetData fetches the broker list (fast phase). Stats are filled in later by the
// GetBrokerStats enrichment (BrokerStatsLoadedMsg).
func (br *BrokerResource) GetData() ([]ResourceItem, error) {
	brokers, err := br.dataSource.GetBrokers()
	if err != nil {
		return nil, err
	}
	items := make([]ResourceItem, 0, len(brokers))
	for _, b := range brokers {
		items = append(items, &BrokerResourceItem{info: b})
	}
	return items, nil
}

// BrokerResourceItem represents a single broker row. Raw typed values are kept
// so sorting operates on numbers/strings, not the formatted display cells.
// hasStats stays false until GetBrokerStats enrichment arrives.
type BrokerResourceItem struct {
	info     api.BrokerInfo
	stats    api.BrokerStats
	hasStats bool
}

// GetID returns the broker ID as a string.
func (b *BrokerResourceItem) GetID() string {
	return strconv.FormatInt(int64(b.info.ID), 10)
}

// Info returns the underlying broker metadata.
func (b *BrokerResourceItem) Info() api.BrokerInfo { return b.info }

// Stats returns the enriched broker statistics (zero value until loaded).
func (b *BrokerResourceItem) Stats() api.BrokerStats { return b.stats }

// HasStats reports whether stats enrichment has completed for this broker.
func (b *BrokerResourceItem) HasStats() bool { return b.hasStats }

// SetStats attaches enriched statistics to the item.
func (b *BrokerResourceItem) SetStats(s api.BrokerStats) {
	b.stats = s
	b.hasStats = true
}

// idCell renders the ID column with a controller marker.
func (b *BrokerResourceItem) idCell() string {
	id := b.GetID()
	if b.info.IsController {
		return id + " ★"
	}
	return id
}

// GetValues returns unstyled column values: ID, Host, Port, Disk, ISR, Skew.
// Stats columns render "…" until enrichment completes.
func (b *BrokerResourceItem) GetValues() []string {
	port := strconv.FormatInt(int64(b.info.Port), 10)
	if !b.hasStats {
		return []string{b.idCell(), b.info.Host, port, "…", "…", "…"}
	}
	disk := shared.FormatDiskUsage(b.stats.SegmentSize, b.stats.SegmentCount)
	isr, _ := shared.FormatISR(b.stats.InSyncReplicaCount, b.stats.ReplicaCount)
	skew := shared.FormatSkew(b.stats.ReplicaSkew)
	return []string{b.idCell(), b.info.Host, port, disk, isr, skew}
}

// GetDetails returns detail fields for the broker.
func (b *BrokerResourceItem) GetDetails() map[string]string {
	port := strconv.FormatInt(int64(b.info.Port), 10)
	controller := "No"
	if b.info.IsController {
		controller = "Yes (Active Controller)"
	}
	m := map[string]string{
		"ID":         b.GetID(),
		"Host":       b.info.Host,
		"Port":       port,
		"Rack":       b.info.Rack,
		"Controller": controller,
	}
	if b.hasStats {
		m["Disk Usage"] = shared.FormatDiskUsage(b.stats.SegmentSize, b.stats.SegmentCount)
		m["Leaders"] = strconv.Itoa(b.stats.LeaderCount)
		m["Replicas"] = strconv.Itoa(b.stats.ReplicaCount)
		isr, _ := shared.FormatISR(b.stats.InSyncReplicaCount, b.stats.ReplicaCount)
		m["ISR"] = isr
		m["Leader Skew"] = shared.FormatSkew(b.stats.LeaderSkew)
		m["Replica Skew"] = shared.FormatSkew(b.stats.ReplicaSkew)
	}
	return m
}

// QuotaResource represents Kafka client quotas (AQ-19).
type QuotaResource struct {
	BaseResource
}

// NewQuotaResource creates a new client-quota resource.
func NewQuotaResource(dataSource api.KafkaDataSource) *QuotaResource {
	return &QuotaResource{
		BaseResource: BaseResource{
			resourceType: QuotaResourceType,
			name:         "Quotas",
			dataSource:   dataSource,
		},
	}
}

// GetData fetches all client quotas from the cluster (already deterministically
// ordered by the datasource: user → client-id → ip, absent identifiers last).
func (qr *QuotaResource) GetData() ([]ResourceItem, error) {
	quotas, err := qr.dataSource.GetClientQuotas()
	if err != nil {
		return nil, err
	}
	items := make([]ResourceItem, 0, len(quotas))
	for _, q := range quotas {
		items = append(items, &QuotaResourceItem{entry: q})
	}
	return items, nil
}

// QuotaResourceItem represents a single client-quota entry.
type QuotaResourceItem struct {
	entry api.ClientQuotaEntry
}

// Entity returns the underlying quota entity.
func (q *QuotaResourceItem) Entity() api.ClientQuotaEntity { return q.entry.Entity }

// Quotas returns the underlying quota property map.
func (q *QuotaResourceItem) Quotas() map[string]float64 { return q.entry.Quotas }

// quotaIDStr renders an entity identifier pointer: nil → "<any>", "" → "<default>".
func quotaIDStr(p *string) string {
	if p == nil {
		return "<any>"
	}
	if *p == "" {
		return "<default>"
	}
	return *p
}

// GetID uniquely identifies the quota entity (independent of its values).
func (q *QuotaResourceItem) GetID() string {
	return "user=" + quotaIDStr(q.entry.Entity.User) +
		"|client=" + quotaIDStr(q.entry.Entity.ClientID) +
		"|ip=" + quotaIDStr(q.entry.Entity.IP)
}

// formatQuotaValues renders the quota map sorted as "name=value" pairs.
func formatQuotaValues(quotas map[string]float64) string {
	if len(quotas) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(quotas))
	for k := range quotas {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+strconv.FormatFloat(quotas[k], 'f', -1, 64))
	}
	return strings.Join(parts, ", ")
}

// GetValues returns column values: user, client id, ip, quotas.
func (q *QuotaResourceItem) GetValues() []string {
	return []string{
		quotaIDStr(q.entry.Entity.User),
		quotaIDStr(q.entry.Entity.ClientID),
		quotaIDStr(q.entry.Entity.IP),
		formatQuotaValues(q.entry.Quotas),
	}
}

// GetDetails returns detail fields for this quota entry.
func (q *QuotaResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"User":     quotaIDStr(q.entry.Entity.User),
		"ClientID": quotaIDStr(q.entry.Entity.ClientID),
		"IP":       quotaIDStr(q.entry.Entity.IP),
		"Quotas":   formatQuotaValues(q.entry.Quotas),
	}
}
