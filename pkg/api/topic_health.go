package api

// TopicHealth is the replication health a topic list needs per row. It is
// deliberately small: it comes from cluster metadata alone, so a whole page of
// topics resolves in one request with no per-partition offset lookups.
type TopicHealth struct {
	// OutOfSyncReplicas is the number of replicas across the topic that are not
	// in the ISR. Zero on a healthy topic.
	OutOfSyncReplicas int
	// UnderReplicatedPartitions is the number of partitions whose ISR is
	// smaller than their replica set. Zero on a healthy topic.
	UnderReplicatedPartitions int
	// IsInternal marks Kafka's own topics (__consumer_offsets and friends).
	IsInternal bool
}
