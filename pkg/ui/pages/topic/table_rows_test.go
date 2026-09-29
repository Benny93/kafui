package topic

import (
	"fmt"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Several partitions share offsets; the highlighted row must be the message
// Enter opens.
func TestRenderedRowMatchesSelection(t *testing.T) {
	m := NewModel(&MockDataSource{}, "test-topic", api.Topic{})
	for p := int32(0); p < 3; p++ {
		for o := int64(0); o < 8; o++ {
			m.addMessageInternal(api.Message{Partition: p, Offset: o, Key: fmt.Sprintf("k-p%d-o%d", p, o)})
		}
	}
	m.sortMessages()
	m.pagination.SortOrder = "newest_first"
	m.pagination.SetPerPage(12)
	m.pagination.SetTotalMessages(len(m.filteredMessages))
	m.updateMessageTable()

	m.renderTableCustom(200, 40)

	require.NotEmpty(t, m.rowStringCache)
	for row, line := range m.rowStringCache {
		m.cursorRow = row
		sel := m.GetSelectedMessage()
		require.NotNil(t, sel)
		assert.Contains(t, line, sel.Key, "row %d", row)
	}
}

func TestPageSizeMatchesRenderedRows(t *testing.T) {
	for _, height := range []int{20, 36, 56} {
		t.Run(fmt.Sprintf("height %d", height), func(t *testing.T) {
			m := NewModel(&MockDataSource{}, "test-topic", api.Topic{})
			for o := int64(0); o < 200; o++ {
				m.addMessageInternal(api.Message{Offset: o})
			}
			m.sortMessages()
			m.pagination.SetTotalMessages(len(m.filteredMessages))
			p := NewTopicContentProvider(m)

			p.RenderContent(160, height)

			assert.Equal(t, m.tableRowBudget(height), m.pagination.PerPage)
			assert.Len(t, m.rowStringCache, m.pagination.PerPage, "every row of the page is drawn")
		})
	}
}
