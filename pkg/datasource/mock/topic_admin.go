package mock

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/analysis"
	"github.com/Benny93/kafui/pkg/api"
)

// mockBrokerIDs mirrors broker.go's fixture cluster (3 brokers).
var mockBrokerIDs = []int32{1, 2, 3}

// currentTopics returns the current context's topic map, creating the context
// entry if needed. Callers hold kp.topicMu.
func currentTopics() map[string]api.Topic {
	ctxData, ok := mockContexts[currentContext]
	if !ok {
		ctxData = &mockContextData{topics: map[string]api.Topic{}}
		mockContexts[currentContext] = ctxData
	}
	if ctxData.topics == nil {
		ctxData.topics = map[string]api.Topic{}
	}
	return ctxData.topics
}

// --- TP-2: GetTopicConfig ---

// GetTopicConfig returns a realistic config set with the topic's own overrides
// layered over cluster defaults, plus one sensitive entry.
func (kp *KafkaDataSourceMock) GetTopicConfig(topicName string) ([]api.TopicConfigEntry, error) {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topic, ok := currentTopics()[topicName]
	if !ok {
		return nil, api.TopicNotFoundError{TopicName: topicName}
	}

	defaults := []struct{ name, def string }{
		{"cleanup.policy", "delete"},
		{"retention.ms", "604800000"},
		{"max.message.bytes", "1048576"},
		{"min.insync.replicas", "1"},
	}
	entries := make([]api.TopicConfigEntry, 0, len(defaults)+2)
	for _, d := range defaults {
		e := api.TopicConfigEntry{Name: d.name, Value: d.def, Default: d.def, Source: "Default config"}
		if ov, has := topic.ConfigEntries[d.name]; has && ov != nil {
			e.Value = *ov
			e.Source = "Topic"
		}
		entries = append(entries, e)
	}
	// Any extra topic-level overrides not covered above.
	seen := map[string]bool{"cleanup.policy": true, "retention.ms": true, "max.message.bytes": true, "min.insync.replicas": true}
	for name, val := range topic.ConfigEntries {
		if seen[name] || val == nil {
			continue
		}
		entries = append(entries, api.TopicConfigEntry{Name: name, Value: *val, Source: "Topic"})
	}
	// A sensitive entry (value masked by the UI layer).
	entries = append(entries, api.TopicConfigEntry{
		Name: "sasl.jaas.config", Value: "secret", Sensitive: true, Source: "Default config",
	})

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// --- TP-3: GetTopicDetails ---

// GetTopicDetails builds multi-partition fixtures including one under-replicated
// partition.
func (kp *KafkaDataSourceMock) GetTopicDetails(topicName string) (api.TopicDetails, error) {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topic, ok := currentTopics()[topicName]
	if !ok {
		return api.TopicDetails{}, api.TopicNotFoundError{TopicName: topicName}
	}

	n := topic.NumPartitions
	if n < 1 {
		n = 1
	}
	rf := int(topic.ReplicationFactor)
	if rf < 1 {
		rf = 1
	}
	if rf > len(mockBrokerIDs) {
		rf = len(mockBrokerIDs)
	}

	perPartition := topic.MessageCount / int64(n)
	details := api.TopicDetails{
		Name:              topicName,
		ReplicationFactor: topic.ReplicationFactor,
		IsInternal:        strings.HasPrefix(topicName, "__"),
	}
	for i := int32(0); i < n; i++ {
		leader := mockBrokerIDs[int(i)%len(mockBrokerIDs)]
		replicas := make([]int32, 0, rf)
		for r := 0; r < rf; r++ {
			replicas = append(replicas, mockBrokerIDs[(int(i)+r)%len(mockBrokerIDs)])
		}
		isr := append([]int32{}, replicas...)
		// Make partition 1 under-replicated when RF allows it.
		if i == 1 && len(isr) > 1 {
			isr = isr[:len(isr)-1]
		}
		details.Partitions = append(details.Partitions, api.PartitionInfo{
			ID:             i,
			Leader:         leader,
			Replicas:       replicas,
			ISR:            isr,
			EarliestOffset: 0,
			LatestOffset:   perPartition,
		})
		details.TotalReplicas += len(replicas)
		details.InSyncReplicas += len(isr)
		if len(isr) < len(replicas) {
			details.UnderReplicatedPartitions++
		}
	}
	return details, nil
}

// --- TP-4: GetTopicSizes ---

// GetTopicSizes returns deterministic sizes (1 KiB per message) for known topics.
func (kp *KafkaDataSourceMock) GetTopicSizes(topicNames []string) (map[string]int64, error) {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topics := currentTopics()
	out := make(map[string]int64, len(topicNames))
	for _, name := range topicNames {
		if t, ok := topics[name]; ok {
			out[name] = t.MessageCount * 1024
		}
	}
	return out, nil
}

// --- TP-5: CreateTopic ---

func (kp *KafkaDataSourceMock) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topics := currentTopics()
	if _, exists := topics[name]; exists {
		return api.TopicAlreadyExistsError{TopicName: name}
	}
	if replicationFactor < 0 {
		replicationFactor = int16(len(mockBrokerIDs))
	}
	entries := map[string]*string{}
	for k, v := range configs {
		entries[k] = v
	}
	topics[name] = api.Topic{
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries:     entries,
		MessageCount:      0,
	}
	return nil
}

// --- TP-6: DeleteTopic + capability ---

func (kp *KafkaDataSourceMock) DeleteTopic(name string) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	if kp.deletionDisabled {
		return api.TopicDeletionDisabledError{TopicName: name}
	}
	topics := currentTopics()
	if _, exists := topics[name]; !exists {
		return api.TopicNotFoundError{TopicName: name}
	}
	delete(topics, name)
	return nil
}

func (kp *KafkaDataSourceMock) IsTopicDeletionEnabled() (bool, error) {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()
	return !kp.deletionDisabled, nil
}

// SetDeletionDisabled toggles the simulated delete.topic.enable=false state for
// UI tests.
func (kp *KafkaDataSourceMock) SetDeletionDisabled(disabled bool) {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()
	kp.deletionDisabled = disabled
}

// --- TP-7: UpdateTopicConfig ---

func (kp *KafkaDataSourceMock) UpdateTopicConfig(name string, entries map[string]*string) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topics := currentTopics()
	topic, ok := topics[name]
	if !ok {
		return api.TopicNotFoundError{TopicName: name}
	}
	if topic.ConfigEntries == nil {
		topic.ConfigEntries = map[string]*string{}
	}
	for k, v := range entries {
		if v == nil {
			delete(topic.ConfigEntries, k)
			continue
		}
		val := *v
		topic.ConfigEntries[k] = &val
	}
	topics[name] = topic
	return nil
}

// --- TP-8: IncreasePartitions ---

func (kp *KafkaDataSourceMock) IncreasePartitions(name string, totalCount int32) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topics := currentTopics()
	topic, ok := topics[name]
	if !ok {
		return api.TopicNotFoundError{TopicName: name}
	}
	switch {
	case totalCount < topic.NumPartitions:
		return api.PartitionDecreaseError{TopicName: name, Current: topic.NumPartitions, Requested: totalCount}
	case totalCount == topic.NumPartitions:
		return api.PartitionNoopError{TopicName: name, Current: topic.NumPartitions}
	}
	topic.NumPartitions = totalCount
	topics[name] = topic
	return nil
}

// --- TP-9: PurgeTopicMessages ---

func (kp *KafkaDataSourceMock) PurgeTopicMessages(name string, partition int32) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topics := currentTopics()
	topic, ok := topics[name]
	if !ok {
		return api.TopicNotFoundError{TopicName: name}
	}
	policy := "delete"
	if p, has := topic.ConfigEntries["cleanup.policy"]; has && p != nil {
		policy = *p
	}
	if !cleanupAllowsDelete(policy) {
		return api.CleanupPolicyError{TopicName: name, Policy: policy}
	}
	// Reset the whole topic's message count (partition-scoped purge is simulated
	// as a proportional reduction).
	if partition == -1 {
		topic.MessageCount = 0
	} else if topic.NumPartitions > 0 {
		topic.MessageCount -= topic.MessageCount / int64(topic.NumPartitions)
		if topic.MessageCount < 0 {
			topic.MessageCount = 0
		}
	}
	topics[name] = topic
	kp.counterMutex.Lock()
	delete(kp.messageCounters, name)
	kp.counterMutex.Unlock()
	return nil
}

func cleanupAllowsDelete(policy string) bool {
	for _, p := range strings.Split(policy, ",") {
		if strings.TrimSpace(p) == "delete" {
			return true
		}
	}
	return false
}

// --- TP-10: RecreateTopic ---

func (kp *KafkaDataSourceMock) RecreateTopic(name string) error {
	kp.topicMu.Lock()
	topics := currentTopics()
	topic, ok := topics[name]
	if !ok {
		kp.topicMu.Unlock()
		return api.TopicNotFoundError{TopicName: name}
	}
	// Snapshot, delete, recreate immediately with a fresh (empty) topic.
	fresh := api.Topic{
		NumPartitions:     topic.NumPartitions,
		ReplicationFactor: topic.ReplicationFactor,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries:     topic.ConfigEntries,
		MessageCount:      0,
	}
	topics[name] = fresh
	kp.topicMu.Unlock()
	return nil
}

// --- TP-11: ChangeReplicationFactor ---

func (kp *KafkaDataSourceMock) ChangeReplicationFactor(name string, newFactor int16) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()

	topics := currentTopics()
	topic, ok := topics[name]
	if !ok {
		return api.TopicNotFoundError{TopicName: name}
	}
	switch {
	case newFactor < 1:
		return api.InvalidReplicationFactorError{TopicName: name, Reason: "replication factor must be at least 1"}
	case int(newFactor) > len(mockBrokerIDs):
		return api.InvalidReplicationFactorError{TopicName: name, Reason: fmt.Sprintf("replication factor %d exceeds the %d online broker(s)", newFactor, len(mockBrokerIDs))}
	case newFactor == topic.ReplicationFactor:
		return api.InvalidReplicationFactorError{TopicName: name, Reason: fmt.Sprintf("replication factor is already %d", newFactor)}
	}
	topic.ReplicationFactor = newFactor
	topics[name] = topic
	return nil
}

// --- TP-29/TP-30: analysis ---

// StartTopicAnalysis simulates a fast scan over a sample of generated messages
// and stores a completed result.
func (kp *KafkaDataSourceMock) StartTopicAnalysis(_ context.Context, topicName string) error {
	kp.topicMu.Lock()
	_, ok := currentTopics()[topicName]
	kp.topicMu.Unlock()
	if !ok {
		return api.TopicNotFoundError{TopicName: topicName}
	}

	agg := analysis.NewAggregator(topicName)
	base := time.Now().Add(-24 * time.Hour)
	const sample = 200
	for i := 0; i < sample; i++ {
		msg := kp.generateMessage(topicName)
		msg.Timestamp = base.Add(time.Duration(i) * time.Minute)
		agg.Add(msg)
	}
	result := agg.Result()

	kp.topicMu.Lock()
	if kp.analyses == nil {
		kp.analyses = map[string]*api.TopicAnalysis{}
	}
	kp.analyses[topicName] = &api.TopicAnalysis{
		Topic:  topicName,
		State:  api.AnalysisCompleted,
		Result: &result,
		Progress: api.AnalysisProgress{
			StartTime:        base,
			ProcessedOffsets: sample,
			TotalOffsets:     sample,
			MessagesScanned:  sample,
			BytesScanned:     result.KeySize.Sum + result.ValueSize.Sum,
		},
	}
	kp.topicMu.Unlock()
	return nil
}

func (kp *KafkaDataSourceMock) GetTopicAnalysis(topicName string) (*api.TopicAnalysis, error) {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()
	if kp.analyses == nil {
		return nil, nil
	}
	return kp.analyses[topicName], nil
}

func (kp *KafkaDataSourceMock) CancelTopicAnalysis(topicName string) error {
	kp.topicMu.Lock()
	defer kp.topicMu.Unlock()
	delete(kp.analyses, topicName)
	return nil
}
