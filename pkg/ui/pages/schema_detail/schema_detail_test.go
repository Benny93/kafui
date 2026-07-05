package schemadetail

import (
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/components/editor"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyDS wraps the full mock datasource and records the schema mutation calls.
type spyDS struct {
	*mock.KafkaDataSourceMock
	callLog       []string
	checkCalls    int
	registerCalls int
	delSubject    []string
	delVersion    [][2]int // {version, permanentFlag(0/1)}
	setSubject    []api.CompatibilityLevel
	topicNames    []string
}

func newSpy() *spyDS {
	m := &mock.KafkaDataSourceMock{}
	m.Init("")
	return &spyDS{KafkaDataSourceMock: m}
}

func (s *spyDS) CheckSchemaCompatibility(subject, text, typ string) (bool, []string, error) {
	s.callLog = append(s.callLog, "check")
	s.checkCalls++
	return s.KafkaDataSourceMock.CheckSchemaCompatibility(subject, text, typ)
}

func (s *spyDS) RegisterSchema(subject, text, typ string) (api.Schema, error) {
	s.callLog = append(s.callLog, "register")
	s.registerCalls++
	return s.KafkaDataSourceMock.RegisterSchema(subject, text, typ)
}

func (s *spyDS) DeleteSubject(subject string, permanent bool) ([]int, error) {
	s.delSubject = append(s.delSubject, subject)
	return s.KafkaDataSourceMock.DeleteSubject(subject, permanent)
}

func (s *spyDS) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	p := 0
	if permanent {
		p = 1
	}
	s.delVersion = append(s.delVersion, [2]int{version, p})
	return s.KafkaDataSourceMock.DeleteSchemaVersion(subject, version, permanent)
}

func (s *spyDS) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	s.setSubject = append(s.setSubject, level)
	return s.KafkaDataSourceMock.SetSubjectCompatibility(subject, level)
}

func (s *spyDS) GetTopicNames() ([]string, error) {
	if s.topicNames != nil {
		return s.topicNames, nil
	}
	return s.KafkaDataSourceMock.GetTopicNames()
}

// ─── helpers ───────────────────────────────────────────────────────────────

func newTestModel(ds api.KafkaDataSource, subject, schemaType string) *Model {
	return &Model{
		dataSource:   ds,
		subject:      subject,
		schemaType:   schemaType,
		spinner:      spinner.New(),
		viewer:       editor.NewViewer(""),
		contentCache: map[int]string{},
	}
}

func run(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// ─── SR-11: version browsing ─────────────────────────────────────────────────

func TestVersionListLoadsAndRenders(t *testing.T) {
	m := newTestModel(newSpy(), "orders-value", "AVRO")

	cmd := m.enterVersions()
	msg, ok := run(cmd).(SchemaVersionsLoadedMsg)
	require.True(t, ok)
	require.NoError(t, msg.Err)
	require.Len(t, msg.Versions, 3)

	m.versions = msg.Versions
	m.versionsLoaded = true

	out := renderVersionList(m, 80, 20)
	assert.Contains(t, out, "Versions of orders-value (3)")
	assert.Contains(t, out, "v3")
	assert.Contains(t, out, "(latest)")

	// Newest-first ordering: cursor 0 is the latest (v3).
	assert.Equal(t, 3, m.SelectedVersion())
}

func TestSelectVersionReloadsContent(t *testing.T) {
	m := newTestModel(newSpy(), "orders-value", "AVRO")
	m.versions, _ = m.dataSource.GetSchemaVersions("orders-value")
	m.versionsLoaded = true

	// Move cursor to the oldest (v1) — newest-first, so index 2.
	require.Nil(t, handleVersionsKey(m, tea.KeyMsg{Type: tea.KeyDown}))
	require.Nil(t, handleVersionsKey(m, tea.KeyMsg{Type: tea.KeyDown}))
	assert.Equal(t, 1, m.SelectedVersion())

	cmd := handleVersionsKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, modeContent, m.mode)
	assert.Equal(t, 1, m.version)

	msg, ok := run(cmd).(SchemaContentLoadedMsg)
	require.True(t, ok)
	assert.Equal(t, 1, msg.Version)
	assert.Contains(t, msg.Content, "OrderCreatedEvent")
}

// ─── SR-12: diff ─────────────────────────────────────────────────────────────

func TestDiffOfTwoVersions(t *testing.T) {
	m := newTestModel(newSpy(), "orders-value", "AVRO")
	m.versions, _ = m.dataSource.GetSchemaVersions("orders-value")
	m.versionsLoaded = true

	cmd := m.enterDiff(1, 3)
	assert.Equal(t, modeDiff, m.mode)
	assert.Equal(t, 1, m.diffLeft)
	assert.Equal(t, 3, m.diffRight)

	// Executing the batched load commands yields both versions' content.
	msgs := drainBatch(cmd)
	got := 0
	page := &SchemaDetailPageModel{model: m}
	for _, msg := range msgs {
		if _, ok := msg.(schemaVersionContentMsg); ok {
			page.Update(msg)
			got++
		}
	}
	assert.Equal(t, 2, got)
	assert.Contains(t, m.contentCache, 1)
	assert.Contains(t, m.contentCache, 3)

	// The rendered diff shows v3's added "createdAt" field as an addition.
	d := editor.Diff(prettySchema(m.contentCache[1], "AVRO"), prettySchema(m.contentCache[3], "AVRO"))
	assert.Contains(t, d, "createdAt")
	assert.NotEmpty(t, renderDiff(m, 80, 20))
}

func TestDiffVersionCycling(t *testing.T) {
	m := newTestModel(newSpy(), "orders-value", "AVRO")
	m.versions, _ = m.dataSource.GetSchemaVersions("orders-value")
	m.versionsLoaded = true
	m.enterDiff(2, 3)

	// Active pane defaults to left (v2). Cycle it down to v1.
	m.cycleDiffVersion(-1)
	assert.Equal(t, 1, m.diffLeft)

	// Switch to the right pane and cycle it — right stays within bounds.
	handleDiffKey(m, keyRunes("l"))
	assert.Equal(t, 1, m.diffActive)
	m.cycleDiffVersion(-1)
	assert.Equal(t, 2, m.diffRight)
}

// ─── SR-16/17: register + compatibility check ───────────────────────────────

func TestRegisterChecksThenRegistersWhenCompatible(t *testing.T) {
	spy := newSpy()
	m := newTestModel(spy, "orders-value", "AVRO")
	m.content = `{"old":true}`
	m.enterRegister()
	m.editor.SetValue(`{"new":true}`)

	cmd := handleRegisterKey(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	msg, ok := run(cmd).(SchemaRegisterResultMsg)
	require.True(t, ok)

	assert.Equal(t, []string{"check", "register"}, spy.callLog)
	assert.False(t, msg.Incompatible)
	assert.NoError(t, msg.Err)
	assert.Equal(t, 4, msg.Schema.Version) // orders-value had 3 versions
}

func TestRegisterStopsAndReportsWhenIncompatible(t *testing.T) {
	spy := newSpy()
	m := newTestModel(spy, "orders-value", "AVRO")
	m.content = `{"old":true}`
	m.enterRegister()
	m.editor.SetValue(`{"INCOMPATIBLE":true}`)

	cmd := handleRegisterKey(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	msg, ok := run(cmd).(SchemaRegisterResultMsg)
	require.True(t, ok)

	// Check ran; register never did.
	assert.Equal(t, []string{"check"}, spy.callLog)
	assert.Equal(t, 0, spy.registerCalls)
	assert.True(t, msg.Incompatible)
	assert.NotEmpty(t, msg.Messages)
}

func TestValidateRegister(t *testing.T) {
	m := newTestModel(newSpy(), "orders-value", "AVRO")
	m.registerSeed = `{"a":1}`
	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{"empty", "  ", true},
		{"unchanged", `{"a":1}`, true},
		{"invalid json", `{not json`, true},
		{"valid changed", `{"a":2}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := m.validateRegister(tt.text)
			assert.Equal(t, tt.wantErr, reason != "")
		})
	}
}

func TestCheckOnlyNeverRegisters(t *testing.T) {
	spy := newSpy()
	m := newTestModel(spy, "orders-value", "AVRO")
	m.content = `{"INCOMPATIBLE":true}`

	cmd := m.checkOnlyCmd()
	msg, ok := run(cmd).(SchemaCheckResultMsg)
	require.True(t, ok)
	assert.False(t, msg.Compatible)
	assert.Equal(t, 0, spy.registerCalls)
	assert.Equal(t, 1, spy.checkCalls)
}

// ─── SR-18: delete subject/version ───────────────────────────────────────────

func TestDeleteSubjectConfirmationAndCall(t *testing.T) {
	spy := newSpy()
	m := newTestModel(spy, "orders-value", "AVRO")

	confirm, ok := run(m.confirmDeleteSubjectCmd()).(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.True(t, confirm.Danger)
	assert.Contains(t, confirm.Message, "orders-value")

	// Nothing happens until the user confirms.
	assert.Empty(t, spy.delSubject)

	res, ok := confirm.OnConfirm().(SchemaDeleteResultMsg)
	require.True(t, ok)
	assert.NoError(t, res.Err)
	assert.True(t, res.BackToList)
	assert.Equal(t, []string{"orders-value"}, spy.delSubject)
}

func TestDeleteVersionConfirmationAndCall(t *testing.T) {
	spy := newSpy()
	m := newTestModel(spy, "orders-value", "AVRO")

	confirm, ok := run(m.confirmDeleteVersionCmd(2)).(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.Empty(t, spy.delVersion)

	res, ok := confirm.OnConfirm().(SchemaDeleteResultMsg)
	require.True(t, ok)
	assert.NoError(t, res.Err)
	require.Len(t, spy.delVersion, 1)
	assert.Equal(t, 2, spy.delVersion[0][0]) // version 2
	assert.Equal(t, 0, spy.delVersion[0][1]) // soft delete
}

// ─── SR-19: compatibility picker ─────────────────────────────────────────────

func TestCompatibilityPickerSetsSubjectLevel(t *testing.T) {
	spy := newSpy()
	m := newTestModel(spy, "orders-value", "AVRO")
	m.compat = api.CompatibilityBackward
	m.enterPicker()
	assert.Equal(t, modePicker, m.mode)

	// Move to FULL and confirm.
	target := api.CompatibilityFull
	for m.SelectedLevel() != target {
		handlePickerKey(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	cmd := handlePickerKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	confirm, ok := run(cmd).(core.ShowConfirmMsg)
	require.True(t, ok)
	assert.Empty(t, spy.setSubject)

	res, ok := confirm.OnConfirm().(SchemaCompatSetResultMsg)
	require.True(t, ok)
	assert.NoError(t, res.Err)
	assert.Equal(t, []api.CompatibilityLevel{target}, spy.setSubject)
}

// ─── SR-13/14: sidebar metadata ──────────────────────────────────────────────

func TestSidebarShowsCompatibilityAndTopic(t *testing.T) {
	m := newTestModel(newSpy(), "orders-value", "AVRO")
	m.compat = api.CompatibilityBackward
	m.compatSpecific = false
	m.compatLoaded = true
	m.topic = "orders"

	section := NewSchemaMetadataSidebarSection(m)
	items := section.RenderItems(10, 30)

	var compat, topic string
	for _, it := range items {
		switch it.Text {
		case "Compatibility":
			compat = it.Value
		case "Topic":
			topic = it.Value
		}
	}
	assert.Equal(t, "BACKWARD (global)", compat)
	assert.Contains(t, topic, "orders")
}

func TestResolveTopic(t *testing.T) {
	spy := newSpy()
	spy.topicNames = []string{"orders", "payments"}
	tests := []struct {
		subject string
		want    string
	}{
		{"orders-value", "orders"},
		{"payments-value", "payments"},
		{"orders-key", "orders"},
		{"unknown-value", ""},
		{"no-suffix", ""},
	}
	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveTopic(tt.subject, spy))
		})
	}
}

// ─── SR-22: not-configured empty state ───────────────────────────────────────

func TestNotConfiguredEmptyState(t *testing.T) {
	msg := friendlySchemaError(api.SchemaRegistryNotConfiguredError{})
	assert.Equal(t, "No schema registry is configured for this cluster.", msg)

	// Surfaced through the page's content-load handler without erroring out.
	m := newTestModel(newSpy(), "orders-value", "AVRO")
	m.loading = true
	page := &SchemaDetailPageModel{model: m}
	page.Update(SchemaContentLoadedMsg{Err: api.SchemaRegistryNotConfiguredError{}})
	assert.False(t, m.loading)
	assert.Contains(t, m.content, "No schema registry is configured")
}

// drainBatch executes a (possibly batched) command and returns all produced
// messages, flattening tea.BatchMsg one level.
func drainBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, drainBatch(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}
