package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitialResourceAppliedSynchronously guards against bug #7 regressing: a
// CLI --resource deep-link must be reflected in both the content provider and
// the sidebar's active-resource highlight from the moment the main page is
// constructed — not via a separate async message that can race the page's own
// default (Topics) Init() and lose.
func TestInitialResourceAppliedSynchronously(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	common := core.NewCommon(ds)
	common.InitialResource = "consumer-groups"

	m := NewModelWithCommon(common)

	require.Equal(t, ConsumerGroupResourceType, m.contentProvider.currentResource.GetType())
	assert.Equal(t, ConsumerGroupResourceType, m.resourcesSection.currentResource,
		"sidebar highlight must match the deep-linked resource, not the Topics default")
	assert.Equal(t, "", common.InitialResource, "InitialResource should be consumed once")
}

// TestInitialResourceUnknownNameIgnored verifies an unrecognized --resource
// value leaves the default (Topics) view in place rather than erroring.
func TestInitialResourceUnknownNameIgnored(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	common := core.NewCommon(ds)
	common.InitialResource = "no-such-resource"

	m := NewModelWithCommon(common)

	assert.Equal(t, TopicResourceType, m.contentProvider.currentResource.GetType())
}
