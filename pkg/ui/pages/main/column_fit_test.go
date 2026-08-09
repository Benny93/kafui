package mainpage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Every resource layout must have at least one flex column, otherwise
// WithTargetWidth is a no-op and the table either under-fills the pane
// (wide terminal) or overflows and gets wrapped by the enclosing box (narrow).
func TestEveryResourceLayoutHasAFlexColumn(t *testing.T) {
	types := []ResourceType{
		TopicResourceType, ContextResourceType, SchemaResourceType,
		ConsumerGroupResourceType, ACLResourceType, QuotaResourceType,
		BrokerResourceType, ConnectorResourceType, ConnectClusterResourceType,
	}

	for _, rt := range types {
		t.Run(fmt.Sprint(rt), func(t *testing.T) {
			cols := createResourceTableColumns(rt)

			flex, fixed := 0, 0
			for _, c := range cols {
				if c.IsFlex() {
					flex++
				} else {
					fixed += c.Width()
				}
			}
			assert.Positive(t, flex, "needs a flex column to fit its pane")

			// An 80-column terminal leaves kafui a ~72-column content pane.
			// Fixed columns plus borders must stay well inside it so the flex
			// columns still get usable width instead of the row wrapping.
			assert.LessOrEqual(t, fixed+len(cols)+1, 60,
				"fixed columns leave too little room on an 80-column terminal")
		})
	}
}

func TestFirstColumnWidth(t *testing.T) {
	// Topics: 1 flex Name + fixed 11+12+10+5+10 = 48, 6 cols → 7 border cells.
	cols := createResourceTableColumns(TopicResourceType)
	assert.Equal(t, 135-7-48, firstColumnWidth(cols, 135))

	// A layout whose first column is fixed has no name budget to compute.
	assert.Equal(t, 0, firstColumnWidth(createResourceTableColumns(BrokerResourceType), 135))

	// Nothing left over must not report a negative width.
	assert.Equal(t, 0, firstColumnWidth(cols, 40))
}
