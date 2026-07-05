package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/template/ui/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexByText maps sidebar items by their Text label for easy lookup in tests.
func indexByText(items []providers.SidebarItem) map[string]providers.SidebarItem {
	m := make(map[string]providers.SidebarItem, len(items))
	for _, it := range items {
		m[it.Text] = it
	}
	return m
}

func f64(v float64) *float64 { return &v }
func i32(v int32) *int32     { return &v }

func TestBrokerResourceItem_GetValues(t *testing.T) {
	t.Run("controller marker, no stats yet", func(t *testing.T) {
		item := &BrokerResourceItem{info: api.BrokerInfo{ID: 1, Host: "h1", Port: 9092, IsController: true}}
		vals := item.GetValues()
		assert.Equal(t, []string{"1 ★", "h1", "9092", "…", "…", "…"}, vals)
	})
	t.Run("non-controller, no stats", func(t *testing.T) {
		item := &BrokerResourceItem{info: api.BrokerInfo{ID: 2, Host: "h2", Port: 9092}}
		vals := item.GetValues()
		assert.Equal(t, []string{"2", "h2", "9092", "…", "…", "…"}, vals)
	})
	t.Run("with stats", func(t *testing.T) {
		item := &BrokerResourceItem{info: api.BrokerInfo{ID: 3, Host: "h3", Port: 9093}}
		item.SetStats(api.BrokerStats{SegmentSize: 1073741824, SegmentCount: 2, InSyncReplicaCount: 5, ReplicaCount: 6, ReplicaSkew: f64(3.2)})
		vals := item.GetValues()
		assert.Equal(t, []string{"3", "h3", "9093", "1.00 GB, 2 segment(s)", "5/6", "3.20%"}, vals)
	})
}

func TestBrokerResourceItem_GetDetails(t *testing.T) {
	item := &BrokerResourceItem{info: api.BrokerInfo{ID: 1, Host: "h1", Port: 9092, Rack: "r", IsController: true}}
	d := item.GetDetails()
	assert.Equal(t, "1", d["ID"])
	assert.Equal(t, "Yes (Active Controller)", d["Controller"])

	item2 := &BrokerResourceItem{info: api.BrokerInfo{ID: 2}}
	assert.Equal(t, "No", item2.GetDetails()["Controller"])
}

func TestBrokerResource_GetData(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	res := NewBrokerResource(ds)
	assert.Equal(t, BrokerResourceType, res.GetType())

	items, err := res.GetData()
	require.NoError(t, err)
	require.Len(t, items, 3)

	// First broker is the controller.
	bri, ok := items[0].(*BrokerResourceItem)
	require.True(t, ok)
	assert.Equal(t, int32(1), bri.info.ID)
	assert.True(t, bri.info.IsController)
	assert.False(t, bri.HasStats())
}

func TestSortBrokerItems(t *testing.T) {
	mk := func(id int32, port int32, skew *float64) *BrokerResourceItem {
		b := &BrokerResourceItem{info: api.BrokerInfo{ID: id, Port: port}}
		b.SetStats(api.BrokerStats{ReplicaSkew: skew})
		return b
	}
	items := []interface{}{
		mk(3, 9094, f64(5.0)),
		mk(1, 9092, nil),
		mk(2, 9093, f64(1.0)),
	}

	sortBrokerItems(items, "id", false)
	assert.Equal(t, int32(1), items[0].(*BrokerResourceItem).info.ID)
	assert.Equal(t, int32(3), items[2].(*BrokerResourceItem).info.ID)

	sortBrokerItems(items, "id", true)
	assert.Equal(t, int32(3), items[0].(*BrokerResourceItem).info.ID)

	// Absent skew always sorts last, regardless of direction.
	sortBrokerItems(items, "skew", false)
	assert.Nil(t, items[len(items)-1].(*BrokerResourceItem).stats.ReplicaSkew)
	sortBrokerItems(items, "skew", true)
	assert.Nil(t, items[len(items)-1].(*BrokerResourceItem).stats.ReplicaSkew)
}

func TestBrokerSummaryItems(t *testing.T) {
	t.Run("healthy cluster", func(t *testing.T) {
		items := brokerSummaryItems(api.BrokerSummary{
			BrokerCount: 3, ControllerID: i32(1), ClusterVersion: "3.6", ControllerType: "KRaft",
			OnlinePartitions: 30, TotalPartitions: 30, UnderReplicated: 0,
			InSyncReplicas: 90, TotalReplicas: 90, OutOfSync: 0,
		})
		byText := indexByText(items)
		assert.Equal(t, "success", byText["Controller"].Status)
		assert.Equal(t, "#1", byText["Controller"].Value)
		assert.Equal(t, "success", byText["Online"].Status)
		assert.Equal(t, "success", byText["Under-repl"].Status)
		assert.Equal(t, "success", byText["In-Sync"].Status)
		assert.Equal(t, "KRaft", byText["Type"].Value)
	})
	t.Run("unhealthy cluster + no controller", func(t *testing.T) {
		items := brokerSummaryItems(api.BrokerSummary{
			BrokerCount: 3, ControllerID: nil,
			OnlinePartitions: 28, TotalPartitions: 30, UnderReplicated: 2,
			InSyncReplicas: 87, TotalReplicas: 90, OutOfSync: 3,
		})
		byText := indexByText(items)
		assert.Equal(t, "error", byText["Controller"].Status)
		assert.Equal(t, "No Active Controller", byText["Controller"].Value)
		assert.Equal(t, "error", byText["Online"].Status)
		assert.Equal(t, "error", byText["Under-repl"].Status)
		assert.Equal(t, "error", byText["In-Sync"].Status)
		assert.Equal(t, "error", byText["Out-of-Sync"].Status)
		assert.Equal(t, "Unknown", byText["Version"].Value)
		assert.Equal(t, "Unknown", byText["Type"].Value)
	})
}
