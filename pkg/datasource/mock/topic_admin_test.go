package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTopicMock(t *testing.T) *KafkaDataSourceMock {
	t.Helper()
	ds := &KafkaDataSourceMock{}
	ds.Init("")
	// Use a disposable context so mutations don't leak into other tests.
	currentContext = "kafka-dev"
	return ds
}

func TestMockCreateAndDeleteRoundTrip(t *testing.T) {
	ds := newTopicMock(t)

	require.NoError(t, ds.CreateTopic("brand-new", 4, 2, map[string]*string{}))
	names, _ := ds.GetTopicNames()
	assert.Contains(t, names, "brand-new")

	// Duplicate create is rejected.
	err := ds.CreateTopic("brand-new", 1, 1, nil)
	var exists api.TopicAlreadyExistsError
	assert.True(t, errors.As(err, &exists))

	require.NoError(t, ds.DeleteTopic("brand-new"))
	names, _ = ds.GetTopicNames()
	assert.NotContains(t, names, "brand-new")

	// Deleting a missing topic is a typed error.
	err = ds.DeleteTopic("brand-new")
	var nf api.TopicNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestMockDeletionDisabled(t *testing.T) {
	ds := newTopicMock(t)
	ds.SetDeletionDisabled(true)

	enabled, _ := ds.IsTopicDeletionEnabled()
	assert.False(t, enabled)

	require.NoError(t, ds.CreateTopic("x", 1, 1, nil))
	err := ds.DeleteTopic("x")
	var dd api.TopicDeletionDisabledError
	assert.True(t, errors.As(err, &dd))
	ds.SetDeletionDisabled(false)
	_ = ds.DeleteTopic("x")
}

func TestMockUpdateConfigRoundTrip(t *testing.T) {
	ds := newTopicMock(t)
	require.NoError(t, ds.CreateTopic("cfg-topic", 1, 1, map[string]*string{}))

	v := "1234"
	require.NoError(t, ds.UpdateTopicConfig("cfg-topic", map[string]*string{"retention.ms": &v}))

	entries, err := ds.GetTopicConfig("cfg-topic")
	require.NoError(t, err)
	byName := map[string]api.TopicConfigEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	assert.Equal(t, "1234", byName["retention.ms"].Value)
	assert.Equal(t, "Topic", byName["retention.ms"].Source)
	// The sensitive fixture entry is present.
	assert.True(t, byName["sasl.jaas.config"].Sensitive)
}

func TestMockIncreasePartitions(t *testing.T) {
	ds := newTopicMock(t)
	require.NoError(t, ds.CreateTopic("parts", 3, 1, nil))

	require.NoError(t, ds.IncreasePartitions("parts", 6))
	d, _ := ds.GetTopicDetails("parts")
	assert.Len(t, d.Partitions, 6)

	var dec api.PartitionDecreaseError
	assert.True(t, errors.As(ds.IncreasePartitions("parts", 2), &dec))
	var noop api.PartitionNoopError
	assert.True(t, errors.As(ds.IncreasePartitions("parts", 6), &noop))
}

func TestMockPurgeMessages(t *testing.T) {
	ds := newTopicMock(t)
	del := "delete"
	require.NoError(t, ds.CreateTopic("purge-me", 2, 1, map[string]*string{"cleanup.policy": &del}))
	// Seed a message count.
	require.NoError(t, ds.UpdateTopicConfig("purge-me", map[string]*string{}))
	topics := currentTopics()
	topic := topics["purge-me"]
	topic.MessageCount = 1000
	topics["purge-me"] = topic

	require.NoError(t, ds.PurgeTopicMessages("purge-me", -1))
	d, _ := ds.GetTopicSizes([]string{"purge-me"})
	assert.Equal(t, int64(0), d["purge-me"])

	// Compact topic rejects purge.
	compact := "compact"
	require.NoError(t, ds.CreateTopic("compacted", 1, 1, map[string]*string{"cleanup.policy": &compact}))
	var cp api.CleanupPolicyError
	assert.True(t, errors.As(ds.PurgeTopicMessages("compacted", -1), &cp))
}

func TestMockRecreateAndReplicationFactor(t *testing.T) {
	ds := newTopicMock(t)
	require.NoError(t, ds.CreateTopic("rc", 3, 2, nil))
	topics := currentTopics()
	topic := topics["rc"]
	topic.MessageCount = 500
	topics["rc"] = topic

	require.NoError(t, ds.RecreateTopic("rc"))
	sizes, _ := ds.GetTopicSizes([]string{"rc"})
	assert.Equal(t, int64(0), sizes["rc"], "recreate resets message count")

	require.NoError(t, ds.ChangeReplicationFactor("rc", 3))
	d, _ := ds.GetTopicDetails("rc")
	assert.Equal(t, int16(3), d.ReplicationFactor)

	var rf api.InvalidReplicationFactorError
	assert.True(t, errors.As(ds.ChangeReplicationFactor("rc", 3), &rf), "equal factor rejected")
	assert.True(t, errors.As(ds.ChangeReplicationFactor("rc", 99), &rf), "too-high factor rejected")
}

func TestMockGetTopicDetailsUnderReplicated(t *testing.T) {
	ds := newTopicMock(t)
	require.NoError(t, ds.CreateTopic("urp", 3, 3, nil))
	d, err := ds.GetTopicDetails("urp")
	require.NoError(t, err)
	assert.Positive(t, d.UnderReplicatedPartitions, "fixture includes an under-replicated partition")

	_, err = ds.GetTopicDetails("does-not-exist")
	var nf api.TopicNotFoundError
	assert.True(t, errors.As(err, &nf))
}

func TestMockAnalysisProducesCompletedResult(t *testing.T) {
	ds := newTopicMock(t)
	require.NoError(t, ds.CreateTopic("an", 3, 1, nil))

	require.NoError(t, ds.StartTopicAnalysis(context.Background(), "an"))
	a, err := ds.GetTopicAnalysis("an")
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, api.AnalysisCompleted, a.State)
	require.NotNil(t, a.Result)
	assert.Positive(t, a.Result.MessageCount)
	assert.LessOrEqual(t, a.Result.ApproxDistinctValues, a.Result.MessageCount)

	require.NoError(t, ds.CancelTopicAnalysis("an"))
	a, _ = ds.GetTopicAnalysis("an")
	assert.Nil(t, a)

	// Analysis of a missing topic is a typed error.
	err = ds.StartTopicAnalysis(context.Background(), "nope")
	var tnf api.TopicNotFoundError
	assert.True(t, errors.As(err, &tnf))
}
