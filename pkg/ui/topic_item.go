package ui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
)

// topicItem implements list.Item for displaying topics in the list
type topicItem struct {
	name  string
	topic api.Topic
}

func (t topicItem) Title() string {
	return t.name
}

func (t topicItem) Description() string {
	return fmt.Sprintf("Partitions: %d, Replication: %d",
		t.topic.NumPartitions,
		t.topic.ReplicationFactor)
}

func (t topicItem) FilterValue() string {
	return t.name
}
