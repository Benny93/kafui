package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTopicModel(topicName string) *Model {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	m := NewModel(ds, topicName, api.Topic{NumPartitions: 3, ReplicationFactor: 1})
	return m
}

func TestTopicConsumerGroups_FetchOnKeypress(t *testing.T) {
	m := newTopicModel("order-events")
	k := m.keys

	// Groups are not fetched until the overlay is opened.
	assert.False(t, m.showGroups)
	assert.Nil(t, m.groups)

	cmd := k.handleShowGroups(m)
	require.NotNil(t, cmd)
	assert.True(t, m.showGroups)
	assert.True(t, m.groupsLoading)

	msg, ok := cmd().(TopicGroupsLoadedMsg)
	require.True(t, ok)
	m.handlers.handleTopicGroupsLoaded(m, msg)

	assert.False(t, m.groupsLoading)
	require.Len(t, m.groups, 1)
	assert.Equal(t, "order-processor", m.groups[0].Name)
}

func TestTopicConsumerGroups_NavigateToDetail(t *testing.T) {
	m := newTopicModel("order-events")
	m.showGroups = true
	m.groups = []api.ConsumerGroup{{Name: "order-processor", CoordinatorID: 1}}
	m.groupsCursor = 0

	cmd := m.keys.handleGroupsOverlayKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	pc, ok := cmd().(core.PageChangeMsg)
	require.True(t, ok)
	assert.Equal(t, "consumer_group:order-processor", pc.PageID)

	// Esc closes the overlay.
	m.keys.handleGroupsOverlayKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.showGroups)
}
