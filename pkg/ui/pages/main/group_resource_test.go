package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lp(v int64) *int64 { return &v }

func enrichedGroupItem(id, state string, members, topics int, lag *int64, coord int32) *ConsumerGroupResourceItem {
	c := &ConsumerGroupResourceItem{id: id}
	c.SetDetail(api.ConsumerGroup{
		Name: id, State: state, MemberCount: members, TopicCount: topics, Lag: lag, CoordinatorID: coord,
	})
	return c
}

func TestConsumerGroupResourceItem_Values(t *testing.T) {
	t.Run("unenriched shows placeholders", func(t *testing.T) {
		c := &ConsumerGroupResourceItem{id: "g1"}
		assert.Equal(t, []string{"g1", "…", "…", "…", "…", "…"}, c.GetValues())
		assert.Equal(t, api.GroupStateUnknown, c.State())
	})
	t.Run("enriched with lag", func(t *testing.T) {
		c := enrichedGroupItem("g1", api.GroupStateStable, 3, 2, lp(42), 1)
		assert.Equal(t, []string{"g1", "Stable", "3", "2", "42", "1"}, c.GetValues())
		d := c.GetDetails()
		assert.Equal(t, "Stable", d["State"])
		assert.Equal(t, "42", d["Lag"])
	})
	t.Run("undefined lag renders dash, never zero", func(t *testing.T) {
		c := enrichedGroupItem("g1", api.GroupStateEmpty, 0, 1, nil, -1)
		vals := c.GetValues()
		assert.Equal(t, "—", vals[4]) // lag
		assert.Equal(t, "—", vals[5]) // coordinator unknown
	})
}

func TestFormatGroupLag(t *testing.T) {
	assert.Equal(t, "—", formatGroupLag(nil))
	assert.Equal(t, "0", formatGroupLag(lp(0)))
	assert.Equal(t, "7", formatGroupLag(lp(7)))
}

func TestSortGroupItems(t *testing.T) {
	items := []interface{}{
		shared.ResourceListItem{ResourceItem: enrichedGroupItem("c", api.GroupStateEmpty, 1, 1, lp(5), 1)},
		shared.ResourceListItem{ResourceItem: enrichedGroupItem("a", api.GroupStateStable, 3, 2, nil, 1)},
		shared.ResourceListItem{ResourceItem: enrichedGroupItem("b", api.GroupStateDead, 0, 1, lp(10), 1)},
	}

	nameOf := func(it interface{}) string {
		c, _, _ := groupItemFrom(it)
		return c.id
	}

	// name asc
	sortGroupItems(items, "name", false)
	assert.Equal(t, "a", nameOf(items[0]))
	assert.Equal(t, "c", nameOf(items[2]))

	// state priority: Stable(0) < Empty(3) < Dead(4)
	sortGroupItems(items, "state", false)
	assert.Equal(t, api.GroupStateStable, mustGroup(items[0]).State())
	assert.Equal(t, api.GroupStateDead, mustGroup(items[2]).State())

	// lag asc: nil treated as 0, so "a" (nil) first
	sortGroupItems(items, "lag", false)
	assert.Equal(t, "a", nameOf(items[0]))
	assert.Equal(t, "b", nameOf(items[2])) // lag 10 largest
}

func mustGroup(it interface{}) *ConsumerGroupResourceItem {
	c, _, _ := groupItemFrom(it)
	return c
}

func TestGroupItemMatchesState(t *testing.T) {
	k := NewKafuiContentProvider(newMockDS())
	stable := shared.ResourceListItem{ResourceItem: enrichedGroupItem("a", api.GroupStateStable, 1, 1, nil, 1)}
	empty := shared.ResourceListItem{ResourceItem: enrichedGroupItem("b", api.GroupStateEmpty, 0, 1, nil, 1)}
	unloaded := shared.ResourceListItem{ResourceItem: &ConsumerGroupResourceItem{id: "c"}}

	assert.True(t, k.groupItemMatchesState(stable, api.GroupStateStable))
	assert.False(t, k.groupItemMatchesState(stable, api.GroupStateEmpty))
	assert.True(t, k.groupItemMatchesState(empty, api.GroupStateEmpty))
	// Unloaded rows are treated as Unknown.
	assert.True(t, k.groupItemMatchesState(unloaded, api.GroupStateUnknown))
}

func newMockDS() *mock.KafkaDataSourceMock {
	m := &mock.KafkaDataSourceMock{}
	m.Init("")
	return m
}

// spyDS records the arguments passed to GetConsumerGroupDetails so tests can
// assert that only the visible page of names is enriched.
type spyDS struct {
	*mock.KafkaDataSourceMock
	lastDetailArgs []string
}

func (s *spyDS) GetConsumerGroupDetails(ids []string) ([]api.ConsumerGroup, error) {
	s.lastDetailArgs = append([]string(nil), ids...)
	return s.KafkaDataSourceMock.GetConsumerGroupDetails(ids)
}

func TestLoadGroupDetails_VisiblePageOnly(t *testing.T) {
	spy := &spyDS{KafkaDataSourceMock: newMockDS()}
	k := NewKafuiContentProvider(spy)
	k.switchResource(SwitchResourceMsg(ConsumerGroupResourceType))

	// Load the group names (fast phase).
	cmd := k.loadCurrentResource()
	msg := cmd()
	list, ok := msg.(CurrentResourceListMsg)
	require.True(t, ok)
	k.handleResourceList(list)

	// Small page so only a subset of rows is visible.
	k.pagination.PerPage = 3
	k.pagination.SetTotalItems(len(k.allItems))
	k.updateTableForCurrentPage()

	detailCmd := k.loadGroupDetails()
	require.NotNil(t, detailCmd)
	detailMsg := detailCmd()

	// Only the visible page's names were requested for enrichment.
	assert.Lenf(t, spy.lastDetailArgs, 3, "should enrich only the 3 visible rows, got %v", spy.lastDetailArgs)

	// Applying details flips the placeholder to real values.
	loaded, ok := detailMsg.(ConsumerGroupDetailsLoadedMsg)
	require.True(t, ok)
	k.applyGroupDetails(loaded)

	// analytics-consumer sorts first (visible) and is a rich mock group; after
	// enrichment it is loaded with its real Stable state.
	for _, item := range k.allItems {
		c, _, ok := groupItemFrom(item)
		if !ok {
			continue
		}
		if c.id == "analytics-consumer" {
			assert.True(t, c.DetailsLoaded())
			assert.Equal(t, api.GroupStateStable, c.State())
		}
	}
}
