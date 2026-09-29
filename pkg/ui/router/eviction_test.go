package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// switchableDataSource lets a test change the active cluster.
type switchableDataSource struct {
	mockDataSource
	ctx string
}

func (s *switchableDataSource) GetContext() string { return s.ctx }
func (s *switchableDataSource) SetContext(name string) error {
	s.ctx = name
	return nil
}

func TestPagesLeavingHistoryAreEvicted(t *testing.T) {
	r, pages := newFakeRouter(t)

	r.NavigateTo("topic:a", nil)
	assert.NotContains(t, r.pages, "topic:b", "a page Back cannot reach must not stay cached")
	assert.True(t, pages["topic:b"].disposed)

	r.NavigateTo("detail:a:0:1", nil)
	r.Back()
	assert.Equal(t, "topic:a", r.GetCurrentPageID())
	assert.NotContains(t, r.pages, "detail:a:0:1", "every opened message stayed in memory")
	assert.Contains(t, r.pages, "topic:a")
	assert.False(t, pages["topic:a"].disposed)

	r.Back()
	assert.Equal(t, "main", r.GetCurrentPageID())
	assert.NotContains(t, r.pages, "topic:a")
	assert.True(t, pages["topic:a"].disposed)
	assert.Contains(t, r.pages, "main", "main is always kept")
}

// A cached topic:<name> page belongs to the cluster it was opened on. After a
// context switch, opening the same name must build a new page (ARCH-3).
func TestContextSwitchEvictsClusterPages(t *testing.T) {
	ds := &switchableDataSource{ctx: "A"}
	r, pages := newFakeRouterWith(t, ds)

	r.NavigateTo("topic:a", nil)
	r.NavigateTo("clusters", nil)
	assert.Equal(t, []string{"main", "topic:a"}, r.GetHistory())

	// The clusters page switches context, then returns to main.
	_ = ds.SetContext("B")
	r.NavigateTo("main", nil)

	assert.Equal(t, "main", r.GetCurrentPageID())
	assert.NotContains(t, r.pages, "topic:a", "the old cluster's topic page survived the switch")
	assert.True(t, pages["topic:a"].disposed)
	assert.NotContains(t, r.pages, "clusters")
	assert.Empty(t, r.GetHistory())
	assert.Same(t, pages["main"], r.pages["main"], "main handles the switch itself and is kept")
}

func TestContextSwitchKeepsMainBeneathTheNextPage(t *testing.T) {
	ds := &switchableDataSource{ctx: "A"}
	r, _ := newFakeRouterWith(t, ds)
	r.NavigateTo("topic:a", nil)
	r.Back()

	// main switches context in place, then the user opens a page.
	_ = ds.SetContext("B")
	r.NavigateTo("topic:b", nil)

	assert.Equal(t, []string{"main"}, r.GetHistory())
	r.Back()
	assert.Equal(t, "main", r.GetCurrentPageID())
}
