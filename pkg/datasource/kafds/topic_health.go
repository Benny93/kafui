package kafds

import (
	"fmt"

	"github.com/IBM/sarama"

	"github.com/Benny93/kafui/pkg/api"
)

// GetTopicHealth implements api.KafkaDataSource. One DescribeTopics covers the
// whole batch: the topics list used to call GetTopicDetails per topic, which
// opened a fresh cluster admin AND a fresh client per topic and then fetched
// offsets sequentially per partition — none of which the OSR column uses. On a
// remote cluster that was the difference between one round trip and hundreds.
func (kp KafkaDataSourceKaf) GetTopicHealth(topicNames []string) (map[string]api.TopicHealth, error) {
	out := make(map[string]api.TopicHealth, len(topicNames))
	if len(topicNames) == 0 {
		return out, nil
	}

	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	md, err := admin.DescribeTopics(topicNames)
	if err != nil {
		return nil, fmt.Errorf("describing topics: %w", err)
	}

	for _, t := range md {
		if t == nil || t.Err != sarama.ErrNoError {
			continue
		}
		out[t.Name] = topicHealthFrom(t)
	}
	return out, nil
}

// topicHealthFrom is the pure aggregation, so the arithmetic is testable
// without a broker.
func topicHealthFrom(t *sarama.TopicMetadata) api.TopicHealth {
	h := api.TopicHealth{IsInternal: t.IsInternal}
	for _, p := range t.Partitions {
		if p == nil {
			continue
		}
		if missing := len(p.Replicas) - len(p.Isr); missing > 0 {
			h.OutOfSyncReplicas += missing
			h.UnderReplicatedPartitions++
		}
	}
	return h
}
