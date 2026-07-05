package mainpage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourcesSectionGatingDefault(t *testing.T) {
	// With no Common (nil), every resource is enabled (safe default before
	// capabilities are known) and RenderItems lists the optional sections.
	r := NewResourcesSection(nil)
	for _, rt := range []ResourceType{TopicResourceType, ConsumerGroupResourceType, SchemaResourceType, ContextResourceType, ACLResourceType} {
		assert.True(t, r.enabled(rt), "resource %v should be enabled by default", rt)
	}

	var names []string
	for _, it := range r.RenderItems(10, 40) {
		names = append(names, it.Text)
	}
	assert.Contains(t, names, "Schemas")
	assert.Contains(t, names, "ACLs")
}
