package core

import (
	"errors"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every message constructor returns a tea.Cmd whose closure yields the typed
// message. These are the shell's vocabulary, so each must round-trip its fields.

func TestNewNotification(t *testing.T) {
	msg := NewNotification(StatusWarning, "title", "body")()
	n, ok := msg.(NotificationMsg)
	require.True(t, ok)
	assert.Equal(t, StatusWarning, n.Severity)
	assert.Equal(t, "title", n.Title)
	assert.Equal(t, "body", n.Message)
}

func TestNotifyError(t *testing.T) {
	err := errors.New("boom")
	n := NotifyError("t", err)().(NotificationMsg)
	assert.Equal(t, StatusError, n.Severity)
	assert.Equal(t, "t", n.Title)
	assert.Equal(t, "boom", n.Message)

	// A nil error still yields a notification, just with an empty body.
	n = NotifyError("t", nil)().(NotificationMsg)
	assert.Equal(t, "", n.Message)
}

func TestCommonMessageConstructors(t *testing.T) {
	e := errors.New("x")

	assert.Equal(t, DataLoadedMsg{Type: "t", Data: 1}, NewDataLoadedMsg("t", 1)())
	assert.Equal(t, DataErrorMsg{Type: "t", Error: e}, NewDataErrorMsg("t", e)())
	assert.Equal(t, StatusMsg{Message: "m", Type: StatusSuccess}, NewStatusMsg("m", StatusSuccess)())
	assert.Equal(t, PageChangeMsg{PageID: "p", Data: "d"}, NewPageChangeMsg("p", "d")())
	assert.Equal(t,
		ResourceSelectedMsg{ResourceID: "id", ResourceType: "topic", Item: "it"},
		NewResourceSelectedMsg("id", "topic", "it")())
}

func TestTypedLoadMessageConstructors(t *testing.T) {
	e := errors.New("load failed")

	topics := map[string]api.Topic{"a": {NumPartitions: 2}}
	assert.Equal(t, TopicsLoadedMsg{Topics: topics}, NewTopicsLoadedMsg(topics)())
	assert.Equal(t, TopicsLoadErrorMsg{Error: e}, NewTopicsLoadErrorMsg(e)())

	groups := []api.ConsumerGroup{{Name: "g"}}
	assert.Equal(t, ConsumerGroupsLoadedMsg{Groups: groups}, NewConsumerGroupsLoadedMsg(groups)())
	assert.Equal(t, ConsumerGroupsLoadErrorMsg{Error: e}, NewConsumerGroupsLoadErrorMsg(e)())

	msgs := []api.Message{{Offset: 3}}
	assert.Equal(t, MessagesConsumedMsg{Messages: msgs}, NewMessagesConsumedMsg(msgs)())
	assert.Equal(t, MessageConsumeErrorMsg{Error: e}, NewMessageConsumeErrorMsg(e)())

	schemas := []api.SchemaInfo{{Subject: "s"}}
	assert.Equal(t, SchemasLoadedMsg{Schemas: schemas}, NewSchemasLoadedMsg(schemas)())
	assert.Equal(t, SchemasLoadErrorMsg{Error: e}, NewSchemasLoadErrorMsg(e)())

	ctxs := []string{"prod"}
	assert.Equal(t, ContextsLoadedMsg{Contexts: ctxs}, NewContextsLoadedMsg(ctxs)())
	assert.Equal(t, ContextsLoadErrorMsg{Error: e}, NewContextsLoadErrorMsg(e)())
}
