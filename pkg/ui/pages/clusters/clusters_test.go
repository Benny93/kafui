package clusters

import (
	"context"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newModel builds a page with a nil-collector Common so tests can feed
// overviews directly via ClusterStatsUpdatedMsg.
func newModel(t *testing.T) *Model {
	t.Helper()
	m := NewModelWithCommon(core.NewCommon(nil))
	m.SetDimensions(160, 40)
	return m
}

// feed delivers overviews as the collector would and returns the rendered content.
func feed(m *Model, clusters []api.ClusterOverview) string {
	m.handle(cluster.ClusterStatsUpdatedMsg{Clusters: clusters})
	return m.renderContent(160, 40)
}

func sample() []api.ClusterOverview {
	return []api.ClusterOverview{
		{Name: "kafka-dev", Status: api.ClusterOnline, Version: "3.7.0", BrokerCount: 3,
			OnlinePartitionCount: 12, TopicCount: 5, MessagesInPerSec: 42.5,
			BytesInPerSec: 1536, BytesOutPerSec: 1048576},
		{Name: "kafka-prod", Status: api.ClusterOnline, Version: "3.6.0", BrokerCount: 5,
			OnlinePartitionCount: 30, TopicCount: 40, MessagesInPerSec: -1,
			BytesInPerSec: -1, BytesOutPerSec: -1, ReadOnly: true},
	}
}

func TestRenderColumnsAndNames(t *testing.T) {
	m := newModel(t)
	out := feed(m, sample())

	for _, col := range []string{"Name", "Status", "Version", "Brokers",
		"Online Partitions", "Topics", "Msgs/s", "Bytes In/s", "Bytes Out/s", "Access"} {
		assert.Contains(t, out, col, "missing column header %q", col)
	}
	assert.Contains(t, out, "kafka-dev")
	assert.Contains(t, out, "kafka-prod")
}

func TestByteRateHumanization(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want string
	}{
		{"bytes", 512, "512 B/s"},
		{"kib", 1536, "1.5 KiB/s"},
		{"mib", 1048576, "1.0 MiB/s"},
		{"unknown", -1, dash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, humanRate(tt.in))
		})
	}
}

func TestRenderShowsRatesAndDash(t *testing.T) {
	m := newModel(t)
	out := feed(m, sample())
	assert.Contains(t, out, "KiB/s") // 1536 humanized
	assert.Contains(t, out, dash)    // negative rate on kafka-prod
}

func TestLoadingStateBeforeFirstUpdate(t *testing.T) {
	m := newModel(t)
	out := m.renderContent(160, 40)
	assert.Contains(t, out, "Loading…")
	assert.False(t, m.loaded)
}

func TestEmptyState(t *testing.T) {
	m := newModel(t)
	out := feed(m, []api.ClusterOverview{})
	assert.Contains(t, out, "No clusters found")
}

func TestReadonlyBadge(t *testing.T) {
	m := newModel(t)
	out := feed(m, sample())
	assert.Contains(t, out, "readonly")
}

func TestOfflineRowSurfacesError(t *testing.T) {
	m := newModel(t)
	clusters := []api.ClusterOverview{
		{Name: "kafka-down", Status: api.ClusterOffline, LastError: "dial tcp: connection refused",
			BytesInPerSec: -1, BytesOutPerSec: -1, MessagesInPerSec: -1},
	}
	out := feed(m, clusters)
	// Offline cluster is selected (cursor 0) so its error shows in the detail area.
	assert.Contains(t, out, "connection refused")
}

func TestStatusCounts(t *testing.T) {
	tests := []struct {
		name                             string
		clusters                         []api.ClusterOverview
		wOnline, wOffline, wInitializing int
	}{
		{"empty", nil, 0, 0, 0},
		{"mixed", []api.ClusterOverview{
			{Status: api.ClusterOnline}, {Status: api.ClusterOnline},
			{Status: api.ClusterOffline},
			{Status: api.ClusterInitializing},
		}, 2, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			on, off, initz := statusCounts(tt.clusters)
			assert.Equal(t, tt.wOnline, on)
			assert.Equal(t, tt.wOffline, off)
			assert.Equal(t, tt.wInitializing, initz)
		})
	}
}

func TestHeaderShowsCounters(t *testing.T) {
	m := newModel(t)
	out := feed(m, sample())
	assert.Contains(t, out, "Online:")
	assert.Contains(t, out, "Offline:")
	assert.Contains(t, out, "Initializing:")
}

func TestOfflineFilterToggle(t *testing.T) {
	m := newModel(t)
	clusters := []api.ClusterOverview{
		{Name: "up-cluster", Status: api.ClusterOnline, BytesInPerSec: -1, BytesOutPerSec: -1, MessagesInPerSec: -1},
		{Name: "down-cluster", Status: api.ClusterOffline, LastError: "boom", BytesInPerSec: -1, BytesOutPerSec: -1, MessagesInPerSec: -1},
	}
	m.handle(cluster.ClusterStatsUpdatedMsg{Clusters: clusters})

	// Full list initially.
	assert.Len(t, m.visibleClusters(), 2)

	// Toggle offline-only via key 'o'.
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.True(t, m.offlineOnly)
	vis := m.visibleClusters()
	require.Len(t, vis, 1)
	assert.Equal(t, "down-cluster", vis[0].Name)

	out := m.renderContent(160, 40)
	assert.Contains(t, out, "[offline only]")
	assert.NotContains(t, out, "up-cluster")

	// Toggle back restores the full list.
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.False(t, m.offlineOnly)
	assert.Len(t, m.visibleClusters(), 2)
}

func TestHelpListsOfflineBinding(t *testing.T) {
	m := newModel(t)
	var keys []string
	for _, b := range m.GetHelp() {
		keys = append(keys, b.Help().Key)
	}
	assert.Contains(t, keys, "o")
	assert.Contains(t, keys, "enter")
	assert.Contains(t, keys, "r")
	assert.Contains(t, keys, "v")
}

func TestPageIdentity(t *testing.T) {
	m := newModel(t)
	assert.Equal(t, "clusters", m.GetID())
	assert.Equal(t, "Clusters", m.GetTitle())
}

// TestWithRealCollector exercises the collector-backed path end to end using the
// mock datasource (which reports several clusters with varied health).
func TestWithRealCollector(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	common := core.NewCommon(ds)
	col := cluster.New(ds, 0, nil)
	col.CollectAll(context.Background())
	common.Collector = col

	m := NewModelWithCommon(common)
	m.SetDimensions(160, 40)
	out := feed(m, nil) // collector cache is the source of truth

	require.True(t, m.loaded)
	require.NotEmpty(t, m.clusters)
	assert.Contains(t, out, "Name")
	assert.Contains(t, out, "kafka-")
}
