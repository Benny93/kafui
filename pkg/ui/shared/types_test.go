package shared

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestResourceListItem_Accessors(t *testing.T) {
	r := ResourceListItem{ResourceItem: &MinimalResourceItem{ID: "wrapped"}, Selected: true}
	assert.Equal(t, "wrapped", r.FilterValue())
	assert.Equal(t, "wrapped", r.GetID())
	assert.True(t, r.IsSelected())

	r.SetSelected(false)
	assert.False(t, r.IsSelected())
}

func TestTopicListItem_Accessors(t *testing.T) {
	tr := TopicListItem{Name: "orders", Selected: true}
	assert.Equal(t, "orders", tr.FilterValue())
	assert.Equal(t, "orders", tr.GetID())
	assert.True(t, tr.IsSelected())

	tr.SetSelected(false)
	assert.False(t, tr.IsSelected())
}

func TestConsumerGroupListItem_Accessors(t *testing.T) {
	c := ConsumerGroupListItem{GroupID: "group-1", Group: api.ConsumerGroup{Name: "group-1"}}
	assert.Equal(t, "group-1", c.FilterValue())
	assert.Equal(t, "group-1", c.GetID())
	assert.False(t, c.IsSelected())

	c.SetSelected(true)
	assert.True(t, c.IsSelected())
}

// A message filters and identifies by key when one exists, otherwise falls back
// to the value body.
func TestMessageListItem_FilterValue(t *testing.T) {
	withKey := MessageListItem{Message: api.Message{Key: "k1", Value: "v1"}}
	assert.Equal(t, "k1", withKey.FilterValue())
	assert.Equal(t, "k1", withKey.GetID())

	keyless := MessageListItem{Message: api.Message{Value: "v2"}}
	assert.Equal(t, "v2", keyless.FilterValue(), "no key: filter by value")
	assert.Equal(t, "", keyless.GetID(), "identity is still the key")

	keyless.SetSelected(true)
	assert.True(t, keyless.IsSelected())
}

func TestUIError_Error(t *testing.T) {
	// With a wrapped cause the message is prefixed by the cause text.
	cause := errors.New("connection refused")
	e := NewUIError(ErrorTypeConnection, "could not reach broker", cause)
	assert.Equal(t, "could not reach broker: connection refused", e.Error())
	assert.Equal(t, ErrorTypeConnection, e.Type)

	// Without a cause only the message is shown.
	bare := UIError{Message: "plain failure"}
	assert.Equal(t, "plain failure", bare.Error())
}

func TestCalculateContentDimensions(t *testing.T) {
	got := CalculateContentDimensions(200, 60, 35, 3, 3)
	assert.Equal(t, 200, got.Width)
	assert.Equal(t, 60, got.Height)
	assert.Equal(t, 165, got.ContentWidth, "total minus sidebar")
	assert.Equal(t, 54, got.ContentHeight, "total minus footer and header")
	assert.Equal(t, 35, got.SidebarWidth)

	// A cramped terminal is clamped up to the minimum content size.
	small := CalculateContentDimensions(40, 10, 35, 3, 3)
	assert.Equal(t, MinContentWidth, small.ContentWidth)
	assert.Equal(t, MinContentHeight, small.ContentHeight)
}

func TestIsValidDimensions(t *testing.T) {
	assert.True(t, IsValidDimensions(MinContentWidth, MinContentHeight), "at the minimum")
	assert.True(t, IsValidDimensions(120, 40))
	assert.False(t, IsValidDimensions(MinContentWidth-1, MinContentHeight))
	assert.False(t, IsValidDimensions(MinContentWidth, MinContentHeight-1))
}

func TestMinimalResourceItem(t *testing.T) {
	m := &MinimalResourceItem{ID: "fallback"}
	assert.Equal(t, "fallback", m.GetID())
	assert.Equal(t, []string{"fallback"}, m.GetValues())
	assert.Equal(t, map[string]string{"Name": "fallback"}, m.GetDetails())
}

// The concrete list-item types satisfy the shared behavioural interfaces.
func TestListItemsImplementInterfaces(t *testing.T) {
	var _ SelectableItem = &ResourceListItem{}
	var _ FilterableItem = ResourceListItem{}
	var _ SelectableItem = &TopicListItem{}
	var _ SelectableItem = &ConsumerGroupListItem{}
	var _ SelectableItem = &MessageListItem{}
}
