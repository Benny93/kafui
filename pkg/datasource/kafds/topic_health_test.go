package kafds

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
)

func TestTopicHealthCountsOutOfSyncReplicas(t *testing.T) {
	md := &sarama.TopicMetadata{
		Name: "orders",
		Partitions: []*sarama.PartitionMetadata{
			// Healthy: every replica in the ISR.
			{ID: 0, Replicas: []int32{1, 2, 3}, Isr: []int32{1, 2, 3}},
			// One replica lagging.
			{ID: 1, Replicas: []int32{1, 2, 3}, Isr: []int32{1, 2}},
			// Two lagging.
			{ID: 2, Replicas: []int32{1, 2, 3}, Isr: []int32{1}},
		},
	}

	h := topicHealthFrom(md)
	assert.Equal(t, 3, h.OutOfSyncReplicas, "one lagging replica plus two")
	assert.Equal(t, 2, h.UnderReplicatedPartitions, "two partitions are short")
	assert.False(t, h.IsInternal)
}

// Zero is the healthy answer and the common one — it must not be confused with
// "failed to load", which the UI renders as N/A from a missing map entry.
func TestTopicHealthIsZeroOnAHealthyTopic(t *testing.T) {
	md := &sarama.TopicMetadata{
		Name: "orders",
		Partitions: []*sarama.PartitionMetadata{
			{ID: 0, Replicas: []int32{1, 2}, Isr: []int32{1, 2}},
			{ID: 1, Replicas: []int32{1, 2}, Isr: []int32{1, 2}},
		},
	}
	h := topicHealthFrom(md)
	assert.Zero(t, h.OutOfSyncReplicas)
	assert.Zero(t, h.UnderReplicatedPartitions)
}

func TestTopicHealthMarksInternalTopics(t *testing.T) {
	h := topicHealthFrom(&sarama.TopicMetadata{Name: "__consumer_offsets", IsInternal: true})
	assert.True(t, h.IsInternal)
}

// A partition with more ISR entries than replicas is nonsense, but must not
// produce a negative count.
func TestTopicHealthNeverGoesNegative(t *testing.T) {
	h := topicHealthFrom(&sarama.TopicMetadata{
		Name:       "weird",
		Partitions: []*sarama.PartitionMetadata{{ID: 0, Replicas: []int32{1}, Isr: []int32{1, 2}}},
	})
	assert.Zero(t, h.OutOfSyncReplicas)
	assert.Zero(t, h.UnderReplicatedPartitions)
}
