package mock

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
)

func newBrokerMock() *KafkaDataSourceMock {
	m := &KafkaDataSourceMock{}
	m.Init("")
	return m
}

func TestMockGetBrokers_StableIDs(t *testing.T) {
	m := newBrokerMock()
	b1, err := m.GetBrokers()
	assert.NoError(t, err)
	assert.Len(t, b1, 3)

	// Deterministic order and IDs across calls.
	b2, _ := m.GetBrokers()
	assert.Equal(t, b1, b2)

	controllers := 0
	for _, b := range b1 {
		if b.IsController {
			controllers++
		}
	}
	assert.Equal(t, 1, controllers, "exactly one controller")
}

func TestMockGetBrokerStats(t *testing.T) {
	m := newBrokerMock()
	stats, summary, err := m.GetBrokerStats()
	assert.NoError(t, err)
	assert.Len(t, stats, 3)
	assert.Equal(t, 3, summary.BrokerCount)
	assert.NotNil(t, summary.ControllerID)

	// At least one broker has replica skew >= 20% for styling tests.
	maxSkew := 0.0
	for _, s := range stats {
		if s.ReplicaSkew != nil && *s.ReplicaSkew > maxSkew {
			maxSkew = *s.ReplicaSkew
		}
	}
	assert.GreaterOrEqual(t, maxSkew, 20.0)
}

func TestMockGetBrokerLogDirs(t *testing.T) {
	m := newBrokerMock()

	all, err := m.GetBrokerLogDirs(nil)
	assert.NoError(t, err)
	assert.Len(t, all, 3)

	// One broker has no data.
	assert.Empty(t, all[3])

	// One dir has an error.
	hasErr := false
	for _, d := range all[1] {
		if d.Error != "" {
			hasErr = true
		}
	}
	assert.True(t, hasErr)

	// Unknown ID dropped.
	sub, _ := m.GetBrokerLogDirs([]int32{1, 99})
	assert.Contains(t, sub, int32(1))
	assert.NotContains(t, sub, int32(99))
}

func TestMockGetBrokerConfig(t *testing.T) {
	m := newBrokerMock()
	entries, err := m.GetBrokerConfig(1)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 15)

	sources := map[string]bool{}
	var sensitive, readonly, dotBytes, dotMs bool
	for _, e := range entries {
		sources[e.Source] = true
		if e.Sensitive {
			sensitive = true
		}
		if e.ReadOnly {
			readonly = true
		}
		if len(e.Name) > 6 && e.Name[len(e.Name)-6:] == ".bytes" {
			dotBytes = true
		}
		if len(e.Name) > 3 && e.Name[len(e.Name)-3:] == ".ms" {
			dotMs = true
		}
	}
	for _, s := range []string{"Dynamic broker config", "Dynamic default broker config", "Static broker config", "Default config", "Unknown"} {
		assert.True(t, sources[s], "missing source type %q", s)
	}
	assert.True(t, sensitive)
	assert.True(t, readonly)
	assert.True(t, dotBytes)
	assert.True(t, dotMs)

	_, err = m.GetBrokerConfig(99)
	var bnf api.BrokerNotFoundError
	assert.True(t, errors.As(err, &bnf))
}

func TestMockAlterBrokerConfig(t *testing.T) {
	m := newBrokerMock()

	assert.NoError(t, m.AlterBrokerConfig(1, "log.retention.ms", "999"))
	entries, _ := m.GetBrokerConfig(1)
	for _, e := range entries {
		if e.Name == "log.retention.ms" {
			assert.Equal(t, "999", e.Value)
		}
	}

	var ice api.InvalidConfigError
	assert.True(t, errors.As(m.AlterBrokerConfig(1, "k", "invalid"), &ice))

	var bnf api.BrokerNotFoundError
	assert.True(t, errors.As(m.AlterBrokerConfig(99, "k", "v"), &bnf))
}

func TestMockAlterReplicaLogDir(t *testing.T) {
	m := newBrokerMock()

	// Success: move an existing partition to another existing dir on broker 1.
	err := m.AlterReplicaLogDir(1, "order-events", 0, "/mnt/disk2/kafka")
	assert.NoError(t, err)

	// Unknown broker.
	var bnf api.BrokerNotFoundError
	assert.True(t, errors.As(m.AlterReplicaLogDir(99, "t", 0, "/d"), &bnf))

	// Unknown log dir.
	var lnf api.LogDirNotFoundError
	assert.True(t, errors.As(m.AlterReplicaLogDir(1, "order-events", 1, "/nope"), &lnf))

	// Unknown topic/partition -> partition error.
	var pe api.PartitionError
	assert.True(t, errors.As(m.AlterReplicaLogDir(1, "no-such-topic", 5, "/var/lib/kafka/logs"), &pe))
}

func TestMockGetBrokerMetrics(t *testing.T) {
	m := newBrokerMock()
	s, err := m.GetBrokerMetrics(2)
	assert.NoError(t, err)
	assert.Contains(t, s, "bytesInPerSec")
}
