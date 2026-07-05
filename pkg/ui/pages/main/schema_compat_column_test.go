package mainpage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// SR-13: the schemas listing carries a Compatibility column sourced from the
// lazy details merge.
func TestSchemaResourceItemCompatibilityColumn(t *testing.T) {
	loaded := &SchemaResourceItem{
		id:            "orders-value",
		subject:       "orders-value",
		version:       3,
		schemaID:      103,
		schemaType:    "AVRO",
		compatibility: "FULL",
		detailsLoaded: true,
	}
	values := loaded.GetValues()
	assert.Equal(t, []string{"orders-value", "3", "103", "AVRO", "FULL"}, values)
	assert.Equal(t, "FULL", loaded.GetDetails()["Compatibility"])

	// Before details load, the column shows a placeholder.
	pending := &SchemaResourceItem{id: "x", subject: "x"}
	assert.Equal(t, []string{"x", "…", "…", "…", "…"}, pending.GetValues())
	assert.Equal(t, "…", pending.GetDetails()["Compatibility"])

	// The schemas resource defines a Compatibility column.
	cols := createResourceTableColumns(SchemaResourceType)
	var titles []string
	for _, c := range cols {
		titles = append(titles, c.Title())
	}
	assert.Contains(t, titles, "Compatibility")
}
