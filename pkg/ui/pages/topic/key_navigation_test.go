package topic

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// navModel returns a topic model seeded with count messages at offsets
// base, base+1, ... and a page size of perPage.
func navModel(t *testing.T, count, perPage int, base int64) *Model {
	t.Helper()
	m := createTestModel()
	m.pagination.SetPerPage(perPage)
	for i := 0; i < count; i++ {
		m.AddMessage(api.Message{Offset: base + int64(i)})
	}
	// AddMessage buffers only; the fetch pipeline syncs pagination totals.
	m.filteredMessages = m.messages
	m.pagination.SetTotalMessages(len(m.filteredMessages))
	m.consumption = NewConsumptionController(m)
	return m
}

func TestHandleNavigation_Up(t *testing.T) {
	m := navModel(t, 10, 5, 0)
	k := NewKeys()

	cmd := k.handleNavigation(m, "up")
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.cursorRow, "clamped at the top")

	m.cursorRow = 3
	before := m.renderVersion
	k.handleNavigation(m, "up")
	assert.Equal(t, 2, m.cursorRow)
	assert.Greater(t, m.renderVersion, before, "cursor moves dirty the cached view")
}

func TestHandleNavigation_Down(t *testing.T) {
	m := navModel(t, 10, 5, 0)
	k := NewKeys()

	k.handleNavigation(m, "down")
	assert.Equal(t, 1, m.cursorRow, "one press moves one row")

	// Walk to the bottom of the 5-row page, then press once more.
	for i := 0; i < 6; i++ {
		k.handleNavigation(m, "down")
	}
	assert.Equal(t, 4, m.cursorRow, "clamped at the last visible row")
}

func TestHandleNavigation_PageUpAndDown(t *testing.T) {
	// Offsets start at 100 so older batches always exist for fetch-more.
	m := navModel(t, 10, 5, 100)
	k := NewKeys()

	// Forward paging: page 0 → 1, cursor reset via pendingReset.
	cmd := k.handleNavigation(m, "pagedown")
	assert.Nil(t, cmd)
	assert.Equal(t, 1, m.pagination.Page)
	assert.Equal(t, 0, m.cursorRow)

	// Last page, not loading, older messages exist → fetch a batch (the Cmd is
	// only constructed here, not executed, so no datasource is touched).
	m.consumeMode = ModeNewest
	cmd = k.handleNavigation(m, "pagedown")
	require.NotNil(t, cmd, "pagedown on the last page requests the next batch")

	// While loading, the fetch-more must not be re-issued.
	m.loading = true
	cmd = k.handleNavigation(m, "pagedown")
	assert.Nil(t, cmd, "no second fetch while one is in flight")
	m.loading = false

	// Already at offset 0: nowhere older to go.
	m2 := navModel(t, 1, 1, 0) // single message at offset 0
	cmd = k.handleNavigation(m2, "pagedown")
	assert.Nil(t, cmd, "at the start of the topic there is no older batch")

	// Back paging: page 1 → 0, cursor reset.
	m.cursorRow = 3
	cmd = k.handleNavigation(m, "pageup")
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.pagination.Page)
	assert.Equal(t, 0, m.cursorRow)

	// On the first page, pageup is a no-op.
	cmd = k.handleNavigation(m, "pageup")
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.pagination.Page)
}

func TestHandleNavigation_HomeAndEnd(t *testing.T) {
	m := navModel(t, 12, 4, 0)
	k := NewKeys()
	m.pagination.NextPage()
	m.cursorRow = 2

	cmd := k.handleNavigation(m, "end")
	assert.Nil(t, cmd)
	assert.Equal(t, m.pagination.TotalPages-1, m.pagination.Page)
	assert.Equal(t, 0, m.cursorRow, "jumping pages resets the cursor")

	m.cursorRow = 1
	cmd = k.handleNavigation(m, "home")
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.pagination.Page)
	assert.Equal(t, 0, m.cursorRow)
}

func TestHandleNavigation_UnknownDirectionIsNoop(t *testing.T) {
	m := navModel(t, 5, 5, 0)
	before := m.renderVersion

	cmd := NewKeys().handleNavigation(m, "sideways")
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.cursorRow)
	assert.Equal(t, before, m.renderVersion)
}
