package mainpage

import (
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func topicRLI(tri *TopicResourceItem) interface{} {
	return shared.ResourceListItem{ResourceItem: tri}
}

func sp(v string) *string { return &v }

// topicSpyDS records the arguments passed to the mutating topic operations.
type topicSpyDS struct {
	*mock.KafkaDataSourceMock
	createName    string
	createParts   int32
	createRF      int16
	createConfigs map[string]*string
	deleteCalls   []string
	recreateCalls []string
	purgeCalls    []string
}

func (s *topicSpyDS) CreateTopic(name string, p int32, rf int16, cfg map[string]*string) error {
	s.createName, s.createParts, s.createRF, s.createConfigs = name, p, rf, cfg
	return s.KafkaDataSourceMock.CreateTopic(name, p, rf, cfg)
}

func (s *topicSpyDS) DeleteTopic(name string) error {
	s.deleteCalls = append(s.deleteCalls, name)
	return s.KafkaDataSourceMock.DeleteTopic(name)
}

func (s *topicSpyDS) RecreateTopic(name string) error {
	s.recreateCalls = append(s.recreateCalls, name)
	return s.KafkaDataSourceMock.RecreateTopic(name)
}

func (s *topicSpyDS) PurgeTopicMessages(name string, p int32) error {
	s.purgeCalls = append(s.purgeCalls, name)
	return s.KafkaDataSourceMock.PurgeTopicMessages(name, p)
}

func newTopicSpy() *topicSpyDS {
	return &topicSpyDS{KafkaDataSourceMock: newMockDS()}
}

// loadTopicsInto runs the quick topic load so allItems holds topic stubs.
func loadTopicsInto(t *testing.T, k *KafuiContentProvider) {
	t.Helper()
	cmd := k.loadCurrentResource()
	require.NotNil(t, cmd)
	msg := cmd()
	list, ok := msg.(CurrentResourceListMsg)
	require.True(t, ok, "expected CurrentResourceListMsg, got %T", msg)
	k.handleResourceList(list)
}

// highlightTopic points the table at the topic with the given name.
func highlightTopic(t *testing.T, k *KafuiContentProvider, name string) {
	t.Helper()
	for i, item := range k.allItems {
		if k.getItemID(item) == name {
			k.resourcesTable = k.resourcesTable.WithHighlightedRow(i)
			return
		}
	}
	t.Fatalf("topic %q not found in allItems", name)
}

// --- TP-14: row rendering ---

func TestTopicRowData_NACells(t *testing.T) {
	// Extended fetch completed but no value → "N/A" (not the loading placeholder).
	tri := &TopicResourceItem{id: "t", partitions: 3, replicationFactor: 2, messageCount: -1, size: -1, outOfSync: -1, detailsExtLoaded: true}
	row := topicRowData(tri, "", "…", 0)
	assert.Equal(t, "N/A", row[colTopicMessages])
	assert.Equal(t, "N/A", row[colTopicSize])
	assert.Equal(t, "N/A", row[colTopicOSR])
}

func TestTopicRowData_LoadingPlaceholder(t *testing.T) {
	tri := &TopicResourceItem{id: "t", partitions: -1, replicationFactor: -1, messageCount: -1, size: -1, outOfSync: -1}
	row := topicRowData(tri, "", "…", 0)
	assert.Equal(t, "…", row[colTopicMessages])
	assert.Equal(t, "…", row[colTopicSize])
	assert.Equal(t, "…", row[colTopicOSR])
	assert.Equal(t, "…", row[colTopicPartitions])
}

func TestTopicRowData_OSRStyledAndSize(t *testing.T) {
	// Force a colour profile so the alert styling emits ANSI codes even without a
	// TTY; otherwise lipgloss renders plain text and the styled/plain cells match.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	tri := &TopicResourceItem{id: "t", partitions: 3, replicationFactor: 2, messageCount: 5, size: 2048, outOfSync: 2, detailsExtLoaded: true}
	row := topicRowData(tri, "", "…", 0)

	osr := row[colTopicOSR].(string)
	assert.Contains(t, osr, "2")
	assert.NotEqual(t, "2", osr, "OSR > 0 should be styled (differs from plain value)")

	assert.Equal(t, "2.00 KB", row[colTopicSize])
	assert.Equal(t, "5", row[colTopicMessages])
}

// --- TP-15: sorting ---

func TestSortTopicItems(t *testing.T) {
	nameOf := func(it interface{}) string { tri, _, _ := topicItemFrom(it); return tri.id }
	fresh := func() []interface{} {
		return []interface{}{
			topicRLI(&TopicResourceItem{id: "b", partitions: 2, outOfSync: 1, replicationFactor: 3, messageCount: 50, size: 100}),
			topicRLI(&TopicResourceItem{id: "a", partitions: 5, outOfSync: 0, replicationFactor: 1, messageCount: 10, size: 300}),
			topicRLI(&TopicResourceItem{id: "c", partitions: 1, outOfSync: 2, replicationFactor: 2, messageCount: 99, size: 200}),
		}
	}

	items := fresh()
	sortTopicItems(items, "name", false)
	assert.Equal(t, []string{"a", "b", "c"}, []string{nameOf(items[0]), nameOf(items[1]), nameOf(items[2])})

	items = fresh()
	sortTopicItems(items, "partitions", false)
	assert.Equal(t, []string{"c", "b", "a"}, []string{nameOf(items[0]), nameOf(items[1]), nameOf(items[2])})

	items = fresh()
	sortTopicItems(items, "partitions", true)
	assert.Equal(t, []string{"a", "b", "c"}, []string{nameOf(items[0]), nameOf(items[1]), nameOf(items[2])})

	items = fresh()
	sortTopicItems(items, "messages", false)
	assert.Equal(t, []string{"a", "b", "c"}, []string{nameOf(items[0]), nameOf(items[1]), nameOf(items[2])})

	items = fresh()
	sortTopicItems(items, "size", false)
	assert.Equal(t, []string{"b", "c", "a"}, []string{nameOf(items[0]), nameOf(items[1]), nameOf(items[2])})

	items = fresh()
	sortTopicItems(items, "osr", false)
	assert.Equal(t, []string{"a", "b", "c"}, []string{nameOf(items[0]), nameOf(items[1]), nameOf(items[2])})
}

// --- TP-16: internal classification + visibility toggle ---

func TestIsInternalTopicName(t *testing.T) {
	assert.True(t, isInternalTopicName("__consumer_offsets"))
	assert.True(t, isInternalTopicName("__transaction_state"))
	assert.False(t, isInternalTopicName("user-events"))
	assert.False(t, isInternalTopicName("_single-underscore"))
}

func TestToggleHideInternal_ExcludesAndResetsPage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	k := NewKafuiContentProvider(newMockDS())
	k.allItems = []interface{}{
		topicRLI(&TopicResourceItem{id: "n1", isInternal: false}),
		topicRLI(&TopicResourceItem{id: "__i1", isInternal: true}),
		topicRLI(&TopicResourceItem{id: "n2", isInternal: false}),
		topicRLI(&TopicResourceItem{id: "__i2", isInternal: true}),
	}
	k.pagination.SetTotalItems(len(k.allItems))
	k.pagination.Page = 1 // simulate being off the first page

	cmd := k.toggleHideInternal()
	assert.Nil(t, cmd)
	assert.True(t, k.hideInternal)
	assert.True(t, k.isFiltered)
	assert.Len(t, k.filteredItems, 2, "internal topics excluded from the filtered view")
	assert.Equal(t, 0, k.pagination.Page, "page resets to 0 on toggle")

	// Persisted across a fresh provider construction.
	k2 := NewKafuiContentProvider(newMockDS())
	assert.True(t, k2.hideInternal)

	// Toggling off restores all items.
	k.toggleHideInternal()
	assert.False(t, k.hideInternal)
	assert.False(t, k.isFiltered)
}

// --- TP-18: create form → CreateTopic ---

func TestHandleTopicFormSubmit_CreateArgsOmitEmptyConfigs(t *testing.T) {
	spy := newTopicSpy()
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)

	cmd := k.handleTopicFormSubmit(map[string]string{
		"name":                "brand-new-topic",
		"partitions":          "3",
		"replication_factor":  "",
		"cleanup.policy":      "delete",
		"retention.ms":        "",
		"max.message.bytes":   "",
		"min.insync.replicas": "",
	})
	require.NotNil(t, cmd)
	msg := cmd()
	created, ok := msg.(topicCreatedMsg)
	require.True(t, ok)
	require.NoError(t, created.err)

	assert.Equal(t, "brand-new-topic", spy.createName)
	assert.Equal(t, int32(3), spy.createParts)
	assert.Equal(t, int16(-1), spy.createRF, "empty replication factor → cluster default (-1)")
	// Only the non-empty config survives; the empty ones are omitted.
	require.Len(t, spy.createConfigs, 1)
	require.Contains(t, spy.createConfigs, "cleanup.policy")
	assert.Equal(t, "delete", *spy.createConfigs["cleanup.policy"])
	assert.NotContains(t, spy.createConfigs, "retention.ms")

	names, err := spy.GetTopicNames()
	require.NoError(t, err)
	assert.Contains(t, names, "brand-new-topic")
}

func TestOpenCreateTopicForm_SetsInputMode(t *testing.T) {
	k := NewKafuiContentProvider(newMockDS())
	loadTopicsInto(t, k)
	k.openCreateTopicForm()
	assert.True(t, k.showTopicForm)
	assert.True(t, k.IsInputMode())
	require.NotNil(t, k.topicForm)
}

// --- TP-19: clone prefill ---

func TestOpenCloneTopicForm_Prefill(t *testing.T) {
	ds := newMockDS()
	require.NoError(t, ds.CreateTopic("clone-src", 5, 2, map[string]*string{
		"cleanup.policy": sp("compact"),
		"retention.ms":   sp("1000"),
	}))

	k := NewKafuiContentProvider(ds)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "clone-src")

	k.openCloneTopicForm()
	require.True(t, k.showTopicForm)
	require.NotNil(t, k.topicForm)

	vals := k.topicForm.Values()
	assert.Equal(t, "clone-src", vals["name"])
	assert.Equal(t, "5", vals["partitions"])
	assert.Equal(t, "2", vals["replication_factor"])
	assert.Equal(t, "compact", vals["cleanup.policy"])
	assert.Equal(t, "1000", vals["retention.ms"])
	// A config equal to its default is not copied.
	assert.Equal(t, "", vals["max.message.bytes"])
}

func TestOpenCloneTopicForm_DisabledOnMultiSelect(t *testing.T) {
	k := NewKafuiContentProvider(newMockDS())
	loadTopicsInto(t, k)
	k.selected = map[string]bool{"a": true, "b": true}

	cmd := k.openCloneTopicForm()
	require.NotNil(t, cmd)
	_, isNotification := cmd().(core.NotificationMsg)
	assert.True(t, isNotification, "multi-select clone should return a status hint")
	assert.False(t, k.showTopicForm)
}

// --- TP-20: delete / recreate ---

func TestDeleteSelectedTopic_ConfirmThenDelete(t *testing.T) {
	// The mock's topic store is process-global, so operate on a topic this test
	// owns rather than a shared fixture.
	spy := newTopicSpy()
	require.NoError(t, spy.KafkaDataSourceMock.CreateTopic("t-del-one", 1, 1, nil))
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "t-del-one")

	cmd := k.deleteSelectedTopics()
	require.NotNil(t, cmd)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok, "delete should request confirmation")
	assert.Contains(t, confirm.Message, "t-del-one")
	assert.Empty(t, spy.deleteCalls, "no delete before confirmation")

	res := confirm.OnConfirm()
	deleted, ok := res.(topicDeletedMsg)
	require.True(t, ok)
	assert.NoError(t, deleted.err)
	assert.Equal(t, []string{"t-del-one"}, spy.deleteCalls)
}

func TestDeleteSelectedTopic_DeletionDisabled(t *testing.T) {
	spy := newTopicSpy()
	require.NoError(t, spy.KafkaDataSourceMock.CreateTopic("t-del-disabled", 1, 1, nil))
	spy.SetDeletionDisabled(true)
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "t-del-disabled")

	cmd := k.deleteSelectedTopics()
	require.NotNil(t, cmd)
	msg := cmd()
	_, isConfirm := msg.(core.ShowConfirmMsg)
	assert.False(t, isConfirm, "deletion-disabled must not show a confirm modal")
	_, isNotification := msg.(core.NotificationMsg)
	assert.True(t, isNotification, "expected a status hint")
	assert.Empty(t, spy.deleteCalls)
}

func TestDeleteSelectedTopic_InternalHint(t *testing.T) {
	ds := newMockDS()
	require.NoError(t, ds.CreateTopic("__internal-x", 1, 1, nil))
	spy := &topicSpyDS{KafkaDataSourceMock: ds}
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "__internal-x")

	msg := k.deleteSelectedTopics()()
	_, isConfirm := msg.(core.ShowConfirmMsg)
	assert.False(t, isConfirm)
	assert.Empty(t, spy.deleteCalls)
}

func TestRecreateSelectedTopic_ConfirmThenRecreate(t *testing.T) {
	spy := newTopicSpy()
	require.NoError(t, spy.KafkaDataSourceMock.CreateTopic("t-recreate", 1, 1, nil))
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "t-recreate")

	confirm, ok := k.recreateSelectedTopic()().(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.Contains(t, confirm.Message, "t-recreate")

	res := confirm.OnConfirm()
	recreated, ok := res.(topicRecreatedMsg)
	require.True(t, ok)
	assert.NoError(t, recreated.err)
	assert.Equal(t, []string{"t-recreate"}, spy.recreateCalls)
}

// --- TP-21: purge / clear messages ---

func TestPurgeSelectedTopic_ConfirmThenPurge(t *testing.T) {
	spy := newTopicSpy()
	require.NoError(t, spy.KafkaDataSourceMock.CreateTopic("t-purge", 1, 1, nil)) // default cleanup.policy = delete
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "t-purge")

	confirm, ok := k.purgeSelectedTopics()().(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.Contains(t, confirm.Message, "t-purge")

	res := confirm.OnConfirm()
	purged, ok := res.(topicPurgedMsg)
	require.True(t, ok)
	assert.NoError(t, purged.err)
	assert.Equal(t, []string{"t-purge"}, spy.purgeCalls)
}

func TestPurgeSelectedTopic_CompactPolicyHint(t *testing.T) {
	ds := newMockDS()
	require.NoError(t, ds.CreateTopic("compact-topic", 1, 1, map[string]*string{"cleanup.policy": sp("compact")}))
	spy := &topicSpyDS{KafkaDataSourceMock: ds}
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)
	highlightTopic(t, k, "compact-topic")

	assert.False(t, k.topicAllowsDelete("compact-topic"))

	msg := k.purgeSelectedTopics()()
	_, isConfirm := msg.(core.ShowConfirmMsg)
	assert.False(t, isConfirm, "compact-only topic must not show a purge confirm")
	assert.Empty(t, spy.purgeCalls)
}

// --- TP-22: multi-select + batch ---

func TestToggleTopicSelection_ExcludesInternal(t *testing.T) {
	ds := newMockDS()
	require.NoError(t, ds.CreateTopic("__internal-y", 1, 1, nil))
	k := NewKafuiContentProvider(ds)
	loadTopicsInto(t, k)

	require.NoError(t, ds.CreateTopic("t-sel-normal", 1, 1, nil))
	loadTopicsInto(t, k) // reload so the new topic is present

	highlightTopic(t, k, "__internal-y")
	k.toggleTopicSelection()
	assert.Empty(t, k.selected, "internal topics are not selectable")

	highlightTopic(t, k, "t-sel-normal")
	k.toggleTopicSelection()
	assert.Len(t, k.selected, 1)
	assert.True(t, k.selected["t-sel-normal"])

	// Toggling again removes it.
	k.toggleTopicSelection()
	assert.Empty(t, k.selected)
}

func TestBatchDelete_CallsPerTopicAndClears(t *testing.T) {
	spy := newTopicSpy()
	require.NoError(t, spy.KafkaDataSourceMock.CreateTopic("t-batch-a", 1, 1, nil))
	require.NoError(t, spy.KafkaDataSourceMock.CreateTopic("t-batch-b", 1, 1, nil))
	k := NewKafuiContentProvider(spy)
	loadTopicsInto(t, k)

	k.selected = map[string]bool{"t-batch-a": true, "t-batch-b": true}

	cmd := k.deleteSelectedTopics() // >= 2 selected → batch
	require.NotNil(t, cmd)
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.Contains(t, confirm.Message, "2 topics")

	res := confirm.OnConfirm()
	batch, ok := res.(topicBatchResultMsg)
	require.True(t, ok)
	assert.Equal(t, 2, batch.total)
	assert.Empty(t, batch.failures)
	assert.ElementsMatch(t, []string{"t-batch-a", "t-batch-b"}, spy.deleteCalls)

	// The result handler clears the selection.
	k.HandleContentUpdate(batch)
	assert.Empty(t, k.selected)
}

// --- key routing sanity ---

func TestTopicKeyRouting_SortAndSelect(t *testing.T) {
	k := NewKafuiContentProvider(newMockDS())
	loadTopicsInto(t, k)

	// 's' cycles the sort column.
	k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	assert.Equal(t, 0, k.topicSortCol)

	// space toggles selection on the highlighted (non-internal) row.
	require.NoError(t, k.dataSource.(*mock.KafkaDataSourceMock).CreateTopic("t-route-one", 1, 1, nil))
	loadTopicsInto(t, k)
	highlightTopic(t, k, "t-route-one")
	k.HandleContentUpdate(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	assert.True(t, k.selected["t-route-one"])
}

// ensure the form submit message type is wired (compile-time reference).
var _ = form.FormSubmitMsg{}
