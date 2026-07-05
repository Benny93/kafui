package kafds

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestCoordinationType(t *testing.T) {
	assert.Equal(t, "kraft", coordinationType("KRaft"))
	assert.Equal(t, "zookeeper", coordinationType("ZooKeeper"))
	assert.Equal(t, "unknown", coordinationType(""))
	assert.Equal(t, "unknown", coordinationType("something-else"))
}

// TestClusterStatisticsShape is a compile+shape guard that ClusterStatistics
// carries the expected fields the collector/dashboard read.
func TestClusterStatisticsShape(t *testing.T) {
	s := api.ClusterStatistics{BrokerCount: 3, ControllerID: 1, OnlinePartitions: 10}
	assert.Equal(t, 3, s.BrokerCount)
	assert.Equal(t, int32(1), s.ControllerID)
}
